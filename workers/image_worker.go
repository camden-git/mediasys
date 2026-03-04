package workers

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/disintegration/imaging"
	"gocv.io/x/gocv"
)

// TaskType constants
const (
	TaskThumbnail = "thumbnail"
	TaskMetadata  = "metadata"
	TaskDetection = "detection"
	TaskAlbumZip  = "album_zip"
	TaskPreview   = "preview"
)

type ImageJob struct {
	OriginalImagePath    string
	OriginalRelativePath string
	ModTimeUnix          int64
	TaskType             string
	AlbumID              int64
}

type ImageProcessor struct {
	JobQueue          chan ImageJob
	DetectionJobQueue chan ImageJob
	Config            config.Config
	ImageRepo         repository.ImageRepositoryInterface
	AlbumRepo         repository.AlbumRepositoryInterface
	FaceRepo          repository.FaceRepositoryInterface
	TagRepo           repository.ImageTagRepositoryInterface
	Wg                sync.WaitGroup
	StopChan          chan struct{}
	Pending           map[string]bool
	Mutex             sync.Mutex
	Hub               *realtime.Hub
}

func NewImageProcessor(
	cfg config.Config,
	imgRepo repository.ImageRepositoryInterface,
	albumRepo repository.AlbumRepositoryInterface,
	faceRepo repository.FaceRepositoryInterface,
	tagRepo repository.ImageTagRepositoryInterface,
	queueSize, numWorkers int,
	numDetectionWorkers, detectionQueueSize int,
	hub *realtime.Hub,
) *ImageProcessor {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 50
	}
	if numDetectionWorkers <= 0 {
		numDetectionWorkers = 1
	}
	if detectionQueueSize <= 0 {
		detectionQueueSize = 10
	}
	proc := &ImageProcessor{
		JobQueue:          make(chan ImageJob, queueSize),
		DetectionJobQueue: make(chan ImageJob, detectionQueueSize),
		Config:            cfg,
		ImageRepo:         imgRepo,
		AlbumRepo:         albumRepo,
		FaceRepo:          faceRepo,
		TagRepo:           tagRepo,
		StopChan:          make(chan struct{}),
		Pending:           make(map[string]bool),
		Hub:               hub,
	}
	proc.Wg.Add(numWorkers + numDetectionWorkers)
	for i := 0; i < numWorkers; i++ {
		go proc.worker(i, cfg)
	}
	for i := 0; i < numDetectionWorkers; i++ {
		go proc.detectionWorker(i, cfg)
	}
	log.Printf("Started %d image processing worker(s) with queue size %d", numWorkers, queueSize)
	log.Printf("Started %d detection worker(s) with queue size %d", numDetectionWorkers, detectionQueueSize)
	return proc
}

// worker handles thumbnail, preview, metadata, and zip tasks. It does NOT load
// ML models, keeping its memory footprint small.
func (ip *ImageProcessor) worker(id int, cfg config.Config) {
	defer ip.Wg.Done()

	mediaStore, err := media.NewLocalStorage(cfg.MediaStoragePath, map[media.AssetType]string{
		media.AssetTypeThumbnail: filepath.Base(cfg.ThumbnailsPath),
		media.AssetTypeBanner:    filepath.Base(cfg.BannersPath),
		media.AssetTypeArchive:   filepath.Base(cfg.ArchivesPath),
		media.AssetTypePreview:   filepath.Base(cfg.PreviewsPath),
	})
	if err != nil {
		log.Printf("Worker %d: FATAL - Failed to initialize media store: %v. Worker exiting.", id, err)
		return
	}
	mediaProcessor := media.NewProcessor(mediaStore)

	log.Printf("Image worker %d started", id)
	for {
		select {
		case job, ok := <-ip.JobQueue:
			if !ok {
				log.Printf("Image worker %d stopping: Job queue closed", id)
				return
			}
			ip.processJob(id, job, mediaProcessor, mediaStore, nil, nil, nil, cfg)

		case <-ip.StopChan:
			log.Printf("Image worker %d stopping: Stop signal received", id)
			return
		}
	}
}

// detectionWorker loads ML models once and handles only TaskDetection jobs.
// Keeping detection in a separate, smaller pool avoids duplicating large model
// weights across every general worker.
func (ip *ImageProcessor) detectionWorker(id int, cfg config.Config) {
	defer ip.Wg.Done()

	var faceDetector *media.DNNFaceDetector
	var retinaFaceDetector *media.RetinaFaceDetector
	var recognitionModel *media.FaceRecognitionModel

	log.Printf("Detection worker %d: FACE_RECOGNITION_ENABLED config value: %v", id, cfg.FaceRecognitionEnabled)
	if cfg.FaceRecognitionEnabled {
		log.Printf("Detection worker %d: Loading face detectors...", id)

		faceDetector = media.NewDNNFaceDetector(cfg.FaceDNNNetConfigPath, cfg.FaceDNNNetModelPath)
		defer func() {
			if faceDetector != nil {
				faceDetector.Close()
			}
		}()
		if faceDetector == nil || !faceDetector.Enabled {
			log.Printf("Detection worker %d: DNN Face Detector disabled.", id)
		}

		retinaFaceDetector = media.NewRetinaFaceDetector(cfg.RetinaFaceModelPath)
		defer func() {
			if retinaFaceDetector != nil {
				retinaFaceDetector.Close()
			}
		}()
		if retinaFaceDetector == nil || !retinaFaceDetector.Enabled {
			log.Printf("Detection worker %d: RetinaFace Detector disabled.", id)
		}

		log.Printf("Detection worker %d: Initializing face recognition model...", id)
		recognitionModel = media.NewFaceRecognitionModel(cfg.FaceRecognitionModelPath, cfg.FaceRecognitionModelName)
		defer func() {
			if recognitionModel != nil && recognitionModel.Enabled {
				recognitionModel.Close()
			}
		}()
		if recognitionModel == nil || !recognitionModel.Enabled {
			log.Printf("Detection worker %d: Face Recognition Model disabled or failed to load.", id)
		} else {
			log.Printf("Detection worker %d: Face Recognition Model enabled (%s).", id, cfg.FaceRecognitionModelName)
		}
	} else {
		log.Printf("Detection worker %d: face detection fully DISABLED — skipping all model loads.", id)
	}

	log.Printf("Detection worker %d started", id)
	for {
		select {
		case job, ok := <-ip.DetectionJobQueue:
			if !ok {
				log.Printf("Detection worker %d stopping: Job queue closed", id)
				return
			}
			ip.processJob(id, job, nil, nil, faceDetector, retinaFaceDetector, recognitionModel, cfg)

		case <-ip.StopChan:
			log.Printf("Detection worker %d stopping: Stop signal received", id)
			return
		}
	}
}

// processJob is shared logic for both worker types. mediaProcessor/mediaStore may be nil
// for detection workers; faceDetector/retinaFaceDetector/recognitionModel may be nil for
// general workers.
func (ip *ImageProcessor) processJob(
	id int,
	job ImageJob,
	mediaProcessor *media.Processor,
	mediaStore media.Store,
	faceDetector *media.DNNFaceDetector,
	retinaFaceDetector *media.RetinaFaceDetector,
	recognitionModel *media.FaceRecognitionModel,
	cfg config.Config,
) {
	var err error
	var pendingKey string
	var statusColumn string
	var entityPath string

	log.Printf("Worker %d: Received job type '%s' for: %s", id, job.TaskType, entityPath)
	if ip.Hub != nil {
		ip.Hub.Broadcast(realtime.Event{
			Type:      "task",
			Path:      job.OriginalRelativePath,
			Task:      job.TaskType,
			Status:    "processing",
			Timestamp: time.Now().Unix(),
		})
	}

	if job.TaskType == TaskAlbumZip {
		err = ip.AlbumRepo.MarkZipProcessing(uint(job.AlbumID))
		statusColumn = "zip_status"
		entityPath = fmt.Sprintf("album ID %d", job.AlbumID)
		pendingKey = fmt.Sprintf("album_%d:%s", job.AlbumID, job.TaskType)
	} else {
		statusColumn = job.TaskType + "_status"
		err = ip.ImageRepo.MarkTaskProcessing(job.OriginalRelativePath, statusColumn)
		log.Printf("Status column: %s", statusColumn)
		entityPath = job.OriginalRelativePath
		pendingKey = fmt.Sprintf("%s:%s", job.OriginalRelativePath, job.TaskType)
	}

	if err != nil {
		log.Printf("Worker %d: ERROR marking %s processing for %s: %v. Skipping job.", id, job.TaskType, entityPath, err)
		if ip.Hub != nil {
			ip.Hub.Broadcast(realtime.Event{Type: "task", Path: job.OriginalRelativePath, Task: job.TaskType, Status: "error", Error: err.Error(), Timestamp: time.Now().Unix()})
		}
		ip.Mutex.Lock()
		delete(ip.Pending, pendingKey)
		ip.Mutex.Unlock()
		return
	}

	switch job.TaskType {
	case TaskThumbnail:
		ip.processThumbnailTask(job, mediaProcessor)
	case TaskMetadata:
		ip.processMetadataTask(job)
	case TaskDetection:
		ip.processDetectionTask(job, faceDetector, retinaFaceDetector, recognitionModel, cfg)
	case TaskAlbumZip:
		ip.processAlbumZipTask(job, mediaStore)
	case TaskPreview:
		ip.processPreviewTask(job, mediaProcessor)
	default:
		log.Printf("Worker %d: ERROR unknown task type '%s'", id, job.TaskType)
	}

	if ip.Hub != nil {
		ip.Hub.Broadcast(realtime.Event{
			Type:      "task",
			Path:      job.OriginalRelativePath,
			Task:      job.TaskType,
			Status:    "done",
			Timestamp: time.Now().Unix(),
		})
	}

	ip.Mutex.Lock()
	delete(ip.Pending, pendingKey)
	ip.Mutex.Unlock()
}

// processThumbnailTask generates thumbnail and updates DB
func (ip *ImageProcessor) processThumbnailTask(job ImageJob, processor *media.Processor) {
	var taskErr error
	var thumbRelPath *string

	file, openErr := os.Open(job.OriginalImagePath)
	if openErr != nil {
		taskErr = fmt.Errorf("failed to open original file: %w", openErr)
		log.Printf("Worker: Skipping thumbnail task for %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		img, format, decodeErr := image.Decode(file)
		file.Close()

		if decodeErr != nil {
			taskErr = fmt.Errorf("failed to decode image for thumbnail: %w", decodeErr)
			log.Printf("Worker: ERROR %v for %s", taskErr, job.OriginalRelativePath)
		} else {
			log.Printf("Worker: Decoded image %s (format: %s) for thumbnail", job.OriginalRelativePath, format)
			relPath, genErr := processor.GenerateThumbnail(img, job.OriginalRelativePath, ip.Config.ThumbnailMaxSize)
			if genErr != nil {
				taskErr = fmt.Errorf("thumbnail generation/save failed: %w", genErr)
				log.Printf("Worker: ERROR %v for %s", taskErr, job.OriginalRelativePath)
			} else {
				thumbRelPath = &relPath
				log.Printf("Worker: Generated thumbnail for %s", job.OriginalRelativePath)
			}
		}
	}

	dbErr := ip.ImageRepo.UpdateThumbnailResult(job.OriginalRelativePath, thumbRelPath, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating thumbnail DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

func (ip *ImageProcessor) processMetadataTask(job ImageJob) {
	var taskErr error
	var metadata *media.Metadata

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping metadata task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for metadata task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		metadata, taskErr = media.GetImageMetadata(job.OriginalImagePath)
		if taskErr != nil {
			log.Printf("Worker: ERROR extracting metadata for %s: %v", job.OriginalRelativePath, taskErr)
		} else {
			log.Printf("Worker: Extracted metadata for %s", job.OriginalRelativePath)
		}
	}

	dbErr := ip.ImageRepo.UpdateMetadataResult(job.OriginalRelativePath, metadata, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating metadata DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}

	// Update XMP-derived tags if metadata was extracted successfully
	if taskErr == nil && metadata != nil && ip.TagRepo != nil {
		xmpTags := buildXMPTags(job.OriginalRelativePath, metadata.Keywords)
		if tagErr := ip.TagRepo.SetXMPTags(job.OriginalRelativePath, xmpTags); tagErr != nil {
			log.Printf("Worker: ERROR updating XMP tags for %s: %v", job.OriginalRelativePath, tagErr)
		}
	}
}

// buildXMPTags converts raw XMP keyword strings into ImageTag records.
// Keywords containing "/" are treated as hierarchical "key/value" pairs.
// Others use tag_key="keyword" with the raw string as tag_value.
func buildXMPTags(imagePath string, keywords []string) []models.ImageTag {
	tags := make([]models.ImageTag, 0, len(keywords))
	for _, kw := range keywords {
		var key, val string
		if idx := strings.Index(kw, "/"); idx > 0 {
			key = strings.TrimSpace(kw[:idx])
			val = strings.TrimSpace(kw[idx+1:])
		} else {
			key = "keyword"
			val = strings.TrimSpace(kw)
		}
		if key == "" || val == "" {
			continue
		}
		tags = append(tags, models.ImageTag{
			ImagePath: imagePath,
			TagKey:    key,
			TagValue:  val,
			Source:    "xmp",
		})
	}
	return tags
}

// processDetectionTask performs detection and updates DB
func (ip *ImageProcessor) processDetectionTask(job ImageJob, faceDetector *media.DNNFaceDetector, retinaFaceDetector *media.RetinaFaceDetector, recognitionModel *media.FaceRecognitionModel, cfg config.Config) {
	var taskErr error
	var detections []media.DetectionResult

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping detection task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for detection task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		// Try RetinaFace first (preferred), fall back to DNN if needed
		if retinaFaceDetector != nil && retinaFaceDetector.Enabled {
			img := gocv.IMRead(job.OriginalImagePath, gocv.IMReadColor)
			if img.Empty() {
				taskErr = fmt.Errorf("failed to read image file for RetinaFace: %s", job.OriginalImagePath)
			} else {
				defer img.Close()

				// Use RetinaFace with face recognition if available
				if recognitionModel != nil && recognitionModel.Enabled {
					log.Printf("Worker: Using RetinaFace WITH face recognition for %s", job.OriginalRelativePath)
					detections = retinaFaceDetector.DetectFacesAndExtractEmbeddings(img, recognitionModel)
				} else {
					log.Printf("Worker: Using RetinaFace WITHOUT face recognition for %s (recognitionModel: %v)", job.OriginalRelativePath, recognitionModel != nil)
					detections = retinaFaceDetector.DetectFaces(img)
				}

				log.Printf("Worker: RetinaFace detection complete for %s: Found %d faces.", job.OriginalRelativePath, len(detections))
			}
		} else if faceDetector != nil && faceDetector.Enabled {
			// Fall back to DNN detector
			detections, taskErr = media.DetectFacesAndAnimals(job.OriginalImagePath, faceDetector)
			if taskErr != nil {
				log.Printf("Worker: ERROR during DNN detection for %s: %v", job.OriginalRelativePath, taskErr)
			} else {
				log.Printf("Worker: DNN detection complete for %s: Found %d objects.", job.OriginalRelativePath, len(detections))
			}
		} else {
			taskErr = fmt.Errorf("no face detector enabled or loaded")
			log.Printf("Worker: Skipping detection for %s: no detector available", job.OriginalRelativePath)
		}
	}

	dbErr := ip.ImageRepo.UpdateDetectionResult(job.OriginalRelativePath, detections, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating detection DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

func (ip *ImageProcessor) processAlbumZipTask(job ImageJob, store media.Store) {
	log.Printf("Worker: Starting ZIP task for Album ID: %d", job.AlbumID)
	var taskErr error
	var finalZipRelPath *string
	var finalZipSize *int64

	album, err := ip.AlbumRepo.GetByID(uint(job.AlbumID))
	if err != nil {
		taskErr = fmt.Errorf("failed to fetch album details for ID %d: %w", job.AlbumID, err)
		log.Printf("Worker: ERROR %v", taskErr)
	} else {
		//zipSaveDirName := filepath.Base(ip.Config.ArchivesPath)
		zipSaveDirAbs := ip.Config.ArchivesPath // full path to archives directory

		// ensure album.Slug is safe for filenames
		safeSlug := strings.ReplaceAll(album.Slug, "/", "_")
		safeSlug = strings.ReplaceAll(safeSlug, "\\", "_")

		zipFilenameBase := fmt.Sprintf("album_%s_%d_archive_%d", safeSlug, album.ID, time.Now().Unix())

		savedZipFilename, zipSizeBytes, zipErr := CreateAlbumZip(
			ip.Config.RootDirectory, // root of all media folders
			album.FolderPath,        // path relative to RootDirectory
			zipSaveDirAbs,           // absolute path to save the zip
			zipFilenameBase,         // filename base for the zip
		)

		if zipErr != nil {
			taskErr = fmt.Errorf("failed to create album zip for %s: %w", album.FolderPath, zipErr)
			log.Printf("Worker: ERROR %v", taskErr)
		} else {
			// relativePathToStore should be relative to the MediaStoragePath root
			// example: if MediaStoragePath is /srv/media and zipSaveDirAbs is /srv/media/archives,
			// then relativePathToStore should be "archives/the_zip_file.zip"
			relativePathToStore, relErr := filepath.Rel(ip.Config.MediaStoragePath, filepath.Join(zipSaveDirAbs, savedZipFilename))
			if relErr != nil {
				taskErr = fmt.Errorf("failed to calculate relative path for zip: %w", relErr)
				log.Printf("Worker: ERROR %v", taskErr)
			} else {
				slashPath := filepath.ToSlash(relativePathToStore)
				finalZipRelPath = &slashPath
				finalZipSize = &zipSizeBytes
				log.Printf("Worker: Successfully created ZIP for Album ID %d: %s", job.AlbumID, slashPath)
			}
		}
	}

	dbErr := ip.AlbumRepo.SetZipResult(uint(job.AlbumID), finalZipRelPath, finalZipSize, taskErr) // Use AlbumRepo
	if dbErr != nil {
		log.Printf("Worker: ERROR updating album ZIP DB result for Album ID %d: %v", job.AlbumID, dbErr)
		if finalZipRelPath != nil && store != nil { // Ensure store is not nil
			fullPathToClean, _ := store.GetFullPath(*finalZipRelPath)
			if fullPathToClean != "" {
				if err := os.Remove(fullPathToClean); err != nil {
					log.Printf("Worker: Failed to remove zip file %s after DB error: %v", fullPathToClean, err)
				}
			}
		}
	}
}

// processPreviewTask generates a scaled preview and updates DB
func (ip *ImageProcessor) processPreviewTask(job ImageJob, processor *media.Processor) {
	var taskErr error
	var previewRelPath *string

	src, openErr := imaging.Open(job.OriginalImagePath, imaging.AutoOrientation(true))
	if openErr != nil {
		taskErr = fmt.Errorf("failed to open original file for preview: %w", openErr)
		log.Printf("Worker: Skipping preview task for %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		relPath, genErr := processor.GeneratePreview(src, job.OriginalRelativePath)
		if genErr != nil {
			taskErr = fmt.Errorf("preview generation/save failed: %w", genErr)
			log.Printf("Worker: ERROR %v for %s", taskErr, job.OriginalRelativePath)
		} else {
			previewRelPath = &relPath
			log.Printf("Worker: Generated preview for %s", job.OriginalRelativePath)
		}
	}

	dbErr := ip.ImageRepo.UpdatePreviewResult(job.OriginalRelativePath, previewRelPath, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating preview DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

// StartMemoryTrimmer starts a background goroutine that calls runtime.GC() and
// debug.FreeOSMemory() on the given interval to return freed heap pages to the OS
// **Set intervalMinutes <= 0 to disable**
func (ip *ImageProcessor) StartMemoryTrimmer(intervalMinutes int) {
	if intervalMinutes <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				runtime.GC()
				debug.FreeOSMemory()
				runtime.ReadMemStats(&after)
				beforeMB := before.HeapSys / (1 << 20)
				afterMB := after.HeapSys / (1 << 20)
				releasedMB := int64(beforeMB) - int64(afterMB)
				log.Printf("Memory trim: released %d MB to OS (HeapSys before=%d MB, after=%d MB)",
					releasedMB, beforeMB, afterMB)
			case <-ip.StopChan:
				return
			}
		}
	}()
	log.Printf("Memory trimmer started (interval: %d minutes)", intervalMinutes)
}

// StartPreviewCleanup starts a background goroutine that evicts stale previews every 6 hours
func (ip *ImageProcessor) StartPreviewCleanup() {
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ip.cleanupStalePreviewFiles()
			case <-ip.StopChan:
				return
			}
		}
	}()
	log.Println("Preview cleanup goroutine started (runs every 6 hours, TTL=7 days)")
}

// cleanupStalePreviewFiles removes preview files not accessed in the last 7 days.
// Images are fetched in batches of 100 to avoid a large transient heap allocation
// on libraries with many stale previews. Since each batch resets preview_status, the
// next iteration always fetches from offset 0 (removed rows don't reappear).
func (ip *ImageProcessor) cleanupStalePreviewFiles() {
	cutoffUnix := time.Now().Add(-7 * 24 * time.Hour).Unix()
	const batchSize = 100
	totalEvicted := 0
	for {
		images, err := ip.ImageRepo.GetStalePreviewImagesBatch(cutoffUnix, 0, batchSize)
		if err != nil {
			log.Printf("Preview cleanup: ERROR fetching stale previews: %v", err)
			return
		}
		if len(images) == 0 {
			break
		}
		for _, img := range images {
			if img.PreviewPath == nil {
				continue
			}
			fullPath := filepath.Join(ip.Config.MediaStoragePath, *img.PreviewPath)
			if removeErr := os.Remove(fullPath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Printf("Preview cleanup: ERROR removing %s: %v", fullPath, removeErr)
			}
			resetErr := ip.ImageRepo.UpdatePreviewResult(img.OriginalPath, nil, img.LastModified, nil)
			if resetErr != nil {
				log.Printf("Preview cleanup: ERROR resetting preview status for %s: %v", img.OriginalPath, resetErr)
			} else {
				log.Printf("Preview cleanup: Evicted preview for %s", img.OriginalPath)
				totalEvicted++
			}
		}
		if len(images) < batchSize {
			break
		}
	}
	log.Printf("Preview cleanup: Evicted %d preview(s) total", totalEvicted)
}

// QueueJob queues a specific task if not already pending. TaskDetection jobs
// are routed to the dedicated detection worker queue; all other jobs go to the
// general worker queue.
func (ip *ImageProcessor) QueueJob(job ImageJob) bool {
	if job.TaskType == TaskDetection && !ip.Config.FaceRecognitionEnabled {
		return false
	}

	var pendingKey string
	if job.TaskType == TaskAlbumZip {
		pendingKey = fmt.Sprintf("album_%d:%s", job.AlbumID, job.TaskType)
	} else {
		pendingKey = fmt.Sprintf("%s:%s", job.OriginalRelativePath, job.TaskType)
	}

	ip.Mutex.Lock()
	if ip.Pending[pendingKey] {
		ip.Mutex.Unlock()
		return false
	}
	ip.Pending[pendingKey] = true
	ip.Mutex.Unlock()

	targetQueue := ip.JobQueue
	if job.TaskType == TaskDetection {
		targetQueue = ip.DetectionJobQueue
	}

	select {
	case targetQueue <- job:
		log.Printf("Queued task '%s' for: %s", job.TaskType, job.OriginalRelativePath)
		return true
	default:
		log.Printf("WARNING: Job queue full. Failed to queue task '%s' for: %s", job.TaskType, job.OriginalRelativePath)
		ip.Mutex.Lock()
		delete(ip.Pending, pendingKey)
		ip.Mutex.Unlock()
		return false
	}
}

func (ip *ImageProcessor) Stop() {
	log.Println("Stopping image processor workers...")
	close(ip.StopChan)
	ip.Wg.Wait()
	log.Println("All image processor workers stopped")
}

package workers

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/utils"
)

// TaskType constants
const (
	TaskThumbnail = "thumbnail"
	TaskMetadata  = "metadata"
	TaskDetection = "detection"
	TaskAlbumZip  = "album_zip"
)

type ImageJob struct {
	OriginalImagePath    string
	OriginalRelativePath string
	ModTimeUnix          int64
	TaskType             string
	AlbumID              int64
}

type ImageProcessor struct {
	JobQueue  chan ImageJob
	Config    config.Config
	ImageRepo repository.ImageRepositoryInterface
	AlbumRepo repository.AlbumRepositoryInterface
	FaceRepo  repository.FaceRepositoryInterface
	Wg        sync.WaitGroup
	StopChan  chan struct{}
	Pending   map[string]bool
	Mutex     sync.Mutex
}

func NewImageProcessor(
	cfg config.Config,
	imgRepo repository.ImageRepositoryInterface,
	albumRepo repository.AlbumRepositoryInterface,
	faceRepo repository.FaceRepositoryInterface,
	queueSize, numWorkers int,
) *ImageProcessor {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}
	proc := &ImageProcessor{
		JobQueue:  make(chan ImageJob, queueSize),
		Config:    cfg,
		ImageRepo: imgRepo,
		AlbumRepo: albumRepo,
		FaceRepo:  faceRepo,
		StopChan:  make(chan struct{}),
		Pending:   make(map[string]bool),
	}
	proc.Wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go proc.worker(i, cfg)
	}
	log.Printf("Started %d image processing worker(s) with queue size %d", numWorkers, queueSize)
	return proc
}

// worker loads resources and processes jobs from the queue
func (ip *ImageProcessor) worker(id int, cfg config.Config) {
	defer ip.Wg.Done()

	mediaStore, err := media.NewLocalStorage(cfg.MediaStoragePath, map[media.AssetType]string{
		media.AssetTypeThumbnail: filepath.Base(cfg.ThumbnailsPath),
		media.AssetTypeBanner:    filepath.Base(cfg.BannersPath),
		media.AssetTypeArchive:   filepath.Base(cfg.ArchivesPath),
	})
	if err != nil {
		log.Printf("Worker %d: FATAL - Failed to initialize media store: %v. Worker exiting.", id, err)
		return
	}
	mediaProcessor := media.NewProcessor(mediaStore)

	log.Printf("Worker %d: Loading DNN face detector...", id)
	faceDetector := media.NewDNNFaceDetector(cfg.FaceDNNNetConfigPath, cfg.FaceDNNNetModelPath)
	defer func() {
		if faceDetector != nil {
			faceDetector.Close()
		}
	}()
	if faceDetector == nil || !faceDetector.Enabled {
		log.Printf("Worker %d: DNN Face Detector disabled.", id)
	}

	log.Printf("Image worker %d started", id)
	for {
		select {
		case job, ok := <-ip.JobQueue:
			if !ok {
				log.Printf("Image worker %d stopping: Job queue closed", id)
				return
			}

			var err error

			var pendingKey string
			var statusColumn string
			var entityPath string

			log.Printf("Worker %d: Received job type '%s' for: %s", id, job.TaskType, entityPath)

			if job.TaskType == TaskAlbumZip {
				err = ip.AlbumRepo.MarkZipProcessing(uint(job.AlbumID))
				statusColumn = "zip_status" // for logging key
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
				ip.Mutex.Lock()
				delete(ip.Pending, pendingKey)
				ip.Mutex.Unlock()
				continue
			}

			switch job.TaskType {
			case TaskThumbnail:
				ip.processThumbnailTask(job, mediaProcessor)
			case TaskMetadata:
				ip.processMetadataTask(job)
			case TaskDetection:
				ip.processDetectionTask(job, faceDetector)
			case TaskAlbumZip:
				ip.processAlbumZipTask(job, mediaStore)
			default:
				log.Printf("Worker %d: ERROR unknown task type '%s'", id, job.TaskType)
			}

			ip.Mutex.Lock()
			delete(ip.Pending, pendingKey)
			ip.Mutex.Unlock()

		case <-ip.StopChan:
			log.Printf("Image worker %d stopping: Stop signal received", id)
			return
		}
	}
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
}

// processDetectionTask performs detection and updates DB
func (ip *ImageProcessor) processDetectionTask(job ImageJob, faceDetector *media.DNNFaceDetector) {
	var taskErr error
	var detections []media.DetectionResult

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping detection task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for detection task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		if faceDetector != nil && faceDetector.Enabled {
			detections, taskErr = media.DetectFacesAndAnimals(job.OriginalImagePath, faceDetector)
			if taskErr != nil {
				log.Printf("Worker: ERROR during detection for %s: %v", job.OriginalRelativePath, taskErr)
			} else {
				log.Printf("Worker: Detection complete for %s: Found %d objects.", job.OriginalRelativePath, len(detections))
			}
		} else {
			taskErr = fmt.Errorf("face detector not enabled or loaded")
			log.Printf("Worker: Skipping detection for %s: detector disabled", job.OriginalRelativePath)
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

		savedZipFilename, zipSizeBytes, zipErr := utils.CreateAlbumZip(
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

// QueueJob queues a specific task if not already pending
func (ip *ImageProcessor) QueueJob(job ImageJob) bool {
	// use composite key: "relativePath:taskType"
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

	select {
	case ip.JobQueue <- job:
		log.Printf("Queued task '%s' for: %s", job.TaskType, job.OriginalRelativePath)
		return true
	default:
		log.Printf("WARNING: Image processing job queue full. Failed to queue task '%s' for: %s", job.TaskType, job.OriginalRelativePath)
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

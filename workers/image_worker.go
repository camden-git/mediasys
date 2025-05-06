package workers

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
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
	JobQueue chan ImageJob
	Config   config.Config
	DB       *sql.DB
	Wg       sync.WaitGroup
	StopChan chan struct{}
	Pending  map[string]bool
	Mutex    sync.Mutex
}

func NewImageProcessor(cfg config.Config, db *sql.DB, queueSize, numWorkers int) *ImageProcessor {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}
	proc := &ImageProcessor{
		JobQueue: make(chan ImageJob, queueSize),
		Config:   cfg,
		DB:       db,
		StopChan: make(chan struct{}),
		Pending:  make(map[string]bool),
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

	log.Printf("Worker %d: Loading DNN face detector...", id)
	faceDetector := utils.NewDNNFaceDetector(cfg.FaceDNNNetConfigPath, cfg.FaceDNNNetModelPath)
	defer func() {
		if faceDetector != nil {
			faceDetector.Close()
		}
		log.Printf("Worker %d: DNN Detector closed", id)
	}()
	if faceDetector == nil || !faceDetector.Enabled {
		log.Printf("Worker %d: DNN Face Detector failed to load or is disabled", id)
	}

	log.Printf("Image worker %d started", id)
	for {
		select {
		case job, ok := <-ip.JobQueue:
			if !ok {
				log.Printf("Image worker %d stopping: Job queue closed", id)
				return
			}

			var pendingKey string
			var statusColumn string
			var entityPath string

			if job.TaskType == TaskAlbumZip {
				pendingKey = fmt.Sprintf("album_%d:%s", job.AlbumID, job.TaskType)
				statusColumn = "zip_status"
				entityPath = fmt.Sprintf("album ID %d", job.AlbumID)
			} else {
				pendingKey = fmt.Sprintf("%s:%s", job.OriginalRelativePath, job.TaskType)
				statusColumn = job.TaskType + "_status"
				entityPath = job.OriginalRelativePath
			}

			log.Printf("Worker %d: Received job type '%s' for: %s", id, job.TaskType, entityPath)

			var markErr error
			if job.TaskType == TaskAlbumZip {
				markErr = database.MarkAlbumZipProcessing(ip.DB, job.AlbumID)
			} else {
				markErr = database.MarkImageTaskProcessing(ip.DB, job.OriginalRelativePath, statusColumn)
			}

			if markErr != nil {
				log.Printf("Worker %d: ERROR marking %s processing for %s: %v. Skipping job.", id, job.TaskType, entityPath, markErr)
				ip.Mutex.Lock()
				delete(ip.Pending, pendingKey)
				ip.Mutex.Unlock()
				continue
			}

			switch job.TaskType {
			case TaskThumbnail:
				ip.processThumbnailTask(job)
			case TaskMetadata:
				ip.processMetadataTask(job)
			case TaskDetection:
				ip.processDetectionTask(job, faceDetector)
			case TaskAlbumZip:
				ip.processAlbumZipTask(job)
			default:
				log.Printf("Worker %d: ERROR unknown task type '%s' for %s", id, job.TaskType, job.OriginalRelativePath)
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
func (ip *ImageProcessor) processThumbnailTask(job ImageJob) {
	var taskErr error
	var thumbPathPtr *string

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping thumbnail task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for thumbnail task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		thumbSavePath, genErr := utils.GenerateThumbnail(
			job.OriginalImagePath,
			ip.Config.ThumbnailDir,
			ip.Config.ThumbnailMaxSize,
		)
		if genErr != nil {
			taskErr = fmt.Errorf("thumbnail generation failed: %w", genErr)
			log.Printf("Worker: ERROR %v", taskErr)
		} else {
			thumbPathPtr = &thumbSavePath
			log.Printf("Worker: Generated thumbnail for %s", job.OriginalRelativePath)
		}
	}

	dbErr := database.UpdateImageThumbnailResult(ip.DB, job.OriginalRelativePath, thumbPathPtr, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating thumbnail DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

func (ip *ImageProcessor) processMetadataTask(job ImageJob) {
	var taskErr error
	var metadata *utils.Metadata

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping metadata task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for metadata task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		metadata, taskErr = utils.GetImageMetadata(job.OriginalImagePath)
		if taskErr != nil {
			log.Printf("Worker: ERROR extracting metadata for %s: %v", job.OriginalRelativePath, taskErr)
		} else {
			log.Printf("Worker: Extracted metadata for %s", job.OriginalRelativePath)
		}
	}

	dbErr := database.UpdateImageMetadataResult(ip.DB, job.OriginalRelativePath, metadata, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating metadata DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

// processDetectionTask performs detection and updates DB
func (ip *ImageProcessor) processDetectionTask(job ImageJob, faceDetector *utils.DNNFaceDetector) {
	var taskErr error
	var detections []utils.DetectionResult

	if _, statErr := os.Stat(job.OriginalImagePath); os.IsNotExist(statErr) {
		taskErr = fmt.Errorf("original file not found: %w", statErr)
		log.Printf("Worker: Skipping detection task for %s: %v", job.OriginalRelativePath, taskErr)
	} else if statErr != nil {
		taskErr = fmt.Errorf("failed to stat original file: %w", statErr)
		log.Printf("Worker: ERROR stating file for detection task %s: %v", job.OriginalRelativePath, taskErr)
	} else {
		if faceDetector != nil && faceDetector.Enabled {
			detections, taskErr = utils.DetectFacesAndAnimals(job.OriginalImagePath, faceDetector)
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

	dbErr := database.UpdateImageDetectionResult(ip.DB, job.OriginalRelativePath, detections, job.ModTimeUnix, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating detection DB result for %s: %v", job.OriginalRelativePath, dbErr)
	}
}

func (ip *ImageProcessor) processAlbumZipTask(job ImageJob) {
	log.Printf("Worker: Starting ZIP task for Album ID: %d", job.AlbumID)
	var taskErr error
	var finalZipPath *string
	var finalZipSize *int64

	album, err := database.GetAlbumByID(ip.DB, job.AlbumID)
	if err != nil {
		taskErr = fmt.Errorf("failed to fetch album details for ID %d: %w", job.AlbumID, err)
		log.Printf("Worker: ERROR %v", taskErr)
	} else {
		zipSaveDirName := "album_archives"
		zipSaveDir := filepath.Join(ip.Config.RootDirectory, zipSaveDirName)
		zipFilenameBase := fmt.Sprintf("album_%d_archive_%d", album.ID, time.Now().Unix()) // Add timestamp for uniqueness

		zipFullPath, zipSize, zipErr := utils.CreateAlbumZip(
			ip.Config.RootDirectory,
			album.FolderPath,
			zipSaveDir,
			zipFilenameBase,
		)
		if zipErr != nil {
			taskErr = fmt.Errorf("failed to create album zip: %w", zipErr)
			log.Printf("Worker: ERROR %v", taskErr)
		} else {
			relativePath := filepath.ToSlash(filepath.Join(zipSaveDirName, filepath.Base(zipFullPath)))
			finalZipPath = &relativePath
			finalZipSize = &zipSize
			log.Printf("Worker: Successfully created ZIP for Album ID %d: %s", job.AlbumID, relativePath)
		}
	}

	// for simplicity, we are not re-passing the album's updated_at. SetAlbumZipResult handles timestamps
	dbErr := database.SetAlbumZipResult(ip.DB, job.AlbumID, finalZipPath, finalZipSize, taskErr)
	if dbErr != nil {
		log.Printf("Worker: ERROR updating album ZIP DB result for Album ID %d: %v", job.AlbumID, dbErr)
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

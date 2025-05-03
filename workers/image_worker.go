package workers

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/utils"
)

// TaskType constants
const (
	TaskThumbnail = "thumbnail"
	TaskMetadata  = "metadata"
	TaskDetection = "detection"
)

type ImageJob struct {
	OriginalImagePath    string
	OriginalRelativePath string
	ModTimeUnix          int64
	TaskType             string
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

			pendingKey := fmt.Sprintf("%s:%s", job.OriginalRelativePath, job.TaskType)
			log.Printf("Worker %d: Received job type '%s' for: %s", id, job.TaskType, job.OriginalRelativePath)

			statusColumn := job.TaskType + "_status"
			err := database.MarkImageTaskProcessing(ip.DB, job.OriginalRelativePath, statusColumn)
			if err != nil {
				log.Printf("Worker %d: ERROR marking %s processing for %s: %v. Skipping job.", id, job.TaskType, job.OriginalRelativePath, err)
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

// QueueJob queues a specific task if not already pending
func (ip *ImageProcessor) QueueJob(job ImageJob) bool {
	// use composite key: "relativePath:taskType"
	pendingKey := fmt.Sprintf("%s:%s", job.OriginalRelativePath, job.TaskType)

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

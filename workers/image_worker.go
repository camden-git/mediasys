package workers

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/disintegration/imaging"
	"gocv.io/x/gocv"
	"gorm.io/gorm"
)

// Job types. TaskProcess covers metadata, thumbnail and preview in one pass so the
// original only has to be fetched from object storage once; progress is still
// reported per sub-task (TaskMetadata, TaskThumbnail, TaskPreview).
const (
	TaskProcess   = "process"
	TaskThumbnail = "thumbnail"
	TaskMetadata  = "metadata"
	TaskPreview   = "preview"
	TaskDetection = "detection"
	TaskAlbumZip  = "album_zip"
)

// how often the dispatcher polls the database for outstanding work
const dispatchInterval = 15 * time.Second

type ImageJob struct {
	ImagePath string
	TaskType  string
	AlbumID   uint
}

func (j ImageJob) key() string {
	if j.TaskType == TaskAlbumZip {
		return fmt.Sprintf("album_%d:%s", j.AlbumID, j.TaskType)
	}
	return j.ImagePath + ":" + j.TaskType
}

type ImageProcessor struct {
	JobQueue          chan ImageJob
	DetectionJobQueue chan ImageJob
	Config            config.Config
	ImageRepo         repository.ImageRepositoryInterface
	AlbumRepo         repository.AlbumRepositoryInterface
	TagRepo           repository.ImageTagRepositoryInterface
	Store             *media.Store
	Processor         *media.Processor
	Hub               *realtime.Hub

	wg       sync.WaitGroup
	stopChan chan struct{}
	wake     chan struct{}
	pending  map[string]bool
	mutex    sync.Mutex
}

func NewImageProcessor(
	cfg config.Config,
	imgRepo repository.ImageRepositoryInterface,
	albumRepo repository.AlbumRepositoryInterface,
	tagRepo repository.ImageTagRepositoryInterface,
	store *media.Store,
	hub *realtime.Hub,
) *ImageProcessor {
	numWorkers := max(cfg.NumThumbnailWorkers, 1)
	queueSize := max(cfg.ThumbnailQueueSize, 1)
	numDetectionWorkers := max(cfg.NumDetectionWorkers, 1)
	detectionQueueSize := max(cfg.DetectionQueueSize, 1)

	proc := &ImageProcessor{
		JobQueue:          make(chan ImageJob, queueSize),
		DetectionJobQueue: make(chan ImageJob, detectionQueueSize),
		Config:            cfg,
		ImageRepo:         imgRepo,
		AlbumRepo:         albumRepo,
		TagRepo:           tagRepo,
		Store:             store,
		Processor:         media.NewProcessor(store),
		Hub:               hub,
		stopChan:          make(chan struct{}),
		wake:              make(chan struct{}, 1),
		pending:           make(map[string]bool),
	}

	proc.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go proc.worker(i)
	}
	if cfg.FaceRecognitionEnabled {
		proc.wg.Add(numDetectionWorkers)
		for i := 0; i < numDetectionWorkers; i++ {
			go proc.detectionWorker(i)
		}
	}
	log.Printf("Started %d image worker(s) (queue %d) and %d detection worker(s) (queue %d)",
		numWorkers, queueSize, numDetectionWorkers, detectionQueueSize)
	return proc
}

// StartDispatcher recovers interrupted work and then keeps the queues fed from
// the database. The database is the source of truth for outstanding work, so
// nothing is lost when a queue is full or the process restarts.
func (ip *ImageProcessor) StartDispatcher() {
	if err := ip.ImageRepo.ResetInterruptedTasks(); err != nil {
		log.Printf("Dispatcher: failed to reset interrupted tasks: %v", err)
	}

	go func() {
		ticker := time.NewTicker(dispatchInterval)
		defer ticker.Stop()
		for {
			ip.dispatch()
			select {
			case <-ticker.C:
			case <-ip.wake:
			case <-ip.stopChan:
				return
			}
		}
	}()
}

// Wake asks the dispatcher to look for new work right away.
func (ip *ImageProcessor) Wake() {
	select {
	case ip.wake <- struct{}{}:
	default:
	}
}

func (ip *ImageProcessor) dispatch() {
	ip.mutex.Lock()
	inFlight := len(ip.pending)
	ip.mutex.Unlock()

	if albums, err := ip.AlbumRepo.ListPendingZips(); err == nil {
		for _, a := range albums {
			ip.QueueJob(ImageJob{AlbumID: a.ID, TaskType: TaskAlbumZip})
		}
	}
	if free := cap(ip.JobQueue) - len(ip.JobQueue); free > 0 {
		images, err := ip.ImageRepo.ListPendingProcessing(free + inFlight)
		if err != nil {
			log.Printf("Dispatcher: failed to list pending images: %v", err)
		}
		for _, img := range images {
			ip.QueueJob(ImageJob{ImagePath: img.OriginalPath, TaskType: TaskProcess})
		}
	}
	if ip.Config.FaceRecognitionEnabled {
		if free := cap(ip.DetectionJobQueue) - len(ip.DetectionJobQueue); free > 0 {
			images, err := ip.ImageRepo.ListPendingDetection(free + inFlight)
			if err != nil {
				log.Printf("Dispatcher: failed to list pending detections: %v", err)
			}
			for _, img := range images {
				ip.QueueJob(ImageJob{ImagePath: img.OriginalPath, TaskType: TaskDetection})
			}
		}
	}
}

// QueueJob queues a task unless it is already queued. Returns false when the
// task is a duplicate or the queue is full; the dispatcher will retry later.
func (ip *ImageProcessor) QueueJob(job ImageJob) bool {
	if job.TaskType == TaskDetection && !ip.Config.FaceRecognitionEnabled {
		return false
	}
	key := job.key()

	ip.mutex.Lock()
	defer ip.mutex.Unlock()
	if ip.pending[key] {
		return false
	}

	target := ip.JobQueue
	if job.TaskType == TaskDetection {
		target = ip.DetectionJobQueue
	}
	select {
	case target <- job:
		ip.pending[key] = true
		return true
	default:
		return false
	}
}

func (ip *ImageProcessor) done(job ImageJob) {
	ip.mutex.Lock()
	delete(ip.pending, job.key())
	ip.mutex.Unlock()
}

func (ip *ImageProcessor) broadcast(albumID uint, path, task, status string, err error) {
	if ip.Hub == nil {
		return
	}
	ev := realtime.Event{Type: "task", AlbumID: albumID, Path: path, Task: task, Status: status, Timestamp: time.Now().Unix()}
	if err != nil {
		ev.Error = err.Error()
	}
	ip.Hub.Broadcast(ev)
}

// worker handles processing and zip jobs. It never loads ML models.
func (ip *ImageProcessor) worker(id int) {
	defer ip.wg.Done()
	for {
		select {
		case job := <-ip.JobQueue:
			switch job.TaskType {
			case TaskProcess:
				ip.processImage(job)
			case TaskAlbumZip:
				ip.processAlbumZip(job)
			default:
				log.Printf("Worker %d: unknown task type %q", id, job.TaskType)
			}
			ip.done(job)
		case <-ip.stopChan:
			return
		}
	}
}

// detectionWorker loads the ML models once and handles only detection jobs.
func (ip *ImageProcessor) detectionWorker(id int) {
	defer ip.wg.Done()
	cfg := ip.Config

	faceDetector := media.NewDNNFaceDetector(cfg.FaceDNNNetConfigPath, cfg.FaceDNNNetModelPath)
	if faceDetector != nil {
		defer faceDetector.Close()
	}
	retinaFaceDetector := media.NewRetinaFaceDetector(cfg.RetinaFaceModelPath)
	if retinaFaceDetector != nil {
		defer retinaFaceDetector.Close()
	}
	recognitionModel := media.NewFaceRecognitionModel(cfg.FaceRecognitionModelPath, cfg.FaceRecognitionModelName)
	if recognitionModel != nil && recognitionModel.Enabled {
		defer recognitionModel.Close()
	}
	log.Printf("Detection worker %d started", id)

	for {
		select {
		case job := <-ip.DetectionJobQueue:
			ip.processDetection(job, faceDetector, retinaFaceDetector, recognitionModel)
			ip.done(job)
		case <-ip.stopChan:
			return
		}
	}
}

// processImage fetches the original once and runs every outstanding
// metadata/thumbnail/preview task for it.
func (ip *ImageProcessor) processImage(job ImageJob) {
	img, err := ip.ImageRepo.GetByPath(job.ImagePath)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Worker: failed to load image %s: %v", job.ImagePath, err)
		}
		return
	}
	needMeta := img.MetadataStatus == database.StatusPending
	needThumb := img.ThumbnailStatus == database.StatusPending
	needPreview := img.PreviewStatus == database.StatusPending
	if !needMeta && !needThumb && !needPreview {
		return
	}

	ctx := context.Background()
	localPath, cleanup, dlErr := ip.Store.Download(ctx, img.ObjectKey)
	defer cleanup()

	// decode once (EXIF orientation applied) for thumbnail, preview and dimensions
	var decoded image.Image
	var decodeErr error
	if dlErr == nil && (needThumb || needPreview || needMeta) {
		decoded, decodeErr = imaging.Open(localPath, imaging.AutoOrientation(true))
		if decodeErr != nil {
			decodeErr = fmt.Errorf("failed to decode image: %w", decodeErr)
		}
	}
	firstErr := func(errs ...error) error {
		for _, e := range errs {
			if e != nil {
				return e
			}
		}
		return nil
	}

	if needMeta {
		ip.runTask(img.AlbumID, img.OriginalPath, TaskMetadata, func() error {
			if dlErr != nil {
				return ip.ImageRepo.UpdateMetadataResult(img.OriginalPath, nil, dlErr)
			}
			meta, metaErr := media.GetImageMetadata(localPath)
			if metaErr == nil && decoded != nil {
				// store the displayed (oriented) size so layouts match thumbnails
				w, h := decoded.Bounds().Dx(), decoded.Bounds().Dy()
				meta.Width, meta.Height = &w, &h
			}
			if dbErr := ip.ImageRepo.UpdateMetadataResult(img.OriginalPath, meta, metaErr); dbErr != nil {
				return dbErr
			}
			if metaErr == nil && meta != nil && ip.TagRepo != nil {
				if tagErr := ip.TagRepo.SetXMPTags(img.OriginalPath, buildXMPTags(img.OriginalPath, meta.Keywords)); tagErr != nil {
					log.Printf("Worker: failed to update XMP tags for %s: %v", img.OriginalPath, tagErr)
				}
			}
			return metaErr
		})
	}

	if needThumb {
		ip.runTask(img.AlbumID, img.OriginalPath, TaskThumbnail, func() error {
			var key *string
			taskErr := firstErr(dlErr, decodeErr)
			if taskErr == nil {
				k, genErr := ip.Processor.GenerateThumbnail(decoded, img.OriginalPath, ip.Config.ThumbnailMaxSize)
				if genErr != nil {
					taskErr = genErr
				} else {
					key = &k
				}
			}
			if dbErr := ip.ImageRepo.UpdateThumbnailResult(img.OriginalPath, key, taskErr); dbErr != nil {
				return dbErr
			}
			return taskErr
		})
	}

	if needPreview {
		ip.runTask(img.AlbumID, img.OriginalPath, TaskPreview, func() error {
			var key *string
			taskErr := firstErr(dlErr, decodeErr)
			if taskErr == nil {
				k, genErr := ip.Processor.GeneratePreview(decoded, img.OriginalPath)
				if genErr != nil {
					taskErr = genErr
				} else {
					key = &k
				}
			}
			if dbErr := ip.ImageRepo.UpdatePreviewResult(img.OriginalPath, key, taskErr); dbErr != nil {
				return dbErr
			}
			return taskErr
		})
	}
}

// runTask wraps a sub-task with status bookkeeping and realtime events.
func (ip *ImageProcessor) runTask(albumID uint, path, task string, fn func() error) {
	if err := ip.ImageRepo.MarkTaskProcessing(path, task+"_status"); err != nil {
		log.Printf("Worker: cannot mark %s processing for %s: %v", task, path, err)
		return
	}
	ip.broadcast(albumID, path, task, "processing", nil)
	if err := fn(); err != nil {
		log.Printf("Worker: %s failed for %s: %v", task, path, err)
		ip.broadcast(albumID, path, task, "error", err)
		return
	}
	ip.broadcast(albumID, path, task, "done", nil)
}

// buildXMPTags converts raw XMP keyword strings into ImageTag records.
// Keywords containing "/" are treated as hierarchical "key/value" pairs.
// Others use tag_key="keyword" with the raw string as tag_value.
func buildXMPTags(imagePath string, keywords []string) []models.ImageTag {
	tags := make([]models.ImageTag, 0, len(keywords))
	seen := make(map[string]bool)
	for _, kw := range keywords {
		var key, val string
		if idx := strings.Index(kw, "/"); idx > 0 {
			key = strings.TrimSpace(kw[:idx])
			val = strings.TrimSpace(kw[idx+1:])
		} else {
			key = "keyword"
			val = strings.TrimSpace(kw)
		}
		if key == "" || val == "" || seen[key+"\x00"+val] {
			continue
		}
		seen[key+"\x00"+val] = true
		tags = append(tags, models.ImageTag{
			ImagePath: imagePath,
			TagKey:    key,
			TagValue:  val,
			Source:    "xmp",
			CreatedAt: time.Now(),
		})
	}
	return tags
}

// processDetection runs face detection (and recognition when available).
func (ip *ImageProcessor) processDetection(job ImageJob, faceDetector *media.DNNFaceDetector, retinaFaceDetector *media.RetinaFaceDetector, recognitionModel *media.FaceRecognitionModel) {
	img, err := ip.ImageRepo.GetByPath(job.ImagePath)
	if err != nil || img.DetectionStatus != database.StatusPending {
		return
	}

	ip.runTask(img.AlbumID, img.OriginalPath, TaskDetection, func() error {
		var detections []media.DetectionResult
		taskErr := func() error {
			localPath, cleanup, err := ip.Store.Download(context.Background(), img.ObjectKey)
			defer cleanup()
			if err != nil {
				return err
			}
			switch {
			case retinaFaceDetector != nil && retinaFaceDetector.Enabled:
				mat := gocv.IMRead(localPath, gocv.IMReadColor)
				if mat.Empty() {
					return fmt.Errorf("failed to read image for detection")
				}
				defer mat.Close()
				if recognitionModel != nil && recognitionModel.Enabled {
					detections = retinaFaceDetector.DetectFacesAndExtractEmbeddings(mat, recognitionModel)
				} else {
					detections = retinaFaceDetector.DetectFaces(mat)
				}
				return nil
			case faceDetector != nil && faceDetector.Enabled:
				var dErr error
				detections, dErr = media.DetectFacesAndAnimals(localPath, faceDetector)
				return dErr
			default:
				return fmt.Errorf("no face detector enabled or loaded")
			}
		}()
		if dbErr := ip.ImageRepo.UpdateDetectionResult(img.OriginalPath, detections, taskErr); dbErr != nil {
			return dbErr
		}
		return taskErr
	})
}

// processAlbumZip builds an archive of an album's originals directly in object storage.
func (ip *ImageProcessor) processAlbumZip(job ImageJob) {
	if err := ip.AlbumRepo.MarkZipProcessing(job.AlbumID); err != nil {
		log.Printf("Worker: cannot mark zip processing for album %d: %v", job.AlbumID, err)
		return
	}
	path := fmt.Sprintf("album:%d", job.AlbumID)
	ip.broadcast(job.AlbumID, path, TaskAlbumZip, "processing", nil)

	var key *string
	var size *int64
	album, taskErr := ip.AlbumRepo.GetByID(job.AlbumID)
	if taskErr == nil {
		var images []models.Image
		images, taskErr = ip.ImageRepo.ListByAlbum(album.ID, nil)
		if taskErr == nil {
			safeSlug := strings.NewReplacer("/", "_", "\\", "_").Replace(album.Slug)
			k := fmt.Sprintf("%salbum_%s_%d_%d.zip", media.PrefixArchives, safeSlug, album.ID, time.Now().Unix())
			var n int64
			n, taskErr = CreateAlbumZip(context.Background(), ip.Store, album.FolderPath, images, k)
			if taskErr == nil {
				key, size = &k, &n
			}
		}
	}

	if dbErr := ip.AlbumRepo.SetZipResult(job.AlbumID, key, size, taskErr); dbErr != nil {
		log.Printf("Worker: failed to store zip result for album %d: %v", job.AlbumID, dbErr)
		if key != nil {
			_ = ip.Store.Delete(context.Background(), *key)
		}
		return
	}
	if taskErr != nil {
		log.Printf("Worker: zip failed for album %d: %v", job.AlbumID, taskErr)
		ip.broadcast(job.AlbumID, path, TaskAlbumZip, "error", taskErr)
		return
	}
	// remove the archive this one replaced
	if album.ZipPath != nil && *album.ZipPath != *key {
		_ = ip.Store.Delete(context.Background(), *album.ZipPath)
	}
	ip.broadcast(job.AlbumID, path, TaskAlbumZip, "done", nil)
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
				log.Printf("Memory trim: HeapSys before=%d MB, after=%d MB",
					before.HeapSys/(1<<20), after.HeapSys/(1<<20))
			case <-ip.stopChan:
				return
			}
		}
	}()
	log.Printf("Memory trimmer started (interval: %d minutes)", intervalMinutes)
}

// Stop signals every worker to exit and waits for them.
func (ip *ImageProcessor) Stop() {
	close(ip.stopChan)
	ip.wg.Wait()
}

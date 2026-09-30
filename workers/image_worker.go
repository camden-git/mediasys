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

	// ctx is cancelled when a graceful shutdown times out, aborting in-flight
	// storage transfers. Work interrupted this way stays in the database as
	// "processing" and is reset to pending on the next start.
	ctx    context.Context
	cancel context.CancelFunc

	detectionEnabled bool

	wg       sync.WaitGroup
	stopOnce sync.Once
	stopChan chan struct{}
	wake     chan struct{}
	pending  map[string]bool
	mutex    sync.Mutex
}

// detectorSet holds the ML models used by one detection worker. Networks are not
// safe for concurrent use, so every worker owns its own set.
type detectorSet struct {
	dnn    *media.DNNFaceDetector
	retina *media.RetinaFaceDetector
	recog  *media.FaceRecognitionModel
}

func (d *detectorSet) Close() {
	d.dnn.Close()
	d.retina.Close()
	d.recog.Close()
}

// loadDetectors loads the configured models. It fails when no detector could be
// loaded, or when a recognition model is configured but unavailable, since faces
// stored without embeddings could never be matched later.
func loadDetectors(cfg config.Config) (*detectorSet, error) {
	set := &detectorSet{}
	set.retina = media.NewRetinaFaceDetector(cfg.RetinaFaceModelPath)
	if !set.retina.Enabled {
		// the legacy Caffe detector is only a fallback for RetinaFace
		set.dnn = media.NewDNNFaceDetector(cfg.FaceDNNNetConfigPath, cfg.FaceDNNNetModelPath)
	}
	set.recog = media.NewFaceRecognitionModel(cfg.FaceRecognitionModelPath, cfg.FaceRecognitionModelName)

	switch {
	case !set.retina.Enabled && !set.dnn.Enabled:
		set.Close()
		return nil, errors.New("no face detector model could be loaded")
	case cfg.FaceRecognitionModelPath != "" && !set.recog.Enabled:
		set.Close()
		return nil, errors.New("face recognition model could not be loaded")
	}
	return set, nil
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

	ctx, cancel := context.WithCancel(context.Background())
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
		ctx:               ctx,
		cancel:            cancel,
		stopChan:          make(chan struct{}),
		wake:              make(chan struct{}, 1),
		pending:           make(map[string]bool),
	}

	proc.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go proc.worker(i)
	}

	// Without a usable detector images stay pending instead of being marked as
	// failed, so they are processed once the models are in place.
	startedDetection := 0
	if cfg.FaceRecognitionEnabled {
		first, err := loadDetectors(cfg)
		if err != nil {
			log.Printf("Face detection disabled: %v; images stay pending until models are available", err)
		} else {
			proc.detectionEnabled = true
			startedDetection = numDetectionWorkers
			proc.wg.Add(numDetectionWorkers)
			for i := 0; i < numDetectionWorkers; i++ {
				go proc.detectionWorker(i, first)
				first = nil // only the first worker reuses the probed models
			}
		}
	}
	log.Printf("Started %d image worker(s) (queue %d) and %d detection worker(s) (queue %d)",
		numWorkers, queueSize, startedDetection, detectionQueueSize)
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
	if ip.detectionEnabled {
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
	if job.TaskType == TaskDetection && !ip.detectionEnabled {
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
			ip.guard(job, func() {
				switch job.TaskType {
				case TaskProcess:
					ip.processImage(job)
				case TaskAlbumZip:
					ip.processAlbumZip(job)
				default:
					log.Printf("Worker %d: unknown task type %q", id, job.TaskType)
				}
			})
		case <-ip.stopChan:
			return
		}
	}
}

// detectionWorker owns one set of ML models (loaded here unless preloaded) and
// handles only detection jobs.
func (ip *ImageProcessor) detectionWorker(id int, set *detectorSet) {
	defer ip.wg.Done()
	if set == nil {
		var err error
		if set, err = loadDetectors(ip.Config); err != nil {
			log.Printf("Detection worker %d not started: %v", id, err)
			return
		}
	}
	defer set.Close()
	log.Printf("Detection worker %d started", id)

	for {
		select {
		case job := <-ip.DetectionJobQueue:
			ip.guard(job, func() { ip.processDetection(job, set) })
		case <-ip.stopChan:
			return
		}
	}
}

// guard runs a job, keeps the worker alive if it panics, and releases the job.
func (ip *ImageProcessor) guard(job ImageJob, fn func()) {
	defer ip.done(job)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Worker: panic in %s job for %s: %v\n%s", job.TaskType, job.ImagePath, r, debug.Stack())
		}
	}()
	fn()
}

// errShutdown marks work abandoned because the processor is shutting down.
var errShutdown = errors.New("image processor is shutting down")

// persistError wraps a failure to record a task outcome in the database.
type persistError struct{ err error }

func (e persistError) Error() string { return "failed to record task result: " + e.err.Error() }
func (e persistError) Unwrap() error { return e.err }

const (
	persistAttempts = 3
	persistBackoff  = 250 * time.Millisecond
)

// persist records a task outcome, retrying transient database errors. A stale
// result (image deleted or replaced) is not retried.
func (ip *ImageProcessor) persist(fn func() error) error {
	var err error
	for attempt := 1; attempt <= persistAttempts; attempt++ {
		if ip.ctx.Err() != nil {
			return errShutdown
		}
		if err = fn(); err == nil || errors.Is(err, repository.ErrStaleImage) {
			return err
		}
		if attempt < persistAttempts {
			select {
			case <-time.After(persistBackoff * time.Duration(attempt)):
			case <-ip.ctx.Done():
				return errShutdown
			}
		}
	}
	return persistError{err}
}

// dropObjects removes generated objects whose database row could not be written.
func (ip *ImageProcessor) dropObjects(keys ...*string) {
	for _, k := range keys {
		if k != nil {
			if err := ip.Store.Delete(context.Background(), *k); err != nil {
				log.Printf("Worker: failed to remove orphaned object %s: %v", *k, err)
			}
		}
	}
}

// decodeOriented decodes the downloaded original, turning decoder panics into errors.
func decodeOriented(localPath string) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("decoder panic: %v", r)
		}
	}()
	img, err = media.OpenImage(localPath)
	if err != nil {
		err = fmt.Errorf("failed to decode image: %w", err)
	}
	return img, err
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

	localPath, cleanup, dlErr := ip.Store.Download(ip.ctx, img.ObjectKey)
	defer cleanup()

	// decode once (EXIF orientation applied) for thumbnail, preview and dimensions
	var decoded image.Image
	var decodeErr error
	if dlErr == nil {
		decoded, decodeErr = decodeOriented(localPath)
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
		ip.runTask(img, TaskMetadata, func() error {
			if dlErr != nil {
				return ip.recordMetadata(img, nil, dlErr)
			}
			meta, metaErr := media.GetImageMetadata(localPath)
			if metaErr == nil && decoded != nil {
				// store the displayed (oriented) size so layouts match thumbnails
				w, h := decoded.Bounds().Dx(), decoded.Bounds().Dy()
				meta.Width, meta.Height = &w, &h
			}
			if dbErr := ip.recordMetadata(img, meta, metaErr); dbErr != nil {
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
		ip.runTask(img, TaskThumbnail, func() error {
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
			if dbErr := ip.persist(func() error {
				return ip.ImageRepo.UpdateThumbnailResult(img.OriginalPath, img.ObjectKey, key, taskErr)
			}); dbErr != nil {
				ip.dropObjects(key)
				return dbErr
			}
			return taskErr
		})
	}

	if needPreview {
		ip.runTask(img, TaskPreview, func() error {
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
			if dbErr := ip.persist(func() error {
				return ip.ImageRepo.UpdatePreviewResult(img.OriginalPath, img.ObjectKey, key, taskErr)
			}); dbErr != nil {
				ip.dropObjects(key)
				return dbErr
			}
			return taskErr
		})
	}
}

func (ip *ImageProcessor) recordMetadata(img *models.Image, meta *media.Metadata, taskErr error) error {
	return ip.persist(func() error {
		return ip.ImageRepo.UpdateMetadataResult(img.OriginalPath, img.ObjectKey, meta, taskErr)
	})
}

// runTask wraps a sub-task with status bookkeeping and realtime events. fn is
// responsible for recording its own result; a panic marks the task as failed.
func (ip *ImageProcessor) runTask(img *models.Image, task string, fn func() error) {
	path, column := img.OriginalPath, task+"_status"
	if err := ip.ImageRepo.MarkTaskProcessing(path, column); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Worker: cannot mark %s processing for %s: %v", task, path, err)
		}
		return
	}
	ip.broadcast(img.AlbumID, path, task, "processing", nil)

	err, panicked := ip.safely(fn)
	var perr persistError
	switch {
	case err == nil:
		ip.broadcast(img.AlbumID, path, task, "done", nil)
		return
	case errors.Is(err, errShutdown):
		return // stays "processing"; reset to pending on next start
	case errors.Is(err, repository.ErrStaleImage):
		return // image was deleted or re-uploaded; nothing to report
	case panicked:
		log.Printf("Worker: %s panicked for %s: %v", task, path, err)
		if markErr := ip.persist(func() error {
			return ip.ImageRepo.MarkTaskError(path, img.ObjectKey, column, err)
		}); markErr != nil && !errors.Is(markErr, repository.ErrStaleImage) && !errors.Is(markErr, errShutdown) {
			log.Printf("Worker: could not mark %s failed for %s: %v", task, path, markErr)
		}
	case errors.As(err, &perr):
		// the result could not be stored; put the task back so it is retried
		// instead of staying "processing" until the next restart
		log.Printf("Worker: %s for %s: %v", task, path, err)
		if reqErr := ip.ImageRepo.RequeueTask(path, column); reqErr != nil {
			log.Printf("Worker: could not requeue %s for %s: %v", task, path, reqErr)
		}
		return
	default:
		log.Printf("Worker: %s failed for %s: %v", task, path, err)
	}
	ip.broadcast(img.AlbumID, path, task, "error", err)
}

// safely runs fn, converting a panic into an error.
func (ip *ImageProcessor) safely(fn func() error) (err error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			err, panicked = fmt.Errorf("panic: %v\n%s", r, debug.Stack()), true
		}
	}()
	return fn(), false
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
func (ip *ImageProcessor) processDetection(job ImageJob, set *detectorSet) {
	img, err := ip.ImageRepo.GetByPath(job.ImagePath)
	if err != nil || img.DetectionStatus != database.StatusPending {
		return
	}

	ip.runTask(img, TaskDetection, func() error {
		detections, taskErr := ip.detect(img, set)
		return func() error {
			if dbErr := ip.persist(func() error {
				return ip.ImageRepo.UpdateDetectionResult(img.OriginalPath, img.ObjectKey, detections, taskErr)
			}); dbErr != nil {
				return dbErr
			}
			return taskErr
		}()
	})
}

// detect downloads the original and runs the loaded detector on it.
func (ip *ImageProcessor) detect(img *models.Image, set *detectorSet) ([]media.DetectionResult, error) {
	localPath, cleanup, err := ip.Store.Download(ip.ctx, img.ObjectKey)
	defer cleanup()
	if err != nil {
		return nil, err
	}
	if err := media.CheckImageFile(localPath); err != nil {
		return nil, err
	}

	switch {
	case set.retina != nil && set.retina.Enabled:
		mat := gocv.IMRead(localPath, gocv.IMReadColor)
		if mat.Empty() {
			return nil, errors.New("failed to read image for detection")
		}
		defer mat.Close()
		if set.recog != nil && set.recog.Enabled {
			return set.retina.DetectFacesAndExtractEmbeddings(mat, set.recog)
		}
		return set.retina.DetectFaces(mat)
	case set.dnn != nil && set.dnn.Enabled:
		return media.DetectFacesAndAnimals(localPath, set.dnn)
	default:
		return nil, errors.New("no face detector enabled or loaded")
	}
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
	var taskErr error
	defer func() {
		// a panic must not leave the archive stuck in "processing"
		if r := recover(); r != nil {
			log.Printf("Worker: zip panicked for album %d: %v\n%s", job.AlbumID, r, debug.Stack())
			_ = ip.AlbumRepo.SetZipResult(job.AlbumID, nil, nil, fmt.Errorf("panic: %v", r))
			ip.broadcast(job.AlbumID, path, TaskAlbumZip, "error", fmt.Errorf("panic: %v", r))
		}
	}()
	album, taskErr := ip.AlbumRepo.GetByID(job.AlbumID)
	if taskErr == nil {
		var images []models.Image
		images, taskErr = ip.ImageRepo.ListByAlbum(album.ID, nil)
		if taskErr == nil {
			safeSlug := strings.NewReplacer("/", "_", "\\", "_").Replace(album.Slug)
			k := fmt.Sprintf("%salbum_%s_%d_%d.zip", media.PrefixArchives, safeSlug, album.ID, time.Now().Unix())
			var n int64
			n, taskErr = CreateAlbumZip(ip.ctx, ip.Store, album.FolderPath, images, k)
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

// Stop signals every worker to exit and waits for in-flight jobs to finish.
func (ip *ImageProcessor) Stop() {
	_ = ip.Shutdown(context.Background())
}

// Shutdown signals every worker to exit and waits for in-flight jobs until ctx
// is done. When ctx expires, in-flight storage transfers are cancelled and
// ctx's error is returned; interrupted tasks are reset to pending on the next
// start. It is safe to call more than once.
func (ip *ImageProcessor) Shutdown(ctx context.Context) error {
	ip.stopOnce.Do(func() { close(ip.stopChan) })

	finished := make(chan struct{})
	go func() {
		ip.wg.Wait()
		close(finished)
	}()
	select {
	case <-finished:
		ip.cancel()
		return nil
	case <-ctx.Done():
		ip.cancel()
		return ctx.Err()
	}
}

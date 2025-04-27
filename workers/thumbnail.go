package workers

import (
	"database/sql"
	"log"
	"os"
	"sync"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/utils"
)

type ThumbnailJob struct {
	OriginalImagePath    string
	OriginalRelativePath string
	ModTimeUnix          int64
}

type ThumbnailGenerator struct {
	JobQueue chan ThumbnailJob
	Config   config.Config
	DB       *sql.DB
	Wg       sync.WaitGroup
	StopChan chan struct{}
	Pending  map[string]bool
	Mutex    sync.Mutex
}

func NewThumbnailGenerator(cfg config.Config, db *sql.DB, queueSize, numWorkers int) *ThumbnailGenerator {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	gen := &ThumbnailGenerator{
		JobQueue: make(chan ThumbnailJob, queueSize),
		Config:   cfg,
		DB:       db,
		StopChan: make(chan struct{}),
		Pending:  make(map[string]bool),
	}

	gen.Wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go gen.worker(i)
	}
	log.Printf("started %d thumbnail worker(s) with queue size %d", numWorkers, queueSize)

	return gen
}

func (tg *ThumbnailGenerator) worker(id int) {
	defer tg.Wg.Done()
	log.Printf("thumbnail worker %d started", id)
	for {
		select {
		case job, ok := <-tg.JobQueue:
			if !ok {
				log.Printf("thumbnail worker %d stopping: Job queue closed", id)
				return
			}
			log.Printf("worker %d processing job for: %s", id, job.OriginalRelativePath)
			tg.processJob(job)
			tg.Mutex.Lock()
			delete(tg.Pending, job.OriginalRelativePath)
			tg.Mutex.Unlock()

		case <-tg.StopChan:
			log.Printf("thumbnail worker %d stopping: stop signal received", id)
			return
		}
	}
}

func (tg *ThumbnailGenerator) processJob(job ThumbnailJob) {
	if _, err := os.Stat(job.OriginalImagePath); os.IsNotExist(err) {
		log.Printf("original file %s not found, skipping thumbnail generation", job.OriginalImagePath)
		return
	} else if err != nil {
		log.Printf("error stating original file %s before thumbnail generation: %v", job.OriginalImagePath, err)
	}

	thumbSavePath, err := utils.GenerateThumbnail(
		job.OriginalImagePath,
		tg.Config.ThumbnailDir,
		tg.Config.ThumbnailWidth,
		tg.Config.ThumbnailHeight,
	)

	if err != nil {
		log.Printf("ERROR generating thumbnail for %s (relative: %s): %v",
			job.OriginalImagePath, job.OriginalRelativePath, err)
		return
	}

	err = database.SetThumbnailInfo(tg.DB, job.OriginalRelativePath, thumbSavePath, job.ModTimeUnix)
	if err != nil {
		log.Printf("ERROR updating thumbnail DB record for %s after generation: %v", job.OriginalRelativePath, err)
		return
	}

	log.Printf("successfully generated thumbnail and updated DB for: %s", job.OriginalRelativePath)
}

func (tg *ThumbnailGenerator) QueueJob(job ThumbnailJob) bool {
	tg.Mutex.Lock()
	if tg.Pending[job.OriginalRelativePath] {
		tg.Mutex.Unlock()
		log.Printf("thumbnail generation for %s already pending, skipping queue", job.OriginalRelativePath)
		return false
	}

	tg.Pending[job.OriginalRelativePath] = true
	tg.Mutex.Unlock()

	select {
	case tg.JobQueue <- job:
		log.Printf("queued thumbnail generation for: %s", job.OriginalRelativePath)
		return true
	default:
		log.Printf("WARNING: Thumbnail job queue full!!!! failed to queue job for: %s", job.OriginalRelativePath)
		tg.Mutex.Lock()
		delete(tg.Pending, job.OriginalRelativePath)
		tg.Mutex.Unlock()
		return false
	}
}

func (tg *ThumbnailGenerator) Stop() {
	log.Println("stopping thumbnail generator...")
	close(tg.StopChan)
	tg.Wg.Wait()
	log.Println("all thumbnail workers stopped")
}

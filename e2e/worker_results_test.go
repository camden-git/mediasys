package e2e_test

import (
	"errors"
	"testing"
	"time"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
)

// A task result must be ignored when the image was deleted or re-uploaded
// (different object key) while the worker was busy, and must not leave faces
// or embeddings behind for the vanished image.
func TestWorkerResultsIgnoredForStaleImage(t *testing.T) {
	env := requireShared(t)
	repo := repository.NewImageRepository(env.app.DB)

	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "Stale "+suffix, "stale-"+suffix, "")
	path := "stale-" + suffix + "/a.jpg"
	now := time.Now().Unix()
	img := &models.Image{OriginalPath: path, AlbumID: album.ID, ObjectKey: "originals/" + path + ".v2", CreatedAt: now, LastModified: now}
	if _, err := repo.Upsert(img); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	thumb := "thumbnails/x.webp"
	if err := repo.UpdateThumbnailResult(path, "originals/"+path+".v1", &thumb, nil); !errors.Is(err, repository.ErrStaleImage) {
		t.Fatalf("stale thumbnail result: got %v, want ErrStaleImage", err)
	}
	detections := []media.DetectionResult{{X: 1, Y: 1, W: 10, H: 10, Confidence: 0.9, Embedding: make([]float32, 512), ModelName: "arcface"}}
	if err := repo.UpdateDetectionResult(path, "originals/"+path+".v1", detections, nil); !errors.Is(err, repository.ErrStaleImage) {
		t.Fatalf("stale detection result: got %v, want ErrStaleImage", err)
	}
	var faces int64
	env.app.DB.Model(&models.Face{}).Where("image_path = ?", path).Count(&faces)
	if faces != 0 {
		t.Fatalf("stale detection created %d faces", faces)
	}

	if err := repo.UpdateThumbnailResult(path, img.ObjectKey, &thumb, nil); err != nil {
		t.Fatalf("current thumbnail result: %v", err)
	}
	if err := repo.UpdateDetectionResult(path, img.ObjectKey, detections, nil); err != nil {
		t.Fatalf("current detection result: %v", err)
	}
	env.app.DB.Model(&models.Face{}).Where("image_path = ?", path).Count(&faces)
	if faces != 1 {
		t.Fatalf("expected 1 face, got %d", faces)
	}

	if err := repo.MarkTaskError(path, img.ObjectKey, "preview_status", errors.New("boom")); err != nil {
		t.Fatalf("MarkTaskError: %v", err)
	}
	got, err := repo.GetByPath(path)
	if err != nil || got.PreviewStatus != "error" || got.PreviewError == nil || *got.PreviewError != "boom" {
		t.Fatalf("preview not marked failed: %+v %v", got, err)
	}
}

// Images queued for face detection while it was enabled must not stay pending
// forever once face recognition is turned off.
func TestMarkDetectionNotRequired(t *testing.T) {
	env := requireShared(t)
	repo := repository.NewImageRepository(env.app.DB)

	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "NoDetect "+suffix, "nodetect-"+suffix, "")
	now := time.Now().Unix()
	want := map[string]string{"pending": "notRequired", "processing": "notRequired", "done": "done", "error": "error"}
	for status := range want {
		path := "nodetect-" + suffix + "/" + status + ".jpg"
		img := &models.Image{OriginalPath: path, AlbumID: album.ID, ObjectKey: "originals/" + path, DetectionStatus: status, CreatedAt: now, LastModified: now}
		if _, err := repo.Upsert(img); err != nil {
			t.Fatalf("Upsert %s: %v", status, err)
		}
	}

	if _, err := repo.MarkDetectionNotRequired(); err != nil {
		t.Fatalf("MarkDetectionNotRequired: %v", err)
	}

	for status, exp := range want {
		got, err := repo.GetByPath("nodetect-" + suffix + "/" + status + ".jpg")
		if err != nil || got.DetectionStatus != exp {
			t.Fatalf("%s: got %+v (err %v), want detection_status %q", status, got, err, exp)
		}
	}
}

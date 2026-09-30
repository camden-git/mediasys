package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"log"
	"net/http"
	"time"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/disintegration/imaging"
	"gorm.io/gorm"
)

const (
	// faceThumbMaxSize is the max width/height of a cached face crop.
	faceThumbMaxSize = 256
	// faceThumbCacheControl lets browsers reuse a crop for a day; the ETag covers revalidation.
	faceThumbCacheControl = "public, max-age=86400"
)

// faceThumbPrefix is the object-storage prefix under which all cached crops of a face live.
func faceThumbPrefix(faceID uint) string {
	return fmt.Sprintf("face_thumbs/%d/", faceID)
}

// faceThumbHash identifies a crop by the image and box it was cut from, so moving the
// box naturally produces a new object.
func faceThumbHash(face *models.Face) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d,%d,%d,%d|%d", face.ImagePath, face.X1, face.Y1, face.X2, face.Y2, faceThumbMaxSize)))
	return hex.EncodeToString(sum[:8])
}

// deleteFaceThumbnails removes every cached crop of a face (best effort).
func deleteFaceThumbnails(store *media.Store, faceID uint) {
	if store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store.DeletePrefix(ctx, faceThumbPrefix(faceID))
}

// cropFace cuts a square, padded crop around the face box, shifting (not shrinking) the
// window to stay inside the image, and scales it down to faceThumbMaxSize.
func cropFace(src image.Image, face *models.Face) (image.Image, error) {
	imgW, imgH := src.Bounds().Dx(), src.Bounds().Dy()
	side := max(face.X2-face.X1, face.Y2-face.Y1)
	// 20% padding on each side
	side = int(float64(side) * 1.4)
	side = max(1, min(side, imgW, imgH))

	x1 := (face.X1+face.X2)/2 - side/2
	y1 := (face.Y1+face.Y2)/2 - side/2
	x1 = max(0, min(x1, imgW-side))
	y1 = max(0, min(y1, imgH-side))

	crop := imaging.Crop(src, image.Rect(x1, y1, x1+side, y1+side))
	if crop.Bounds().Empty() {
		return nil, errors.New("empty face crop")
	}
	return imaging.Fit(crop, faceThumbMaxSize, faceThumbMaxSize, imaging.Lanczos), nil
}

// serveFaceThumbnail writes the cached JPEG crop of a face, generating it from the
// original image on first use.
func serveFaceThumbnail(w http.ResponseWriter, r *http.Request, store *media.Store, imageRepo repository.ImageRepositoryInterface, face *models.Face) {
	hash := faceThumbHash(face)
	key := faceThumbPrefix(face.ID) + hash + ".jpg"
	w.Header().Set("ETag", `"`+hash+`"`)
	w.Header().Set("Cache-Control", faceThumbCacheControl)
	w.Header().Set("Content-Type", "image/jpeg")

	if r.Header.Get("If-None-Match") == `"`+hash+`"` {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if obj, info, err := store.Get(r.Context(), key); err == nil {
		defer obj.Close()
		http.ServeContent(w, r, "", info.LastModified, obj)
		return
	} else if !errors.Is(err, media.ErrNotFound) {
		log.Printf("face thumbnail: failed to read cache %s: %v", key, err)
	}

	img, err := imageRepo.GetByPath(face.ImagePath)
	if err != nil {
		w.Header().Del("ETag")
		w.Header().Del("Cache-Control")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "ImageNotFound", "face image not found")
		} else {
			log.Printf("face thumbnail: failed to load image %s: %v", face.ImagePath, err)
			WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		}
		return
	}
	obj, _, err := store.Get(r.Context(), img.ObjectKey)
	if err != nil {
		log.Printf("face thumbnail: failed to open image %s: %v", img.ObjectKey, err)
		w.Header().Del("ETag")
		w.Header().Del("Cache-Control")
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}
	src, err := imaging.Decode(obj, imaging.AutoOrientation(true))
	obj.Close()
	if err != nil {
		log.Printf("face thumbnail: failed to decode image %s: %v", img.ObjectKey, err)
		w.Header().Del("ETag")
		w.Header().Del("Cache-Control")
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}

	thumb, err := cropFace(src, face)
	var buf bytes.Buffer
	if err == nil {
		err = imaging.Encode(&buf, thumb, imaging.JPEG, imaging.JPEGQuality(85))
	}
	if err != nil {
		log.Printf("face thumbnail: failed to build crop for face %d: %v", face.ID, err)
		w.Header().Del("ETag")
		w.Header().Del("Cache-Control")
		WriteAPIError(w, http.StatusInternalServerError, "ThumbnailError", "could not generate face thumbnail")
		return
	}

	if _, err := store.Put(r.Context(), key, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/jpeg"); err != nil {
		log.Printf("face thumbnail: failed to cache %s: %v", key, err)
	}
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(buf.Bytes()))
}

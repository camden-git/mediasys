package handlers

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"gocv.io/x/gocv"
	"gorm.io/gorm"
)

type ImagePreviewHandler struct {
	FaceRepo  repository.FaceRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
	Store     *media.Store
	Cfg       config.Config
}

// lookupImage resolves the image referenced by the route wildcard.
func (iph *ImagePreviewHandler) lookupImage(w http.ResponseWriter, r *http.Request) (*models.Image, bool) {
	imagePath, ok := wildcardKey(r)
	if !ok {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPath", "invalid image path")
		return nil, false
	}
	img, err := iph.ImageRepo.GetByPath(imagePath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
		} else {
			log.Printf("lookupImage: failed to load %s: %v", imagePath, err)
			WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to load image")
		}
		return nil, false
	}
	return img, true
}

// ServeOriginal streams the original uploaded file.
// Registered at /api/originals/* — the image path is the wildcard segment.
func (iph *ImagePreviewHandler) ServeOriginal(w http.ResponseWriter, r *http.Request) {
	img, ok := iph.lookupImage(w, r)
	if !ok {
		return
	}
	kind := "inline"
	if r.URL.Query().Has("download") {
		kind = "attachment"
	}
	serveObject(w, r, iph.Store, img.ObjectKey, cacheFor(24*time.Hour), contentDisposition(kind, path.Base(img.OriginalPath)))
}

// ServeScaledPreview serves the pre-generated WebP preview, generating one on
// the fly from the original if processing has not finished yet.
// Registered at /api/preview/* — the image path is the wildcard segment.
func (iph *ImagePreviewHandler) ServeScaledPreview(w http.ResponseWriter, r *http.Request) {
	img, ok := iph.lookupImage(w, r)
	if !ok {
		return
	}

	if img.PreviewStatus == database.StatusDone && img.PreviewPath != nil {
		serveObject(w, r, iph.Store, *img.PreviewPath, cacheFor(24*time.Hour), "")
		return
	}

	obj, _, err := iph.Store.Get(r.Context(), img.ObjectKey)
	if err != nil {
		log.Printf("ServeScaledPreview: failed to open original %s: %v", img.ObjectKey, err)
		http.NotFound(w, r)
		return
	}
	defer obj.Close()

	src, err := imaging.Decode(obj, imaging.AutoOrientation(true))
	if err != nil {
		log.Printf("ServeScaledPreview: failed to decode %s: %v", img.OriginalPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}
	imgW, imgH := src.Bounds().Dx(), src.Bounds().Dy()
	if imgW >= imgH && imgW > media.PreviewLandscapeMaxLong {
		src = imaging.Fit(src, media.PreviewLandscapeMaxLong, 99999, imaging.Lanczos)
	} else if imgH > imgW && imgH > media.PreviewPortraitMaxLong {
		src = imaging.Fit(src, 99999, media.PreviewPortraitMaxLong, imaging.Lanczos)
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "no-cache")
	if err := webp.Encode(w, src, &webp.Options{Quality: media.PreviewQuality}); err != nil {
		log.Printf("ServeScaledPreview: encode error for %s: %v", img.OriginalPath, err)
	}
}

// ServeImageWithFaces draws detected face boxes over the original (debug helper).
func (iph *ImagePreviewHandler) ServeImageWithFaces(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimPrefix(r.URL.Query().Get("path"), "/")
	if dbPath == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing 'path' query parameter")
		return
	}
	imgRow, err := iph.ImageRepo.GetByPath(dbPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	obj, _, err := iph.Store.Get(r.Context(), imgRow.ObjectKey)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := io.ReadAll(obj)
	obj.Close()
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "ImageReadError", "Failed to read image")
		return
	}

	img, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil || img.Empty() {
		WriteAPIError(w, http.StatusInternalServerError, "ImageDecodeError", "Failed to decode image")
		return
	}
	defer img.Close()

	faces, err := iph.FaceRepo.ListByImagePath(dbPath)
	if err != nil {
		log.Printf("Error fetching faces for image %s: %v", dbPath, err)
	}

	blue := color.RGBA{B: 255}
	for _, face := range faces {
		rect := image.Rectangle{Min: image.Pt(max(0, face.X1), max(0, face.Y1)), Max: image.Pt(face.X2, face.Y2)}
		gocv.Rectangle(&img, rect, blue, 2)
		label := fmt.Sprintf("ID:%d", face.ID)
		if face.PersonID == nil {
			label = "Untagged"
		}
		gocv.PutText(&img, label, image.Pt(rect.Min.X, rect.Min.Y-5), gocv.FontHersheySimplex, 0.5, blue, 1)
	}

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "ImageEncodeError", "Failed to encode image")
		return
	}
	defer buf.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.GetBytes())
}

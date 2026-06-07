package handlers

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"
	"gocv.io/x/gocv"
	"gorm.io/gorm"
)

type ImagePreviewHandler struct {
	FaceRepo  repository.FaceRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
	Cfg       config.Config
}

func (iph *ImagePreviewHandler) ServeImageWithFaces(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	decodedPath, err := url.QueryUnescape(relativePath)
	if err != nil {
		http.Error(w, "Invalid URL encoding for path parameter", http.StatusBadRequest)
		return
	}
	cleanRelativePath := filepath.Clean(decodedPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		http.Error(w, "Invalid path: must be relative, no '..'", http.StatusBadRequest)
		return
	}
	dbPath := filepath.ToSlash(cleanRelativePath)

	fullPath := filepath.Join(iph.Cfg.RootDirectory, dbPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Printf("Error stating image file %s: %v", fullPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	img := gocv.IMRead(fullPath, gocv.IMReadColor)
	if img.Empty() {
		log.Printf("Failed to read image file with gocv: %s", fullPath)
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}
	defer img.Close()

	faces, err := iph.FaceRepo.ListByImagePath(dbPath)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { // ignore ErrNoRows, just means no faces to draw
		log.Printf("Error fetching faces for image %s: %v", dbPath, err)
		// do not return, proceed to show image without boxes if DB error occurs
	}

	blue := color.RGBA{B: 255} // color.RGBA{0, 0, 255, 0}
	thickness := 2

	if len(faces) > 0 {
		log.Printf("Drawing %d face boxes for %s", len(faces), dbPath)
		for _, face := range faces {
			topLeft := image.Pt(max(0, face.X1), max(0, face.Y1))
			bottomRight := image.Pt(face.X2, face.Y2)
			rect := image.Rectangle{Min: topLeft, Max: bottomRight}

			gocv.Rectangle(&img, rect, blue, thickness)

			label := fmt.Sprintf("ID:%d", face.ID)
			if face.PersonID == nil {
				label = "Untagged"
			}
			gocv.PutText(&img, label, image.Pt(rect.Min.X, rect.Min.Y-5), gocv.FontHersheySimplex, 0.5, blue, 1)
		}
	} else {
		log.Printf("No faces found in DB for %s, serving original", dbPath)
	}

	buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
	if err != nil {
		log.Printf("Error encoding image %s after drawing: %v", dbPath, err)
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}
	defer buf.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	_, err = w.Write(buf.GetBytes())
	if err != nil {
		log.Printf("Error writing image response for %s: %v", dbPath, err)
		// Cannot send error header now, just log
	}
}

// ServeScaledPreview serves a scaled JPEG preview of the requested image.
// If a pre-generated preview exists on disk, it is served directly (fast path).
// Otherwise an on-the-fly preview is generated and streamed (fallback path).
// Registered at /api/preview/* — the image path is the wildcard segment.
func (iph *ImagePreviewHandler) ServeScaledPreview(w http.ResponseWriter, r *http.Request) {
	wildcardPath := chi.URLParam(r, "*")
	if wildcardPath == "" {
		WriteAPIError(w, http.StatusBadRequest, "MissingPath", "image path is required")
		return
	}

	decodedPath, err := url.PathUnescape(wildcardPath)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPath", "invalid URL encoding in path")
		return
	}

	cleanRelativePath := filepath.Clean(decodedPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPath", "path must be relative with no '..'")
		return
	}

	fullPath := filepath.Join(iph.Cfg.RootDirectory, cleanRelativePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Printf("ServeScaledPreview: stat error for %s: %v", fullPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "StatError", "could not stat image file")
		return
	}

	// Fast path: serve pre-generated preview from disk
	if iph.ImageRepo != nil {
		dbPath := filepath.ToSlash(cleanRelativePath)
		imageInfo, dbErr := iph.ImageRepo.GetByPath(dbPath)
		if dbErr == nil && imageInfo != nil &&
			imageInfo.PreviewStatus == "done" && imageInfo.PreviewPath != nil {

			fullPreviewPath := filepath.Join(iph.Cfg.MediaStoragePath, *imageInfo.PreviewPath)
			go func() {
				if touchErr := iph.ImageRepo.TouchPreviewLastRequested(dbPath); touchErr != nil {
					log.Printf("ServeScaledPreview: failed to touch preview timestamp for %s: %v", dbPath, touchErr)
				}
			}()
			w.Header().Set("Cache-Control", "public, max-age=86400")
			http.ServeFile(w, r, fullPreviewPath)
			return
		}
	}

	// Fallback: generate on-the-fly
	src, err := imaging.Open(fullPath, imaging.AutoOrientation(true))
	if err != nil {
		log.Printf("ServeScaledPreview: failed to open image %s: %v", fullPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}

	imgW, imgH := src.Bounds().Dx(), src.Bounds().Dy()
	if imgW >= imgH {
		if imgW > 3400 {
			src = imaging.Fit(src, 3400, 99999, imaging.Lanczos)
		}
	} else {
		if imgH > 2200 {
			src = imaging.Fit(src, 99999, 2200, imaging.Lanczos)
		}
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if err := webp.Encode(w, src, &webp.Options{Quality: 85}); err != nil {
		log.Printf("ServeScaledPreview: encode error for %s: %v", fullPath, err)
	}
}

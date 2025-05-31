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
	"gocv.io/x/gocv"
	"gorm.io/gorm"
)

type ImagePreviewHandler struct {
	FaceRepo repository.FaceRepositoryInterface
	Cfg      config.Config
	// GormDB *gorm.DB
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

	blue := color.RGBA{0, 0, 255, 0}
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

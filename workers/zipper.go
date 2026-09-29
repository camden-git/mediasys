package workers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
)

// CreateAlbumZip streams the originals of images into a ZIP archive stored at
// archiveKey. Nothing touches local disk: objects are read from the store and
// the archive is uploaded as it is written. Returns the archive size.
func CreateAlbumZip(ctx context.Context, store *media.Store, albumFolder string, images []models.Image, archiveKey string) (int64, error) {
	if len(images) == 0 {
		return 0, fmt.Errorf("album has no images to zip")
	}

	pr, pw := io.Pipe()
	written := make(chan error, 1)

	go func() {
		zw := zip.NewWriter(pw)
		added := 0
		var err error
		for _, img := range images {
			obj, info, getErr := store.Get(ctx, img.ObjectKey)
			if getErr != nil {
				// a single missing object shouldn't sink the whole archive
				log.Printf("zipper: skipping %s: %v", img.OriginalPath, getErr)
				continue
			}
			err = addToZip(zw, entryName(albumFolder, img.OriginalPath), obj, info.LastModified)
			obj.Close()
			if err != nil {
				break
			}
			added++
		}
		if err == nil {
			err = zw.Close()
		}
		if err == nil && added == 0 {
			err = fmt.Errorf("no images could be added to the archive")
		}
		written <- err
		pw.CloseWithError(err)
	}()

	size, putErr := store.Put(ctx, archiveKey, pr, -1, "application/zip")
	pr.Close()
	if zipErr := <-written; zipErr != nil {
		_ = store.Delete(ctx, archiveKey)
		return 0, zipErr
	}
	if putErr != nil {
		return 0, putErr
	}
	log.Printf("zipper: created %s (%d bytes, %d images)", archiveKey, size, len(images))
	return size, nil
}

func addToZip(zw *zip.Writer, name string, src io.Reader, modified time.Time) error {
	// photos are already compressed; storing avoids burning CPU for no gain
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store, Modified: modified})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

// entryName returns the path of an image inside the archive, relative to the album folder.
func entryName(albumFolder, imagePath string) string {
	name := strings.TrimPrefix(imagePath, strings.TrimSuffix(albumFolder, "/")+"/")
	return strings.TrimPrefix(name, "/")
}

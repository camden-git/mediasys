package e2e_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"testing"
	"time"
)

// generateJPEG returns a tiny valid JPEG so uploads exercise the real image
// decode/thumbnail/metadata pipeline.
func generateJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("failed to encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

type adminAlbum struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	FolderPath string `json:"folder_path"`
}

// createAlbum creates an album as the given (already-authenticated) actor and
// returns its admin representation.
func createAlbum(t *testing.T, token, name, slug string, sortOrder string) adminAlbum {
	t.Helper()
	payload := map[string]any{
		"name":        name,
		"slug":        slug,
		"folder_path": slug,
	}
	if sortOrder != "" {
		payload["sort_order"] = sortOrder
	}
	resp := doJSON(t, http.MethodPost, "/api/admin/albums", token, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create album %q: %d %s", slug, resp.StatusCode, resp.Body)
	}
	var album adminAlbum
	resp.decodeData(t, &album)
	return album
}

type fileInfo struct {
	Name            string  `json:"name"`
	Path            string  `json:"path"`
	ThumbnailPath   *string `json:"thumbnail_path,omitempty"`
	ThumbnailStatus string  `json:"thumbnail_status,omitempty"`
	MetadataStatus  string  `json:"metadata_status,omitempty"`
}

type directoryListing struct {
	Files   []fileInfo `json:"files"`
	Total   int        `json:"total"`
	Offset  int        `json:"offset"`
	Limit   int        `json:"limit"`
	HasMore bool       `json:"has_more"`
}

// TestAlbumUploadAndProcessing covers: admin creates an album, uploads a small
// generated JPEG, the image appears in the (admin and public) album listing, and
// its original/thumbnail can be fetched once background processing finishes.
func TestAlbumUploadAndProcessing(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	slug := "upload-test-" + suffix

	album := createAlbum(t, env.adminToken, "Upload Test "+suffix, slug, "")

	jpegBytes := generateJPEG(t, 64, 48)
	body, contentType := multipartUpload(t, "photo.jpg", jpegBytes)
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/upload", album.ID), env.adminToken, body, contentType)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", resp.StatusCode, resp.Body)
	}
	var uploadResult struct {
		Uploaded int `json:"uploaded"`
	}
	resp.decodeData(t, &uploadResult)
	if uploadResult.Uploaded != 1 {
		t.Fatalf("expected 1 uploaded file, got %d (body: %s)", uploadResult.Uploaded, resp.Body)
	}

	var processed fileInfo
	pollUntil(t, processingTimeout(), 500*time.Millisecond, "image thumbnail/metadata processing to finish", func() bool {
		listResp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/images", album.ID), env.adminToken, nil, "")
		if listResp.StatusCode != http.StatusOK {
			t.Fatalf("listing images: unexpected status %d: %s", listResp.StatusCode, listResp.Body)
		}
		var listing directoryListing
		listResp.decodeData(t, &listing)
		if len(listing.Files) != 1 {
			return false
		}
		f := listing.Files[0]
		if f.ThumbnailStatus == "done" && f.MetadataStatus == "done" && f.ThumbnailPath != nil {
			processed = f
			return true
		}
		return false
	})

	if processed.ThumbnailPath == nil {
		t.Fatalf("processed image had no thumbnail path")
	}

	t.Run("original is fetchable", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/originals"+processed.Path, "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 fetching original, got %d: %s", resp.StatusCode, resp.Body)
		}
		if len(resp.Body) != len(jpegBytes) {
			t.Fatalf("expected original body length %d, got %d", len(jpegBytes), len(resp.Body))
		}
	})

	t.Run("thumbnail is fetchable", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api"+*processed.ThumbnailPath, "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 fetching thumbnail, got %d: %s", resp.StatusCode, resp.Body)
		}
		if len(resp.Body) == 0 {
			t.Fatalf("expected non-empty thumbnail body")
		}
	})

	t.Run("image appears in the public album contents listing", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/albums/"+slug+"/contents", "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var listing directoryListing
		resp.decodeData(t, &listing)
		if listing.Total != 1 {
			t.Fatalf("expected 1 image in public listing, got %d", listing.Total)
		}
		if len(listing.Files) != 1 || listing.Files[0].Path != processed.Path {
			t.Fatalf("expected public listing to contain %s, got %+v", processed.Path, listing.Files)
		}
	})
}

// TestAlbumListingSortAndPaging is a regression/coverage test for offset/limit paging
// and filename_nat ordering on GET /api/albums/{id}/contents.
func TestAlbumListingSortAndPaging(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	slug := "paging-test-" + suffix

	album := createAlbum(t, env.adminToken, "Paging Test "+suffix, slug, "filename_nat")

	// natural sort should order these as img1, img2, img10 - not lexicographic
	// ("img1", "img10", "img2").
	filenames := []string{"img2.jpg", "img10.jpg", "img1.jpg"}
	for _, fn := range filenames {
		body, contentType := multipartUpload(t, fn, generateJPEG(t, 16, 16))
		resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/upload", album.ID), env.adminToken, body, contentType)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("upload of %s failed: %d %s", fn, resp.StatusCode, resp.Body)
		}
	}

	// sorting/paging works off the images table directly (not the background
	// processing status), so no need to poll - just wait for all 3 rows to exist.
	pollUntil(t, 10*time.Second, 200*time.Millisecond, "all 3 uploads to be recorded", func() bool {
		resp := doRequest(t, http.MethodGet, "/api/albums/"+slug+"/contents?limit=100", "", nil, "")
		if resp.StatusCode != http.StatusOK {
			return false
		}
		var listing directoryListing
		resp.decodeData(t, &listing)
		return listing.Total == 3
	})

	t.Run("natural sort order", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/albums/"+slug+"/contents?limit=100", "", nil, "")
		var listing directoryListing
		resp.decodeData(t, &listing)
		if len(listing.Files) != 3 {
			t.Fatalf("expected 3 files, got %d", len(listing.Files))
		}
		want := []string{"img1.jpg", "img2.jpg", "img10.jpg"}
		for i, f := range listing.Files {
			if f.Name != want[i] {
				t.Fatalf("expected natural sort order %v, got %v (position %d was %q)", want, namesOf(listing.Files), i, f.Name)
			}
		}
	})

	t.Run("offset and limit page through results", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/albums/"+slug+"/contents?limit=2&offset=0", "", nil, "")
		var page1 directoryListing
		resp.decodeData(t, &page1)
		if len(page1.Files) != 2 || !page1.HasMore || page1.Total != 3 {
			t.Fatalf("expected page1 of 2 with has_more=true total=3, got %+v", page1)
		}

		resp = doRequest(t, http.MethodGet, "/api/albums/"+slug+"/contents?limit=2&offset=2", "", nil, "")
		var page2 directoryListing
		resp.decodeData(t, &page2)
		if len(page2.Files) != 1 || page2.HasMore || page2.Total != 3 {
			t.Fatalf("expected page2 of 1 with has_more=false total=3, got %+v", page2)
		}

		if page1.Files[0].Name == page2.Files[0].Name {
			t.Fatalf("expected page1 and page2 to contain different files")
		}
	})
}

func namesOf(files []fileInfo) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	return names
}

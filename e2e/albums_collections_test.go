package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/camden-git/mediasysbackend/models"
)

func TestCollectionNotFoundReturns404(t *testing.T) {
	env := requireShared(t)
	const missing = "/api/admin/collections/999999"

	cases := []struct {
		name, method, path string
		payload            any
	}{
		{"update", http.MethodPut, missing, map[string]any{"name": "x"}},
		{"delete", http.MethodDelete, missing, nil},
		{"inherit", http.MethodPut, missing + "/banners/inherit", map[string]any{"inherit": true}},
		{"filters", http.MethodPut, missing + "/filters", map[string]any{"filters": []any{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, tc.method, tc.path, env.adminToken, tc.payload)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d %s", resp.StatusCode, resp.Body)
			}
			assertErrorShape(t, resp)
		})
	}
}

func TestDeleteAlbumRemovesImagesAndRows(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "Del "+s, "del-"+s, "")
	imgPath := uploadImage(t, env.adminToken, album.ID, "a.jpg")

	resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/albums/%d", album.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete album: %d %s", resp.StatusCode, resp.Body)
	}
	if resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d", album.ID), env.adminToken, nil, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted album 404, got %d", resp.StatusCode)
	}
	var count int64
	if err := env.app.DB.Unscoped().Model(&models.Image{}).Where("original_path = ?", imgPath).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected image rows removed, got %d", count)
	}
	if resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/albums/%d", album.ID), env.adminToken, nil, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected second delete 404, got %d", resp.StatusCode)
	}
}

func TestOriginalsServePercentInFilename(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "Pct "+s, "pct-"+s, "")
	imgPath := uploadImage(t, env.adminToken, album.ID, "100% fun.jpg")
	if !strings.HasSuffix(imgPath, "100% fun.jpg") {
		t.Fatalf("unexpected image path %q", imgPath)
	}

	standard := (&url.URL{Path: "/" + imgPath}).EscapedPath()
	// forces RawPath to be set on the server side: every byte percent-encoded
	var all strings.Builder
	for _, b := range []byte(imgPath) {
		if b == '/' {
			all.WriteByte('/')
			continue
		}
		fmt.Fprintf(&all, "%%%02X", b)
	}
	for name, escaped := range map[string]string{"standard": standard, "fully-encoded": "/" + all.String()} {
		t.Run(name, func(t *testing.T) {
			resp := doRequest(t, http.MethodGet, "/api/originals"+escaped, "", nil, "")
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d %s", resp.StatusCode, resp.Body)
			}
		})
	}
}

func getWithHeaders(t *testing.T, path string, headers map[string]string) apiResponse {
	t.Helper()
	env := requireShared(t)
	req, err := http.NewRequest(http.MethodGet, env.server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return apiResponse{StatusCode: resp.StatusCode, Header: resp.Header, Body: body}
}

func TestPreviewAndOriginalCaching(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "Prev "+s, "prev-"+s, "")
	imgPath := uploadImage(t, env.adminToken, album.ID, "p.jpg")
	previewURL := "/api/preview/" + (&url.URL{Path: imgPath}).EscapedPath()
	originalURL := "/api/originals/" + (&url.URL{Path: imgPath}).EscapedPath()

	pollUntil(t, 30*time.Second, 250*time.Millisecond, "preview to be generated", func() bool {
		return getWithHeaders(t, previewURL, nil).StatusCode == http.StatusOK
	})

	for _, u := range []string{previewURL, originalURL} {
		resp := getWithHeaders(t, u, nil)
		etag := resp.Header.Get("ETag")
		if resp.StatusCode != http.StatusOK || etag == "" {
			t.Fatalf("%s: expected 200 with ETag, got %d etag=%q", u, resp.StatusCode, etag)
		}
		if cc := resp.Header.Get("Cache-Control"); strings.Contains(cc, "max-age=86400") {
			t.Fatalf("%s: cache max-age too long: %q", u, cc)
		}
		cond := getWithHeaders(t, u, map[string]string{"If-None-Match": etag})
		if cond.StatusCode != http.StatusNotModified {
			t.Fatalf("%s: expected 304, got %d", u, cond.StatusCode)
		}
	}

	// a preview that is not ready is a 404, not an on-demand decode of the original
	if err := env.app.DB.Model(&models.Image{}).Where("original_path = ?", imgPath).
		Updates(map[string]any{"preview_status": "error", "preview_path": nil}).Error; err != nil {
		t.Fatal(err)
	}
	resp := getWithHeaders(t, previewURL, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unavailable preview, got %d", resp.StatusCode)
	}
	assertErrorShape(t, resp)
}

func TestPublicListLimitIsClamped(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "Lim "+s, "lim-"+s, "")

	resp := doRequest(t, http.MethodGet, "/api/albums/"+album.Slug+"/contents?limit=100000", "", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contents: %d %s", resp.StatusCode, resp.Body)
	}
	var listing directoryListing
	resp.decode(t, &listing)
	if listing.Limit != 500 {
		t.Fatalf("expected limit clamped to 500, got %d", listing.Limit)
	}
}

func TestBannerUploadBodyIsCapped(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "Cap "+s, "cap-"+s, "")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("banner_image", "big.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(make([]byte, 21<<20)); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/banners", album.ID), env.adminToken, &buf, mw.FormDataContentType())
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d %s", resp.StatusCode, resp.Body)
	}
	assertErrorShape(t, resp)
}

func TestAlbumGroupAssignmentAndDeletion(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	album := createAlbum(t, env.adminToken, "GA "+s, "ga-"+s, "")

	resp := doJSON(t, http.MethodPost, "/api/admin/groups", env.adminToken, map[string]any{"name": "G " + s, "slug": "g-" + s})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create group: %d %s", resp.StatusCode, resp.Body)
	}
	var group struct {
		ID uint `json:"id"`
	}
	resp.decodeData(t, &group)

	albumGroupURL := fmt.Sprintf("/api/admin/albums/%d/group", album.ID)

	// unknown group / album
	if resp := doJSON(t, http.MethodPut, albumGroupURL, env.adminToken, map[string]any{"group_id": 999999}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown group: expected 404, got %d %s", resp.StatusCode, resp.Body)
	}
	if resp := doJSON(t, http.MethodPut, "/api/admin/albums/999999/group", env.adminToken, map[string]any{"group_id": group.ID}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown album: expected 404, got %d %s", resp.StatusCode, resp.Body)
	}

	if resp := doJSON(t, http.MethodPut, albumGroupURL, env.adminToken, map[string]any{"group_id": group.ID}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("assign: %d %s", resp.StatusCode, resp.Body)
	}

	// give the group a banner, then delete it
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("banner_image", "b.jpg")
	_, _ = part.Write(generateJPEG(t, 64, 32))
	_ = mw.Close()
	resp = doRequest(t, http.MethodPut, fmt.Sprintf("/api/admin/groups/%d/banner", group.ID), env.adminToken, &buf, mw.FormDataContentType())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("banner: %d %s", resp.StatusCode, resp.Body)
	}
	var withBanner struct {
		BannerImagePath *string `json:"banner_image_path"`
	}
	resp.decodeData(t, &withBanner)
	if withBanner.BannerImagePath == nil {
		t.Fatalf("expected banner path in %s", resp.Body)
	}
	if _, err := env.app.Store.Stat(context.Background(), *withBanner.BannerImagePath); err != nil {
		t.Fatalf("banner object should exist: %v", err)
	}

	if resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/groups/%d", group.ID), env.adminToken, nil, ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete group: %d %s", resp.StatusCode, resp.Body)
	}
	if resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/groups/%d", group.ID), env.adminToken, nil, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete: expected 404, got %d", resp.StatusCode)
	}

	var a models.Album
	if err := env.app.DB.First(&a, album.ID).Error; err != nil {
		t.Fatal(err)
	}
	if a.GroupID != nil {
		t.Fatalf("expected album group_id cleared, got %d", *a.GroupID)
	}
	if _, err := env.app.Store.Stat(context.Background(), *withBanner.BannerImagePath); err == nil {
		t.Fatal("expected group banner object to be deleted")
	}
}

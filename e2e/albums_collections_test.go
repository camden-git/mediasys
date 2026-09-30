package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

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

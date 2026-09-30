package e2e_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestPublicAlbumOmitsZipInternals(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	slug := "pub-" + s
	createAlbum(t, env.adminToken, "Pub "+s, slug, "")

	for _, path := range []string{"/api/albums/" + slug, "/api/albums"} {
		resp := doRequest(t, http.MethodGet, path, "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d %s", path, resp.StatusCode, resp.Body)
		}
		body := string(resp.Body)
		for _, banned := range []string{"zip_path", "zip_error", "deleted_at"} {
			if strings.Contains(body, banned) {
				t.Fatalf("%s: response must not contain %s: %s", path, banned, body)
			}
		}
		if !strings.Contains(body, `"zip_ready":false`) {
			t.Fatalf("%s: expected zip_ready:false in %s", path, body)
		}
	}

	if resp := doJSON(t, http.MethodPost, "/api/admin/albums", env.adminToken, map[string]any{"name": "Digits " + s, "slug": "12345"}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("all-digit slug: expected 400, got %d %s", resp.StatusCode, resp.Body)
	}
	if resp := doJSON(t, http.MethodPost, "/api/admin/albums", env.adminToken, map[string]any{"name": "Sort " + s, "slug": "sort-" + s, "sort_order": "bogus"}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad sort order: expected 400, got %d %s", resp.StatusCode, resp.Body)
	}
}

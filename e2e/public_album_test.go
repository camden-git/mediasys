package e2e_test

import (
	"fmt"
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

// TestHiddenAlbumNotReachableByID verifies hidden (unlisted) albums resolve on the public
// routes only by slug, so they cannot be enumerated by numeric ID, except for users with
// view access to them.
func TestHiddenAlbumNotReachableByID(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()

	hidden := createAlbum(t, env.adminToken, "Unlisted "+s, "unlisted-"+s, "")
	secCleanupDelete(t, fmt.Sprintf("/api/admin/albums/%d", hidden.ID))
	if resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/albums/%d", hidden.ID), env.adminToken, map[string]any{"is_hidden": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("hide album: %d %s", resp.StatusCode, resp.Body)
	}
	uploadImage(t, env.adminToken, hidden.ID, "unlisted.jpg")

	visible := createAlbum(t, env.adminToken, "Listed "+s, "listed-"+s, "")
	secCleanupDelete(t, fmt.Sprintf("/api/admin/albums/%d", visible.ID))

	byID := fmt.Sprintf("/api/albums/%d", hidden.ID)
	bySlug := "/api/albums/" + hidden.Slug

	t.Run("anonymous lookup by ID is not found", func(t *testing.T) {
		for _, path := range []string{byID, byID + "/contents", byID + "/zip"} {
			resp := doRequest(t, http.MethodGet, path, "", nil, "")
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("%s: expected 404, got %d %s", path, resp.StatusCode, resp.Body)
			}
			if body := assertErrorShape(t, resp); body.Errors[0].Code != "AlbumNotFound" {
				t.Fatalf("%s: expected AlbumNotFound, got %s", path, resp.Body)
			}
		}
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/share/albums/%d", hidden.ID), "", nil, "")
		if resp.StatusCode != http.StatusNotFound || strings.Contains(string(resp.Body), hidden.Slug) {
			t.Fatalf("share by ID: expected 404 without the slug, got %d %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("anonymous lookup by slug works", func(t *testing.T) {
		for _, path := range []string{bySlug, bySlug + "/contents", "/api/share/albums/" + hidden.Slug} {
			if resp := doRequest(t, http.MethodGet, path, "", nil, ""); resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d %s", path, resp.StatusCode, resp.Body)
			}
		}
		var listing directoryListing
		doRequest(t, http.MethodGet, bySlug+"/contents", "", nil, "").decodeData(t, &listing)
		if listing.Total != 1 {
			t.Fatalf("expected 1 image by slug, got %d", listing.Total)
		}
	})

	t.Run("visible album still resolves by ID", func(t *testing.T) {
		path := fmt.Sprintf("/api/albums/%d", visible.ID)
		if resp := doRequest(t, http.MethodGet, path, "", nil, ""); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d %s", path, resp.StatusCode, resp.Body)
		}
	})

	t.Run("users with view access can look up by ID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, byID, env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("admin: expected 200, got %d %s", resp.StatusCode, resp.Body)
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "private") {
			t.Fatalf("privileged lookup must not be publicly cacheable, got Cache-Control %q", cc)
		}

		viewer := secMakeCleanUser(t, "hid_viewer_"+s, nil, nil)
		grantAlbumPermission(t, env.adminToken, hidden.ID, viewer.ID, []string{"album.view.content"})
		if resp := doRequest(t, http.MethodGet, byID+"/contents", secLoginAs(t, viewer.Username), nil, ""); resp.StatusCode != http.StatusOK {
			t.Fatalf("album viewer: expected 200, got %d %s", resp.StatusCode, resp.Body)
		}

		outsider := secMakeCleanUser(t, "hid_outsider_"+s, nil, nil)
		grantAlbumPermission(t, env.adminToken, visible.ID, outsider.ID, []string{"album.view.content"})
		if resp := doRequest(t, http.MethodGet, byID, secLoginAs(t, outsider.Username), nil, ""); resp.StatusCode != http.StatusNotFound {
			t.Fatalf("user without access: expected 404, got %d %s", resp.StatusCode, resp.Body)
		}
	})
}

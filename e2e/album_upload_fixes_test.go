package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/camden-git/mediasysbackend/models"
)

type uploadResult struct {
	Uploaded int `json:"uploaded"`
	Failed   []struct {
		Path  string `json:"path"`
		Error string `json:"error"`
	} `json:"failed"`
}

// multipartUploadWithPath uploads one file with an explicit relative_path. The
// server strips the first path segment (the browser's top-level folder).
func multipartUploadWithPath(t *testing.T, relPath, filename string, data []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("relative_path", relPath); err != nil {
		t.Fatalf("failed to write relative_path: %v", err)
	}
	part, err := w.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("failed to create multipart field: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write multipart data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

func uploadWithPath(t *testing.T, albumID uint, relPath string, data []byte) uploadResult {
	t.Helper()
	env := requireShared(t)
	body, contentType := multipartUploadWithPath(t, relPath, "x.jpg", data)
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/upload", albumID), env.adminToken, body, contentType)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", resp.StatusCode, resp.Body)
	}
	var res uploadResult
	resp.decodeData(t, &res)
	return res
}

func createAlbumWithFolder(t *testing.T, name, slug, folder string) apiResponse {
	t.Helper()
	env := requireShared(t)
	return doJSON(t, http.MethodPost, "/api/admin/albums", env.adminToken, map[string]any{
		"name": name, "slug": slug, "folder_path": folder,
	})
}

func TestGroupDetailHidesHiddenAlbums(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	visible := createAlbum(t, env.adminToken, "Vis "+s, "vis-"+s, "")
	hidden := createAlbum(t, env.adminToken, "Hid "+s, "hid-"+s, "")
	if resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/albums/%d", hidden.ID), env.adminToken, map[string]any{"is_hidden": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("hide album: %d %s", resp.StatusCode, resp.Body)
	}

	resp := doJSON(t, http.MethodPost, "/api/admin/groups", env.adminToken, map[string]any{"name": "Grp " + s, "slug": "grp-" + s})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create group: %d %s", resp.StatusCode, resp.Body)
	}
	var group struct {
		ID uint `json:"id"`
	}
	resp.decodeData(t, &group)
	for _, a := range []adminAlbum{visible, hidden} {
		r := doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/albums/%d/group", a.ID), env.adminToken, map[string]any{"group_id": group.ID})
		if r.StatusCode >= 300 {
			t.Fatalf("assign group: %d %s", r.StatusCode, r.Body)
		}
	}

	resp = doRequest(t, http.MethodGet, "/api/groups/grp-"+s, "", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get group: %d %s", resp.StatusCode, resp.Body)
	}
	var got struct {
		Albums []adminAlbum `json:"albums"`
	}
	resp.decodeData(t, &got)
	if len(got.Albums) != 1 || got.Albums[0].ID != visible.ID {
		t.Fatalf("expected only the visible album, got %+v", got.Albums)
	}
}

func TestOverlappingAlbumFoldersRejected(t *testing.T) {
	s := randomSuffix()
	base := "fold-" + s

	if resp := createAlbumWithFolder(t, "Base "+s, "base-"+s, base+"/a"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create base album: %d %s", resp.StatusCode, resp.Body)
	}
	for i, folder := range []string{base + "/a/b", base, base + "/a"} {
		resp := createAlbumWithFolder(t, fmt.Sprintf("Dup%d %s", i, s), fmt.Sprintf("dup%d-%s", i, s), folder)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("folder %q: expected 409, got %d %s", folder, resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	}
	// a sibling sharing a name prefix is fine
	if resp := createAlbumWithFolder(t, "Sib "+s, "sib-"+s, base+"/ab"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("sibling album should be allowed: %d %s", resp.StatusCode, resp.Body)
	}
}

func TestUploadCannotOverwriteOtherAlbum(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	a := createAlbum(t, env.adminToken, "Own A "+s, "own-a-"+s, "")
	b := createAlbum(t, env.adminToken, "Own B "+s, "own-b-"+s, "")

	// force a legacy-style nesting: B's folder lives inside A's folder
	if err := env.app.DB.Model(&models.Album{}).Where("id = ?", b.ID).Update("folder_path", a.FolderPath+"/b").Error; err != nil {
		t.Fatalf("failed to nest album folder: %v", err)
	}

	original := generateJPEG(t, 32, 32)
	if res := uploadWithPath(t, b.ID, "top/x.jpg", original); res.Uploaded != 1 {
		t.Fatalf("upload to B failed: %+v", res)
	}

	// A uploading b/x.jpg resolves to a/b/x.jpg, which B owns
	res := uploadWithPath(t, a.ID, "top/b/x.jpg", generateJPEG(t, 16, 16))
	if res.Uploaded != 0 || len(res.Failed) != 1 {
		t.Fatalf("expected the colliding upload to fail, got %+v", res)
	}

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/images", b.ID), env.adminToken, nil, "")
	var listing directoryListing
	resp.decodeData(t, &listing)
	if len(listing.Files) != 1 {
		t.Fatalf("B should still own its image, got %+v", listing.Files)
	}
	orig := doRequest(t, http.MethodGet, "/api/originals"+listing.Files[0].Path, "", nil, "")
	if orig.StatusCode != http.StatusOK || !bytes.Equal(orig.Body, original) {
		t.Fatalf("B's original was modified: status %d", orig.StatusCode)
	}
}

func TestReuploadReplacesOriginal(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	a := createAlbum(t, env.adminToken, "Reup "+s, "reup-"+s, "")

	first := generateJPEG(t, 32, 32)
	second := generateJPEG(t, 48, 24)
	for _, data := range [][]byte{first, second} {
		if res := uploadWithPath(t, a.ID, "top/p.jpg", data); res.Uploaded != 1 {
			t.Fatalf("upload failed: %+v", res)
		}
	}

	orig := doRequest(t, http.MethodGet, "/api/originals/"+a.FolderPath+"/p.jpg", "", nil, "")
	if orig.StatusCode != http.StatusOK || !bytes.Equal(orig.Body, second) {
		t.Fatalf("expected the re-uploaded original to be served, status %d", orig.StatusCode)
	}
}

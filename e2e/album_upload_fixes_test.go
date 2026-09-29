package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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

	resp = doRequest(t, http.MethodGet, "/api/admin/albums/"+visible.Slug, env.adminToken, nil, "")
	var adm struct {
		GroupID *uint `json:"group_id"`
	}
	resp.decodeData(t, &adm)
	if adm.GroupID == nil || *adm.GroupID != group.ID {
		t.Fatalf("expected admin album group_id %d, got %v", group.ID, adm.GroupID)
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

func TestZipInvalidatedWhenImagesChange(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	a := createAlbum(t, env.adminToken, "Zip "+s, "zip-"+s, "")
	zipURL := fmt.Sprintf("/api/admin/albums/%d/zip", a.ID)

	for _, name := range []string{"top/one.jpg", "top/two.jpg"} {
		if res := uploadWithPath(t, a.ID, name, generateJPEG(t, 24, 24)); res.Uploaded != 1 {
			t.Fatalf("upload failed: %+v", res)
		}
	}
	requestZip := func() {
		if resp := doRequest(t, http.MethodPost, zipURL, env.adminToken, nil, ""); resp.StatusCode != http.StatusAccepted {
			t.Fatalf("zip request: %d %s", resp.StatusCode, resp.Body)
		}
		pollUntil(t, 20*time.Second, 200*time.Millisecond, "zip to be ready", func() bool {
			return doRequest(t, http.MethodGet, zipURL, env.adminToken, nil, "").StatusCode == http.StatusOK
		})
	}

	requestZip()
	del := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/albums/%d/images?path=%s/one.jpg", a.ID, a.FolderPath), env.adminToken, nil, "")
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete image: %d %s", del.StatusCode, del.Body)
	}
	if resp := doRequest(t, http.MethodGet, zipURL, env.adminToken, nil, ""); resp.StatusCode == http.StatusOK {
		t.Fatalf("stale zip still served after image delete")
	}

	requestZip()
	if res := uploadWithPath(t, a.ID, "top/three.jpg", generateJPEG(t, 24, 24)); res.Uploaded != 1 {
		t.Fatalf("upload failed: %+v", res)
	}
	if resp := doRequest(t, http.MethodGet, zipURL, env.adminToken, nil, ""); resp.StatusCode == http.StatusOK {
		t.Fatalf("stale zip still served after upload")
	}
}

func TestPerAlbumPermissionBySlug(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	a := createAlbum(t, env.adminToken, "Slug A "+s, "slug-a-"+s, "")
	b := createAlbum(t, env.adminToken, "Slug B "+s, "slug-b-"+s, "")

	username, password := "slugu_"+s, "test-password-"+s
	userID := createUser(t, env.adminToken, username, password)
	grantAlbumPermission(t, env.adminToken, a.ID, userID, []string{"album.view.content"})
	token, err := login(env.server.URL, username, password)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	resp := doRequest(t, http.MethodGet, "/api/admin/albums/"+a.Slug, token, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for granted album by slug, got %d: %s", resp.StatusCode, resp.Body)
	}
	var got adminAlbum
	resp.decodeData(t, &got)
	if got.ID != a.ID {
		t.Fatalf("expected album %d, got %d", a.ID, got.ID)
	}

	resp = doRequest(t, http.MethodGet, "/api/admin/albums/"+b.Slug, token, nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for other album by slug, got %d: %s", resp.StatusCode, resp.Body)
	}
	assertErrorShape(t, resp)

	resp = doRequest(t, http.MethodGet, "/api/admin/albums/no-such-slug-"+s, token, nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown slug, got %d: %s", resp.StatusCode, resp.Body)
	}
}

// wsCollect connects to /api/ws and collects decoded events until closed.
func wsCollect(t *testing.T, token string) (events func() []map[string]any, closeFn func()) {
	t.Helper()
	env := requireShared(t)
	url := "ws" + strings.TrimPrefix(env.server.URL, "http") + "/api/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Sec-WebSocket-Protocol": {"bearer, " + token}})
	if err != nil {
		t.Fatalf("ws dial failed: %v", err)
	}
	var mu sync.Mutex
	var got []map[string]any
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev map[string]any
			if json.Unmarshal(data, &ev) == nil {
				mu.Lock()
				got = append(got, ev)
				mu.Unlock()
			}
		}
	}()
	return func() []map[string]any {
			mu.Lock()
			defer mu.Unlock()
			return append([]map[string]any(nil), got...)
		}, func() {
			conn.Close()
		}
}

func TestWebSocketEventsAreScopedByAlbum(t *testing.T) {
	env := requireShared(t)
	s := randomSuffix()
	a := createAlbum(t, env.adminToken, "WS A "+s, "ws-a-"+s, "")
	b := createAlbum(t, env.adminToken, "WS B "+s, "ws-b-"+s, "")

	scopedName, nobodyName, pw := "wsscoped_"+s, "wsnobody_"+s, "test-password-"+s
	scopedID := createUser(t, env.adminToken, scopedName, pw)
	createUser(t, env.adminToken, nobodyName, pw)
	grantAlbumPermission(t, env.adminToken, a.ID, scopedID, []string{"album.view.content"})
	scopedToken, err := login(env.server.URL, scopedName, pw)
	if err != nil {
		t.Fatal(err)
	}
	nobodyToken, err := login(env.server.URL, nobodyName, pw)
	if err != nil {
		t.Fatal(err)
	}

	adminEvents, closeAdmin := wsCollect(t, env.adminToken)
	defer closeAdmin()
	scopedEvents, closeScoped := wsCollect(t, scopedToken)
	defer closeScoped()
	nobodyEvents, closeNobody := wsCollect(t, nobodyToken)
	defer closeNobody()
	time.Sleep(200 * time.Millisecond) // let the hub register the clients

	uploadWithPath(t, a.ID, "top/a.jpg", generateJPEG(t, 16, 16))
	uploadWithPath(t, b.ID, "top/b.jpg", generateJPEG(t, 16, 16))

	albumIDs := func(evs []map[string]any) map[float64]bool {
		ids := map[float64]bool{}
		for _, ev := range evs {
			if id, ok := ev["album_id"].(float64); ok {
				ids[id] = true
			}
		}
		return ids
	}
	pollUntil(t, 10*time.Second, 100*time.Millisecond, "admin to see both albums", func() bool {
		ids := albumIDs(adminEvents())
		return ids[float64(a.ID)] && ids[float64(b.ID)]
	})
	pollUntil(t, 10*time.Second, 100*time.Millisecond, "scoped user to see album A", func() bool {
		return albumIDs(scopedEvents())[float64(a.ID)]
	})
	time.Sleep(500 * time.Millisecond)
	if albumIDs(scopedEvents())[float64(b.ID)] {
		t.Fatalf("scoped user received events for an album they cannot view")
	}
	if n := len(nobodyEvents()); n != 0 {
		t.Fatalf("user without permissions received %d events", n)
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

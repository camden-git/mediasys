package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type faceBody struct {
	ID       uint  `json:"id"`
	PersonID *uint `json:"person_id"`
}

type personBody struct {
	ID             uint       `json:"id"`
	KeyPhotoFaceID *uint      `json:"key_photo_face_id"`
	Faces          []faceBody `json:"faces"`
}

// uploadImage uploads a generated JPEG into the album and returns its logical image path
// (without a leading slash), waiting until the image shows up in the admin listing.
func uploadImage(t *testing.T, token string, albumID uint, filename string) string {
	t.Helper()
	body, contentType := multipartUpload(t, filename, generateJPEG(t, 64, 48))
	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/upload", albumID), token, body, contentType)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", resp.StatusCode, resp.Body)
	}
	var path string
	pollUntil(t, 30*time.Second, 250*time.Millisecond, "uploaded image to be listed", func() bool {
		listResp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/images", albumID), token, nil, "")
		if listResp.StatusCode != http.StatusOK {
			return false
		}
		var listing directoryListing
		listResp.decodeData(t, &listing)
		if len(listing.Files) != 1 {
			return false
		}
		path = listing.Files[0].Path
		return true
	})
	if path != "" && path[0] == '/' {
		path = path[1:]
	}
	return path
}

func createPerson(t *testing.T, token, name string, aliases ...string) personBody {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/people", token, map[string]any{"primary_name": name, "aliases": aliases})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create person: %d %s", resp.StatusCode, resp.Body)
	}
	var p personBody
	resp.decode(t, &p)
	return p
}

func addFace(t *testing.T, token, imagePath string, personID *uint) faceBody {
	t.Helper()
	payload := map[string]any{"image_path": imagePath, "x1": 5, "y1": 5, "x2": 30, "y2": 30}
	if personID != nil {
		payload["person_id"] = *personID
	}
	resp := doJSON(t, http.MethodPost, "/api/images/faces", token, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to add face: %d %s", resp.StatusCode, resp.Body)
	}
	var f faceBody
	resp.decode(t, &f)
	return f
}

func getPerson(t *testing.T, id uint) personBody {
	t.Helper()
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/people/%d", id), "", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get person %d: %d %s", id, resp.StatusCode, resp.Body)
	}
	var p personBody
	resp.decode(t, &p)
	return p
}

func setKeyPhoto(t *testing.T, token string, personID, faceID uint) {
	t.Helper()
	resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/people/%d/key-photo", personID), token, map[string]any{"face_id": faceID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to set key photo: %d %s", resp.StatusCode, resp.Body)
	}
}

// TestAdminFaceToolingRequiresFaceManage: the untagged queue, similar/suggest and
// single-face lookups are admin tooling and must require the face.manage permission.
func TestAdminFaceToolingRequiresFaceManage(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	username := "noface" + suffix
	createUser(t, env.adminToken, username, "password-"+suffix)
	userToken, err := login(env.server.URL, username, "password-"+suffix)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	paths := []string{"/api/faces/untagged", "/api/faces/1", "/api/faces/1/similar", "/api/faces/1/suggest"}
	for _, path := range paths {
		t.Run("anonymous "+path, func(t *testing.T) {
			resp := doRequest(t, http.MethodGet, path, "", nil, "")
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", resp.StatusCode, resp.Body)
			}
		})
		t.Run("no permission "+path, func(t *testing.T) {
			resp := doRequest(t, http.MethodGet, path, userToken, nil, "")
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403, got %d: %s", resp.StatusCode, resp.Body)
			}
		})
		t.Run("admin "+path, func(t *testing.T) {
			resp := doRequest(t, http.MethodGet, path, env.adminToken, nil, "")
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				t.Fatalf("admin should be allowed, got %d: %s", resp.StatusCode, resp.Body)
			}
		})
	}

	t.Run("public face routes stay public", func(t *testing.T) {
		for _, path := range []string{"/api/images/faces?path=nope.jpg", "/api/people", "/api/search/faces?query=zzz"} {
			resp := doRequest(t, http.MethodGet, path, "", nil, "")
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", path, resp.StatusCode, resp.Body)
			}
		}
	})
}

// TestHiddenAlbumFacesExcludedFromPublicPeople: images in hidden albums must not show
// up in the public person detail or the public face search.
func TestHiddenAlbumFacesExcludedFromPublicPeople(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	visible := createAlbum(t, env.adminToken, "Visible "+suffix, "vis-"+suffix, "")
	hidden := createAlbum(t, env.adminToken, "Hidden "+suffix, "hid-"+suffix, "")
	resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/albums/%d", hidden.ID), env.adminToken, map[string]any{"is_hidden": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to hide album: %d %s", resp.StatusCode, resp.Body)
	}

	visiblePath := uploadImage(t, env.adminToken, visible.ID, "a.jpg")
	hiddenPath := uploadImage(t, env.adminToken, hidden.ID, "b.jpg")

	name := "Hideme" + suffix
	person := createPerson(t, env.adminToken, name)
	visibleFace := addFace(t, env.adminToken, visiblePath, &person.ID)
	addFace(t, env.adminToken, hiddenPath, &person.ID)

	t.Run("person detail", func(t *testing.T) {
		p := getPerson(t, person.ID)
		if len(p.Faces) != 1 || p.Faces[0].ID != visibleFace.ID {
			t.Fatalf("expected only the visible-album face, got %+v", p.Faces)
		}
	})

	t.Run("face search", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/search/faces?query="+url.QueryEscape(name), "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var results []struct {
			ImagePath string `json:"image_path"`
		}
		resp.decode(t, &results)
		if len(results) != 1 || results[0].ImagePath != visiblePath {
			t.Fatalf("expected only %q, got %+v", visiblePath, results)
		}
	})

	t.Run("direct lightbox face lookup still works for hidden album", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/images/faces?path="+url.QueryEscape(hiddenPath), "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var faces []faceBody
		resp.decode(t, &faces)
		if len(faces) != 1 {
			t.Fatalf("expected 1 face on hidden image, got %d", len(faces))
		}
	})
}

// TestDeletePersonCleansUp: deleting a person unassigns their faces and removes their
// aliases so search no longer matches them.
func TestDeletePersonCleansUp(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "Del "+suffix, "del-"+suffix, "")
	imagePath := uploadImage(t, env.adminToken, album.ID, "a.jpg")

	alias := "Zaliasdel" + suffix
	person := createPerson(t, env.adminToken, "Deleteme"+suffix, alias)
	face := addFace(t, env.adminToken, imagePath, &person.ID)

	search := func() int {
		resp := doRequest(t, http.MethodGet, "/api/search/faces?query="+url.QueryEscape(alias), "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("search failed: %d %s", resp.StatusCode, resp.Body)
		}
		var results []map[string]any
		resp.decode(t, &results)
		return len(results)
	}
	if n := search(); n != 1 {
		t.Fatalf("expected alias search to match before delete, got %d results", n)
	}

	resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/people/%d", person.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete person failed: %d %s", resp.StatusCode, resp.Body)
	}

	if n := search(); n != 0 {
		t.Fatalf("expected alias search to be empty after delete, got %d results", n)
	}

	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/api/faces/%d", face.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get face failed: %d %s", resp.StatusCode, resp.Body)
	}
	var got faceBody
	resp.decode(t, &got)
	if got.PersonID != nil {
		t.Fatalf("expected face to be unassigned after person delete, person_id=%d", *got.PersonID)
	}

	resp = doRequest(t, http.MethodDelete, fmt.Sprintf("/api/people/%d", person.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 deleting missing person, got %d", resp.StatusCode)
	}
}

// TestKeyPhotoClearedWhenFaceChanges: people.key_photo_face_id must not keep pointing at a
// face that was deleted, untagged, or retagged to someone else.
func TestKeyPhotoClearedWhenFaceChanges(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "Key "+suffix, "key-"+suffix, "")
	imagePath := uploadImage(t, env.adminToken, album.ID, "a.jpg")

	keyPhotoStatus := func(personID uint) int {
		return doRequest(t, http.MethodGet, fmt.Sprintf("/api/people/%d/key-photo.jpg", personID), "", nil, "").StatusCode
	}

	setup := func(name string) (personBody, faceBody) {
		p := createPerson(t, env.adminToken, name+suffix)
		f := addFace(t, env.adminToken, imagePath, &p.ID)
		setKeyPhoto(t, env.adminToken, p.ID, f.ID)
		if got := getPerson(t, p.ID); got.KeyPhotoFaceID == nil || *got.KeyPhotoFaceID != f.ID {
			t.Fatalf("key photo not set: %+v", got)
		}
		return p, f
	}

	t.Run("deleted", func(t *testing.T) {
		p, f := setup("KeyDel")
		resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/faces/%d", f.ID), env.adminToken, nil, "")
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("delete face failed: %d %s", resp.StatusCode, resp.Body)
		}
		if got := getPerson(t, p.ID); got.KeyPhotoFaceID != nil {
			t.Fatalf("expected key photo cleared, got %d", *got.KeyPhotoFaceID)
		}
	})

	t.Run("untagged", func(t *testing.T) {
		p, f := setup("KeyUntag")
		resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/faces/%d", f.ID), env.adminToken, map[string]any{"person_id": nil})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("untag face failed: %d %s", resp.StatusCode, resp.Body)
		}
		if got := getPerson(t, p.ID); got.KeyPhotoFaceID != nil {
			t.Fatalf("expected key photo cleared, got %d", *got.KeyPhotoFaceID)
		}
		if code := keyPhotoStatus(p.ID); code != http.StatusNotFound {
			t.Fatalf("expected 404 key photo, got %d", code)
		}
	})

	t.Run("retagged", func(t *testing.T) {
		a, f := setup("KeyA")
		b := createPerson(t, env.adminToken, "KeyB"+suffix)
		resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/faces/%d", f.ID), env.adminToken, map[string]any{"person_id": b.ID})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("retag face failed: %d %s", resp.StatusCode, resp.Body)
		}
		if got := getPerson(t, a.ID); got.KeyPhotoFaceID != nil {
			t.Fatalf("expected old person's key photo cleared, got %d", *got.KeyPhotoFaceID)
		}
		if code := keyPhotoStatus(a.ID); code != http.StatusNotFound {
			t.Fatalf("expected 404 key photo for old person, got %d", code)
		}
	})
}

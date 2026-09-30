package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/camden-git/mediasysbackend/models"
)

// unitVec returns a 512-d vector along axis, nudged by tilt on the next axis.
func unitVec(axis int, tilt float32) []float32 {
	v := make([]float32, models.FaceEmbeddingDimension)
	v[axis] = 1
	v[(axis+1)%len(v)] = tilt
	return v
}

func addEmbedding(t *testing.T, faceID uint, vec []float32) {
	t.Helper()
	env := requireShared(t)
	e := &models.FaceEmbedding{FaceID: faceID, EmbeddingModel: "arcface"}
	e.SetEmbedding(vec)
	if err := env.app.DB.Create(e).Error; err != nil {
		t.Fatalf("failed to create embedding: %v", err)
	}
}

func setImageDims(t *testing.T, imagePath string, w, h int) {
	t.Helper()
	env := requireShared(t)
	if err := env.app.DB.Model(&models.Image{}).Where("original_path = ?", imagePath).
		Updates(map[string]any{"width": w, "height": h}).Error; err != nil {
		t.Fatalf("failed to set image dims: %v", err)
	}
}

type untaggedBody struct {
	FaceID            uint  `json:"face_id"`
	SuggestedPersonID *uint `json:"suggested_person_id"`
	SuggestionCount   int   `json:"suggestion_count"`
	SimilarFacesCount int   `json:"similar_faces_count"`
	ImageWidth        int   `json:"image_width"`
	ImageHeight       int   `json:"image_height"`
}

func listUntagged(t *testing.T, token, query string) []untaggedBody {
	t.Helper()
	resp := doRequest(t, http.MethodGet, "/api/faces/untagged?"+query, token, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("untagged failed: %d %s", resp.StatusCode, resp.Body)
	}
	var out []untaggedBody
	resp.decodeData(t, &out)
	return out
}

func TestUntaggedFacesSQLFilteringAndSuggestions(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	albumA := createAlbum(t, env.adminToken, "UA "+suffix, "ua-"+suffix, "")
	albumB := createAlbum(t, env.adminToken, "UB "+suffix, "ub-"+suffix, "")
	imgA := uploadImage(t, env.adminToken, albumA.ID, "a.jpg")
	imgB := uploadImage(t, env.adminToken, albumB.ID, "b.jpg")
	setImageDims(t, imgA, 64, 48)
	setImageDims(t, imgB, 64, 48)

	axis := 100
	confirmedPerson := createPerson(t, env.adminToken, "Confirmed"+suffix)
	autoPerson := createPerson(t, env.adminToken, "Auto"+suffix)

	// confirmed tag via PUT
	fc := addFace(t, env.adminToken, imgB, nil)
	resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/faces/%d", fc.ID), env.adminToken, map[string]any{"person_id": confirmedPerson.ID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tag via PUT failed: %d %s", resp.StatusCode, resp.Body)
	}
	var tagged struct {
		Confirmed bool `json:"confirmed"`
	}
	resp.decodeData(t, &tagged)
	if !tagged.Confirmed {
		t.Fatalf("manual PUT tag should be confirmed: %s", resp.Body)
	}
	addEmbedding(t, fc.ID, unitVec(axis, 0.10))

	// two unconfirmed faces of another person: more votes, but they must not count
	for i := 0; i < 2; i++ {
		f := addFace(t, env.adminToken, imgB, &autoPerson.ID)
		addEmbedding(t, f.ID, unitVec(axis, 0.01*float32(i+1)))
	}

	// two untagged faces on image A, one on image B
	u1 := addFace(t, env.adminToken, imgA, nil)
	u2 := addFace(t, env.adminToken, imgA, nil)
	u3 := addFace(t, env.adminToken, imgB, nil)
	addEmbedding(t, u1.ID, unitVec(axis, 0))
	addEmbedding(t, u2.ID, unitVec(axis, 0.02))
	addEmbedding(t, u3.ID, unitVec(axis, 0.03))

	byID := func(items []untaggedBody) map[uint]untaggedBody {
		m := map[uint]untaggedBody{}
		for _, it := range items {
			m[it.FaceID] = it
		}
		return m
	}

	t.Run("album filter", func(t *testing.T) {
		got := byID(listUntagged(t, env.adminToken, fmt.Sprintf("album_id=%d&limit=100", albumA.ID)))
		if len(got) != 2 || got[u1.ID].FaceID == 0 || got[u2.ID].FaceID == 0 {
			t.Fatalf("expected only album A faces, got %+v", got)
		}
		if got[u1.ID].ImageWidth != 64 || got[u1.ID].ImageHeight != 48 {
			t.Fatalf("expected image dims, got %+v", got[u1.ID])
		}
		if resp := doRequest(t, http.MethodGet, "/api/faces/untagged?album_id=abc", env.adminToken, nil, ""); resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad album_id, got %d", resp.StatusCode)
		}
	})

	t.Run("group by image", func(t *testing.T) {
		items := listUntagged(t, env.adminToken, fmt.Sprintf("album_id=%d&group_by_image=true", albumA.ID))
		if len(items) != 1 {
			t.Fatalf("expected one face per image, got %+v", items)
		}
	})

	t.Run("limit", func(t *testing.T) {
		if items := listUntagged(t, env.adminToken, fmt.Sprintf("album_id=%d&limit=1", albumA.ID)); len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
	})

	t.Run("only confirmed faces vote", func(t *testing.T) {
		got := byID(listUntagged(t, env.adminToken, fmt.Sprintf("album_id=%d&limit=100", albumA.ID)))
		s := got[u1.ID]
		if s.SuggestedPersonID == nil || *s.SuggestedPersonID != confirmedPerson.ID || s.SuggestionCount != 1 {
			t.Fatalf("expected suggestion of confirmed person with 1 vote, got %+v", s)
		}
		if s.SimilarFacesCount < 4 {
			t.Fatalf("expected neighbours to be counted, got %d", s.SimilarFacesCount)
		}
	})

	t.Run("suggest endpoint", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/faces/%d/suggest", u1.ID), env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("suggest failed: %d %s", resp.StatusCode, resp.Body)
		}
		var s struct {
			PersonID *uint `json:"suggested_person_id"`
		}
		resp.decodeData(t, &s)
		if s.PersonID == nil || *s.PersonID != confirmedPerson.ID {
			t.Fatalf("unexpected suggestion: %s", resp.Body)
		}
	})

	t.Run("similar excludes self", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/faces/%d/similar?limit=100", u1.ID), env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("similar failed: %d %s", resp.StatusCode, resp.Body)
		}
		var sim []struct {
			FaceID uint `json:"face_id"`
		}
		resp.decodeData(t, &sim)
		if len(sim) < 5 {
			t.Fatalf("expected >=5 similar faces, got %d", len(sim))
		}
		for _, s := range sim {
			if s.FaceID == u1.ID {
				t.Fatal("target face returned as its own neighbour")
			}
		}
	})

	t.Run("auto-tag refuses tagged face", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, fmt.Sprintf("/api/faces/%d/auto-tag", fc.ID), env.adminToken, nil, "")
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409, got %d %s", resp.StatusCode, resp.Body)
		}
	})
}

func TestUpdateFaceValidationAndMove(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "UV "+suffix, "uv-"+suffix, "")
	img := uploadImage(t, env.adminToken, album.ID, "a.jpg")
	setImageDims(t, img, 64, 48)
	face := addFace(t, env.adminToken, img, nil)
	addEmbedding(t, face.ID, unitVec(200, 0))

	put := func(payload map[string]any) apiResponse {
		return doJSON(t, http.MethodPut, fmt.Sprintf("/api/faces/%d", face.ID), env.adminToken, payload)
	}

	bad := []map[string]any{
		{"x1": -1},
		{"x2": 3},   // x2 <= x1 (x1 is 5)
		{"y2": 500}, // outside the 64x48 image
		{"x1": "left"},
		{"x1": 1.5},
		{"x1": 40, "x2": 30},
	}
	for _, p := range bad {
		if resp := put(p); resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("payload %v: expected 400, got %d %s", p, resp.StatusCode, resp.Body)
		}
	}

	var count int64
	env.app.DB.Model(&models.FaceEmbedding{}).Where("face_id = ?", face.ID).Count(&count)
	if count != 1 {
		t.Fatalf("rejected updates must keep the embedding, got %d", count)
	}

	if resp := put(map[string]any{"x2": 40}); resp.StatusCode != http.StatusOK {
		t.Fatalf("valid move failed: %d %s", resp.StatusCode, resp.Body)
	}
	env.app.DB.Unscoped().Model(&models.FaceEmbedding{}).Where("face_id = ?", face.ID).Count(&count)
	if count != 0 {
		t.Fatalf("moving the box should hard-delete the stale embedding, got %d", count)
	}

	// deleting a face removes its embedding row too
	face2 := addFace(t, env.adminToken, img, nil)
	addEmbedding(t, face2.ID, unitVec(210, 0))
	if resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/faces/%d", face2.ID), env.adminToken, nil, ""); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete failed: %d", resp.StatusCode)
	}
	env.app.DB.Unscoped().Model(&models.FaceEmbedding{}).Where("face_id = ?", face2.ID).Count(&count)
	if count != 0 {
		t.Fatalf("delete should hard-delete the embedding, got %d", count)
	}

	// AddFace validates against image bounds
	resp := doJSON(t, http.MethodPost, "/api/images/faces", env.adminToken, map[string]any{"image_path": img, "x1": 5, "y1": 5, "x2": 500, "y2": 30})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-image box, got %d %s", resp.StatusCode, resp.Body)
	}
}

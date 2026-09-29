package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

// TestCollectionPrivacyRegression is a regression test: a collection created with
// is_public=false must stay private (excluded from the public collection listing,
// and stored/returned as is_public=false, not silently flipped to the true default).
func TestCollectionPrivacyRegression(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	slug := "private-coll-" + suffix

	createResp := doJSON(t, http.MethodPost, "/api/admin/collections", env.adminToken, map[string]any{
		"name":      "Private Collection " + suffix,
		"slug":      slug,
		"is_public": false,
	})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create private collection: %d %s", createResp.StatusCode, createResp.Body)
	}
	var created struct {
		ID       uint `json:"id"`
		IsPublic bool `json:"is_public"`
	}
	createResp.decodeData(t, &created)
	if created.IsPublic {
		t.Fatalf("expected created collection to have is_public=false, got true")
	}

	t.Run("excluded from public listing", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/collections", "", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var collections []struct {
			Slug string `json:"slug"`
		}
		resp.decodeData(t, &collections)
		for _, c := range collections {
			if c.Slug == slug {
				t.Fatalf("private collection %q should not appear in the public listing", slug)
			}
		}
	})

	t.Run("admin view still reports is_public=false", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/collections/%d", created.ID), env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var c struct {
			IsPublic bool `json:"is_public"`
		}
		resp.decodeData(t, &c)
		if c.IsPublic {
			t.Fatalf("expected is_public=false to persist, got true")
		}
	})
}

// TestAlbumGroupSlugRecreateAfterDelete is a regression test: an album group can be
// recreated with the same slug after the original was deleted (soft-delete must not
// hold the slug's unique index forever).
func TestAlbumGroupSlugRecreateAfterDelete(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	slug := "recreate-group-" + suffix

	first := doJSON(t, http.MethodPost, "/api/admin/groups", env.adminToken, map[string]any{
		"name": "Recreate Group " + suffix,
		"slug": slug,
	})
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create group: %d %s", first.StatusCode, first.Body)
	}
	var group struct {
		ID uint `json:"id"`
	}
	first.decodeData(t, &group)

	del := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/groups/%d", group.ID), env.adminToken, nil, "")
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("failed to delete group: %d %s", del.StatusCode, del.Body)
	}

	second := doJSON(t, http.MethodPost, "/api/admin/groups", env.adminToken, map[string]any{
		"name": "Recreate Group " + suffix + " v2",
		"slug": slug,
	})
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("expected recreating a group with a deleted group's slug to succeed, got %d: %s", second.StatusCode, second.Body)
	}
}

// TestCollectionSlugRecreateAfterDelete mirrors TestAlbumGroupSlugRecreateAfterDelete
// for collections.
func TestCollectionSlugRecreateAfterDelete(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	slug := "recreate-coll-" + suffix

	first := doJSON(t, http.MethodPost, "/api/admin/collections", env.adminToken, map[string]any{
		"name": "Recreate Collection " + suffix,
		"slug": slug,
	})
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create collection: %d %s", first.StatusCode, first.Body)
	}
	var collection struct {
		ID uint `json:"id"`
	}
	first.decodeData(t, &collection)

	del := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/collections/%d", collection.ID), env.adminToken, nil, "")
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("failed to delete collection: %d %s", del.StatusCode, del.Body)
	}

	second := doJSON(t, http.MethodPost, "/api/admin/collections", env.adminToken, map[string]any{
		"name": "Recreate Collection " + suffix + " v2",
		"slug": slug,
	})
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("expected recreating a collection with a deleted collection's slug to succeed, got %d: %s", second.StatusCode, second.Body)
	}
}

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

// createUser creates a user with no global permissions and returns its ID.
func createUser(t *testing.T, adminToken, username, password string) uint {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/admin/users", adminToken, map[string]any{
		"username":   username,
		"password":   password,
		"first_name": "Test",
		"last_name":  "User",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create user %q: %d %s", username, resp.StatusCode, resp.Body)
	}
	var user struct {
		ID uint `json:"id"`
	}
	resp.decodeData(t, &user)
	return user.ID
}

// grantAlbumPermission grants the given user album-scoped permissions on albumID.
func grantAlbumPermission(t *testing.T, adminToken string, albumID, userID uint, perms []string) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/users", albumID), adminToken, map[string]any{
		"user_id":     userID,
		"permissions": perms,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to grant album permission: %d %s", resp.StatusCode, resp.Body)
	}
}

// TestPerAlbumPermissions is a regression test for per-album access control: a user
// granted a permission scoped to album A can access album A's admin routes but not
// album B's, even though they have no global album permissions.
func TestPerAlbumPermissions(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	albumA := createAlbum(t, env.adminToken, "Perm Album A "+suffix, "perm-a-"+suffix, "")
	albumB := createAlbum(t, env.adminToken, "Perm Album B "+suffix, "perm-b-"+suffix, "")

	username := "scoped_" + suffix
	password := "test-password-" + suffix
	userID := createUser(t, env.adminToken, username, password)
	grantAlbumPermission(t, env.adminToken, albumA.ID, userID, []string{"album.view.content"})

	token, err := login(env.server.URL, username, password)
	if err != nil {
		t.Fatalf("failed to log in as scoped user: %v", err)
	}

	t.Run("can access album A", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d", albumA.ID), token, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for album A, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("cannot access album B", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d", albumB.ID), token, nil, "")
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for album B, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})

	t.Run("cannot access album B images", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/images", albumB.ID), token, nil, "")
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for album B images, got %d: %s", resp.StatusCode, resp.Body)
		}
	})
}

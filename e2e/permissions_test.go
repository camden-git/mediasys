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

// directAlbumPermissions returns the user's direct permissions on albumID, as seen by the admin.
func directAlbumPermissions(t *testing.T, userID, albumID uint) []string {
	t.Helper()
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", userID), requireShared(t).adminToken, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get user %d: %d %s", userID, resp.StatusCode, resp.Body)
	}
	var user struct {
		AlbumPermissions []struct {
			AlbumID     uint     `json:"album_id"`
			Permissions []string `json:"permissions"`
		} `json:"album_permissions"`
	}
	resp.decodeData(t, &user)
	for _, p := range user.AlbumPermissions {
		if p.AlbumID == albumID {
			return p.Permissions
		}
	}
	return nil
}

// TestAlbumMemberManagersCannotGrantWhatTheyLack verifies album member managers (per-album
// or global) can only hand out album permissions they hold on that album themselves.
func TestAlbumMemberManagersCannotGrantWhatTheyLack(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	album := createAlbum(t, env.adminToken, "Grant "+suffix, "grant-"+suffix, "")
	secCleanupDelete(t, fmt.Sprintf("/api/admin/albums/%d", album.ID))
	usersPath := fmt.Sprintf("/api/admin/albums/%d/users", album.ID)

	manager := secMakeCleanUser(t, "am_manager_"+suffix, nil, nil)
	managerPerms := []string{"album.manage.members", "album.photo.upload"}
	grantAlbumPermission(t, env.adminToken, album.ID, manager.ID, managerPerms)
	managerToken := secLoginAs(t, manager.Username)

	globalManager := secMakeCleanUser(t, "am_global_"+suffix, []string{"album.manage.members.global"}, nil)
	globalToken := secLoginAs(t, globalManager.Username)

	member := secMakeCleanUser(t, "am_member_"+suffix, nil, nil)
	memberPath := fmt.Sprintf("%s/%d", usersPath, member.ID)

	t.Run("cannot grant self a permission not held", func(t *testing.T) {
		selfPath := fmt.Sprintf("%s/%d", usersPath, manager.ID)
		secExpectForbidden(t, doJSON(t, http.MethodPut, selfPath, managerToken, map[string]any{
			"permissions": append([]string{"album.photo.delete"}, managerPerms...),
		}))
		if got := directAlbumPermissions(t, manager.ID, album.ID); len(got) != len(managerPerms) {
			t.Fatalf("manager permissions changed despite 403: %v", got)
		}
	})

	t.Run("cannot add a member with a permission not held", func(t *testing.T) {
		secExpectForbidden(t, doJSON(t, http.MethodPost, usersPath, managerToken, map[string]any{
			"user_id": member.ID, "permissions": []string{"album.photo.delete"},
		}))
		secExpectForbidden(t, doJSON(t, http.MethodPost, usersPath, globalToken, map[string]any{
			"user_id": member.ID, "permissions": []string{"album.view.content"},
		}))
		if got := directAlbumPermissions(t, member.ID, album.ID); got != nil {
			t.Fatalf("member was added despite 403: %v", got)
		}
	})

	t.Run("can add a member with held permissions", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, usersPath, managerToken, map[string]any{
			"user_id": member.ID, "permissions": []string{"album.photo.upload"},
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("cannot extend a member with a permission not held", func(t *testing.T) {
		secExpectForbidden(t, doJSON(t, http.MethodPut, memberPath, managerToken, map[string]any{
			"permissions": []string{"album.photo.upload", "album.photo.editmeta"},
		}))
		secExpectForbidden(t, doJSON(t, http.MethodPut, memberPath, globalToken, map[string]any{
			"permissions": []string{"album.photo.upload", "album.view.content"},
		}))
		if got := directAlbumPermissions(t, member.ID, album.ID); len(got) != 1 || got[0] != "album.photo.upload" {
			t.Fatalf("member permissions changed despite 403: %v", got)
		}
	})

	t.Run("existing permissions the caller lacks can be kept", func(t *testing.T) {
		resp := doJSON(t, http.MethodPut, memberPath, env.adminToken, map[string]any{
			"permissions": []string{"album.photo.upload", "album.photo.delete"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("admin update failed: %d %s", resp.StatusCode, resp.Body)
		}
		resp = doJSON(t, http.MethodPut, memberPath, managerToken, map[string]any{
			"permissions": []string{"album.photo.delete"},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 keeping an existing permission, got %d: %s", resp.StatusCode, resp.Body)
		}
	})
}

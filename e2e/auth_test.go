package e2e_test

import (
	"net/http"
	"testing"
)

// TestInitialSetupLoginMe covers the basic auth flow: the initial admin created by
// TestMain can fetch /api/auth/me, a second initial-admin attempt is rejected now
// that a user exists, and bad credentials are rejected.
func TestInitialSetupLoginMe(t *testing.T) {
	env := requireShared(t)

	t.Run("me returns the authenticated admin", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/auth/me", env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
		}
		var user struct {
			Username string `json:"username"`
		}
		resp.decode(t, &user)
		if user.Username != env.adminUsername {
			t.Fatalf("expected username %q, got %q", env.adminUsername, user.Username)
		}
	})

	t.Run("me without a token is rejected", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/auth/me", "", nil, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})

	t.Run("setup is rejected once a user exists", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, "/api/setup/initial-admin", "", map[string]string{
			"username": "someone-else",
			"password": "irrelevant",
		})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})

	t.Run("login with wrong password is rejected", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": env.adminUsername,
			"password": "definitely-wrong",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})
}

// TestAnonymousRejectedOnMutatingAndDebugRoutes is a regression test for a security bug
// where mutating people/face routes and /api/debug/* did not require authentication.
func TestAnonymousRejectedOnMutatingAndDebugRoutes(t *testing.T) {
	requireShared(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"create album", http.MethodPost, "/api/admin/albums"},
		{"list admin users", http.MethodGet, "/api/admin/users"},
		{"create person", http.MethodPost, "/api/people"},
		{"add face", http.MethodPost, "/api/images/faces"},
		{"update face", http.MethodPut, "/api/faces/1"},
		{"delete face", http.MethodDelete, "/api/faces/1"},
		{"tag face", http.MethodPost, "/api/faces/1/tag"},
		{"debug faces", http.MethodGet, "/api/debug/faces"},
		{"debug queue detection", http.MethodPost, "/api/debug/queue_detection"},
		{"debug detection status", http.MethodGet, "/api/debug/detection_status"},
		{"debug image with faces", http.MethodGet, "/api/debug/image_with_faces"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doRequest(t, tc.method, tc.path, "", nil, "")
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s: expected 401 for anonymous request, got %d: %s", tc.method, tc.path, resp.StatusCode, resp.Body)
			}
			assertErrorShape(t, resp)
		})
	}
}

// TestErrorResponseShape spot-checks that a variety of error conditions produce the
// standardized {"errors":[{code,status,detail}]} body.
func TestErrorResponseShape(t *testing.T) {
	env := requireShared(t)

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/albums/does-not-exist-xyz", "", nil, "")
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})

	t.Run("forbidden", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, "/api/admin/albums", env.adminToken, map[string]any{})
		// missing name/slug -> validation error, still standard shape
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})

	t.Run("malformed json", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/api/auth/login", "", nil, "application/json")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})
}

package e2e_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/camden-git/mediasysbackend/cli"
	"github.com/camden-git/mediasysbackend/repository"
)

// TestCLICreateUser covers `mediasys user create`: users it creates can log in,
// --admin grants Super Administrator, and invalid input is rejected.
func TestCLICreateUser(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	// other tests rely on the shared admin being the only Super Administrator
	createUser := func(t *testing.T, in cli.NewUser) {
		t.Helper()
		user, err := cli.CreateUser(env.app.DB, in)
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		t.Cleanup(func() {
			if err := repository.NewGormUserRepository(env.app.DB).Delete(user.ID); err != nil {
				t.Errorf("failed to delete user %q: %v", user.Username, err)
			}
		})
	}

	t.Run("admin can log in and use admin routes", func(t *testing.T) {
		name, password := "cli_admin_"+suffix, "cli-admin-password"
		createUser(t, cli.NewUser{Username: name, Password: password, Admin: true})
		token, err := login(env.server.URL, name, password)
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if resp := doRequest(t, http.MethodGet, "/api/admin/users", token, nil, ""); resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 listing users as admin, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("regular user can log in but not use admin routes", func(t *testing.T) {
		name, password := "cli_user_"+suffix, "cli-user-password"
		createUser(t, cli.NewUser{Username: " " + name + " ", FirstName: "Cli", Password: password})
		token, err := login(env.server.URL, name, password)
		if err != nil {
			t.Fatalf("login with trimmed username: %v", err)
		}
		if resp := doRequest(t, http.MethodGet, "/api/admin/users", token, nil, ""); resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 listing users as a regular user, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("invalid input is rejected", func(t *testing.T) {
		cases := []struct {
			name    string
			in      cli.NewUser
			wantErr string
		}{
			{"duplicate username", cli.NewUser{Username: env.adminUsername, Password: "long-enough-password"}, "already taken"},
			{"short password", cli.NewUser{Username: "cli_short_" + suffix, Password: "short"}, "at least"},
			{"blank username", cli.NewUser{Username: "   ", Password: "long-enough-password"}, "username is required"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := cli.CreateUser(env.app.DB, tc.in)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
			})
		}
	})
}

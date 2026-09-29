package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

const superAdminRoleName = "root"

type secRole struct {
	ID                     uint     `json:"id"`
	Name                   string   `json:"name"`
	GlobalPermissions      []string `json:"global_permissions"`
	GlobalAlbumPermissions []string `json:"global_album_permissions"`
	AlbumPermissions       []struct {
		AlbumID     uint     `json:"album_id"`
		Permissions []string `json:"permissions"`
	} `json:"album_permissions"`
}

type secUser struct {
	ID                uint     `json:"id"`
	Username          string   `json:"username"`
	FirstName         string   `json:"first_name"`
	GlobalPermissions []string `json:"global_permissions"`
	Roles             []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"roles"`
}

func secPostUser(t *testing.T, token, username string, globalPerms []string, roleIDs []uint) apiResponse {
	t.Helper()
	return doJSON(t, http.MethodPost, "/api/admin/users", token, map[string]any{
		"username":           username,
		"password":           "test-password-" + username,
		"first_name":         "Sec",
		"last_name":          "User",
		"global_permissions": globalPerms,
		"role_ids":           roleIDs,
	})
}

func secMakeUser(t *testing.T, token, username string, globalPerms []string, roleIDs []uint) secUser {
	t.Helper()
	resp := secPostUser(t, token, username, globalPerms, roleIDs)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create user %q: %d %s", username, resp.StatusCode, resp.Body)
	}
	var u secUser
	resp.decodeData(t, &u)
	return u
}

func secLoginAs(t *testing.T, username string) string {
	t.Helper()
	token, err := login(requireShared(t).server.URL, username, "test-password-"+username)
	if err != nil {
		t.Fatalf("failed to log in as %q: %v", username, err)
	}
	return token
}

func secMakeRole(t *testing.T, token, name string, global, albumGlobal []string) secRole {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/admin/roles", token, map[string]any{
		"name":                     name,
		"global_permissions":       global,
		"global_album_permissions": albumGlobal,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create role %q: %d %s", name, resp.StatusCode, resp.Body)
	}
	var role secRole
	resp.decodeData(t, &role)
	return role
}

func secGetUser(t *testing.T, token string, id uint) secUser {
	t.Helper()
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", id), token, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get user %d: %d %s", id, resp.StatusCode, resp.Body)
	}
	var u secUser
	resp.decodeData(t, &u)
	return u
}

func secGetRole(t *testing.T, token string, id uint) secRole {
	t.Helper()
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/roles/%d", id), token, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to get role %d: %d %s", id, resp.StatusCode, resp.Body)
	}
	var role secRole
	resp.decodeData(t, &role)
	return role
}

// secFindRoleID looks a role up by name via the list endpoint (as the initial admin).
func secFindRoleID(t *testing.T, token, name string) (uint, bool) {
	t.Helper()
	resp := doRequest(t, http.MethodGet, "/api/admin/roles?per_page=100", token, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to list roles: %d %s", resp.StatusCode, resp.Body)
	}
	var roles []secRole
	resp.decodeData(t, &roles)
	for _, r := range roles {
		if r.Name == name {
			return r.ID, true
		}
	}
	return 0, false
}

func secAdminID(t *testing.T) uint {
	t.Helper()
	env := requireShared(t)
	resp := doRequest(t, http.MethodGet, "/api/admin/users?per_page=100", env.adminToken, nil, "")
	var users []secUser
	resp.decodeData(t, &users)
	for _, u := range users {
		if u.Username == env.adminUsername {
			return u.ID
		}
	}
	t.Fatalf("initial admin not found in user list")
	return 0
}

func secExpectForbidden(t *testing.T, resp apiResponse) {
	t.Helper()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, resp.Body)
	}
	assertErrorShape(t, resp)
}

// TestUserDelegationLimits verifies a non-super-admin cannot use user/role endpoints to
// grant anything they do not hold themselves.
func TestUserDelegationLimits(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	rootID, ok := secFindRoleID(t, env.adminToken, superAdminRoleName)
	if !ok {
		t.Fatalf("super administrator role not found")
	}
	adminID := secAdminID(t)

	staffName := "staff_" + suffix
	staff := secMakeUser(t, env.adminToken, staffName, []string{
		"user.create", "user.edit", "user.delete", "user.list", "user.view",
		"role.create", "role.edit", "role.edit.users", "role.list", "role.view",
	}, nil)
	staffToken := secLoginAs(t, staffName)

	powerRole := secMakeRole(t, env.adminToken, "power_"+suffix, []string{"role.delete"}, nil)
	albumRole := secMakeRole(t, env.adminToken, "albumall_"+suffix, nil, []string{"album.photo.delete"})
	modestRole := secMakeRole(t, env.adminToken, "modest_"+suffix, []string{"user.list"}, nil)
	victim := secMakeUser(t, env.adminToken, "victim_"+suffix, nil, nil)

	t.Run("create user with permission caller lacks", func(t *testing.T) {
		secExpectForbidden(t, secPostUser(t, staffToken, "esc1_"+suffix, []string{"role.delete"}, nil))
	})

	t.Run("create user with super admin role", func(t *testing.T) {
		secExpectForbidden(t, secPostUser(t, staffToken, "esc2_"+suffix, nil, []uint{rootID}))
	})

	t.Run("create user with role carrying permissions caller lacks", func(t *testing.T) {
		secExpectForbidden(t, secPostUser(t, staffToken, "esc3_"+suffix, nil, []uint{powerRole.ID}))
		secExpectForbidden(t, secPostUser(t, staffToken, "esc4_"+suffix, nil, []uint{albumRole.ID}))
	})

	t.Run("create user within own permissions", func(t *testing.T) {
		u := secMakeUser(t, staffToken, "ok1_"+suffix, []string{"user.list"}, []uint{modestRole.ID})
		if len(u.Roles) != 1 || u.Roles[0].ID != modestRole.ID {
			t.Fatalf("expected the modest role to be assigned, got %+v", u.Roles)
		}
	})

	t.Run("update user escalation", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%d", victim.ID)
		secExpectForbidden(t, doJSON(t, http.MethodPut, path, staffToken, map[string]any{"global_permissions": []string{"role.delete"}}))
		secExpectForbidden(t, doJSON(t, http.MethodPut, path, staffToken, map[string]any{"role_ids": []uint{rootID}}))
		secExpectForbidden(t, doJSON(t, http.MethodPut, path, staffToken, map[string]any{"role_ids": []uint{powerRole.ID}}))

		got := secGetUser(t, env.adminToken, victim.ID)
		if len(got.Roles) != 0 || len(got.GlobalPermissions) != 0 {
			t.Fatalf("victim was modified despite 403s: %+v", got)
		}

		resp := doJSON(t, http.MethodPut, path, staffToken, map[string]any{"role_ids": []uint{modestRole.ID}})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 assigning a permitted role, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("cannot remove a role the caller could not grant", func(t *testing.T) {
		holder := secMakeUser(t, env.adminToken, "holder_"+suffix, nil, []uint{powerRole.ID})
		secExpectForbidden(t, doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/users/%d", holder.ID), staffToken, map[string]any{"role_ids": []uint{}}))
	})

	t.Run("cannot edit or delete a super admin", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%d", adminID)
		secExpectForbidden(t, doJSON(t, http.MethodPut, path, staffToken, map[string]any{"password": "hijacked-password"}))
		secExpectForbidden(t, doRequest(t, http.MethodDelete, path, staffToken, nil, ""))
		if _, err := login(env.server.URL, env.adminUsername, env.adminPassword); err != nil {
			t.Fatalf("admin password was changed or admin deleted: %v", err)
		}
	})

	t.Run("role create and update cannot grant permissions caller lacks", func(t *testing.T) {
		secExpectForbidden(t, doJSON(t, http.MethodPost, "/api/admin/roles", staffToken, map[string]any{
			"name": "esc_role_" + suffix, "global_permissions": []string{"role.delete"},
		}))
		secExpectForbidden(t, doJSON(t, http.MethodPost, "/api/admin/roles", staffToken, map[string]any{
			"name": "esc_role_album_" + suffix, "global_album_permissions": []string{"album.photo.delete"},
		}))

		mine := secMakeRole(t, staffToken, "mine_"+suffix, []string{"user.list"}, nil)
		secExpectForbidden(t, doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/roles/%d", mine.ID), staffToken, map[string]any{
			"global_permissions": []string{"user.list", "role.delete"},
		}))
		if got := secGetRole(t, env.adminToken, mine.ID); len(got.GlobalPermissions) != 1 {
			t.Fatalf("role was modified despite 403: %+v", got)
		}
	})

	t.Run("role membership endpoints respect delegation", func(t *testing.T) {
		secExpectForbidden(t, doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/roles/%d/users", powerRole.ID), staffToken, map[string]any{"user_id": staff.ID}))
		secExpectForbidden(t, doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/roles/%d/users", powerRole.ID), staffToken, map[string]any{"user_id": victim.ID}))

		holder := secMakeUser(t, env.adminToken, "holder2_"+suffix, nil, []uint{powerRole.ID})
		secExpectForbidden(t, doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/roles/%d/users/%d", powerRole.ID, holder.ID), staffToken, nil, ""))

		resp := doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/roles/%d/users", modestRole.ID), staffToken, map[string]any{"user_id": victim.ID})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 adding user to a permitted role, got %d: %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("add nonexistent user to role returns 404", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/roles/%d/users", modestRole.ID), env.adminToken, map[string]any{"user_id": 999999})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
	})
}

// TestCannotDeleteSelfOrLastSuperAdmin covers the guards that keep at least one Super
// Administrator around.
func TestCannotDeleteSelfOrLastSuperAdmin(t *testing.T) {
	env := requireShared(t)
	adminID := secAdminID(t)
	rootID, _ := secFindRoleID(t, env.adminToken, superAdminRoleName)

	t.Run("cannot delete self", func(t *testing.T) {
		secExpectForbidden(t, doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", adminID), env.adminToken, nil, ""))
		secGetUser(t, env.adminToken, adminID)
	})

	t.Run("cannot remove super admin role from the last super admin", func(t *testing.T) {
		secExpectForbidden(t, doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/users/%d", adminID), env.adminToken, map[string]any{"role_ids": []uint{}}))
		got := secGetUser(t, env.adminToken, adminID)
		if len(got.Roles) != 1 || got.Roles[0].ID != rootID {
			t.Fatalf("super admin role was removed: %+v", got.Roles)
		}
	})
}

// TestSuperAdminManagesSuperAdmins verifies super admins are exempt from the delegation
// limits: they can create and delete other super admins while more than one exists.
func TestSuperAdminManagesSuperAdmins(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	rootID, _ := secFindRoleID(t, env.adminToken, superAdminRoleName)

	second := secMakeUser(t, env.adminToken, "root2_"+suffix, nil, []uint{rootID})
	secondToken := secLoginAs(t, second.Username)
	third := secMakeUser(t, secondToken, "root3_"+suffix, []string{"role.delete"}, []uint{rootID})

	resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", third.ID), secondToken, nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting a non-last super admin, got %d: %s", resp.StatusCode, resp.Body)
	}
	resp = doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/users/%d", second.ID), env.adminToken, map[string]any{"role_ids": []uint{}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 demoting a non-last super admin, got %d: %s", resp.StatusCode, resp.Body)
	}
}

// TestUserRoleReplacement verifies roles dropped from a user are really removed, and that
// updating a user or profile does not overwrite concurrent role edits.
func TestUserRoleReplacement(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	roleA := secMakeRole(t, env.adminToken, "repl_a_"+suffix, []string{"user.list"}, nil)
	roleB := secMakeRole(t, env.adminToken, "repl_b_"+suffix, []string{"user.view"}, nil)
	user := secMakeUser(t, env.adminToken, "repl_"+suffix, nil, []uint{roleA.ID, roleB.ID})
	if len(user.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %+v", user.Roles)
	}
	path := fmt.Sprintf("/api/admin/users/%d", user.ID)

	resp := doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"role_ids": []uint{roleB.ID}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update failed: %d %s", resp.StatusCode, resp.Body)
	}
	if got := secGetUser(t, env.adminToken, user.ID); len(got.Roles) != 1 || got.Roles[0].ID != roleB.ID {
		t.Fatalf("expected only role B after update, got %+v", got.Roles)
	}

	resp = doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"role_ids": []uint{}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update failed: %d %s", resp.StatusCode, resp.Body)
	}
	if got := secGetUser(t, env.adminToken, user.ID); len(got.Roles) != 0 {
		t.Fatalf("expected no roles after clearing, got %+v", got.Roles)
	}

	t.Run("profile update does not revert role edits", func(t *testing.T) {
		doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"role_ids": []uint{roleA.ID}})
		userToken := secLoginAs(t, user.Username)
		// the user's token resolves the role on each request, so edit the role after a first read
		resp := doRequest(t, http.MethodGet, "/api/auth/me", userToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("me failed: %d %s", resp.StatusCode, resp.Body)
		}
		resp = doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/roles/%d", roleA.ID), env.adminToken, map[string]any{"name": "repl_a_renamed_" + suffix})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("role rename failed: %d %s", resp.StatusCode, resp.Body)
		}
		resp = doJSON(t, http.MethodPut, "/api/auth/me", userToken, map[string]any{"first_name": "Renamed"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("profile update failed: %d %s", resp.StatusCode, resp.Body)
		}
		if got := secGetRole(t, env.adminToken, roleA.ID); got.Name != "repl_a_renamed_"+suffix {
			t.Fatalf("role rename was reverted: %+v", got)
		}
		if got := secGetUser(t, env.adminToken, user.ID); got.FirstName != "Renamed" {
			t.Fatalf("profile update lost: %+v", got)
		}
	})
}

// TestRoleWritesAreAtomic verifies invalid role payloads leave no partial state behind.
func TestRoleWritesAreAtomic(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "Atomic Album "+suffix, "atomic-"+suffix, "")

	t.Run("create with invalid album permission leaves no role", func(t *testing.T) {
		name := "orphan_" + suffix
		resp := doJSON(t, http.MethodPost, "/api/admin/roles", env.adminToken, map[string]any{
			"name": name,
			"album_permissions": []map[string]any{
				{"album_id": album.ID, "permissions": []string{"not.a.permission"}},
			},
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.StatusCode, resp.Body)
		}
		assertErrorShape(t, resp)
		if _, found := secFindRoleID(t, env.adminToken, name); found {
			t.Fatalf("orphan role %q was left behind", name)
		}
	})

	t.Run("update with invalid album permission keeps existing ones", func(t *testing.T) {
		resp := doJSON(t, http.MethodPost, "/api/admin/roles", env.adminToken, map[string]any{
			"name": "atomic_" + suffix,
			"album_permissions": []map[string]any{
				{"album_id": album.ID, "permissions": []string{"album.view.content"}},
			},
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create failed: %d %s", resp.StatusCode, resp.Body)
		}
		var role secRole
		resp.decodeData(t, &role)
		if len(role.AlbumPermissions) != 1 {
			t.Fatalf("expected one album permission, got %+v", role.AlbumPermissions)
		}

		path := fmt.Sprintf("/api/admin/roles/%d", role.ID)
		resp = doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{
			"album_permissions": []map[string]any{
				{"album_id": album.ID, "permissions": []string{"not.a.permission"}},
			},
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.StatusCode, resp.Body)
		}
		got := secGetRole(t, env.adminToken, role.ID)
		if len(got.AlbumPermissions) != 1 || got.AlbumPermissions[0].Permissions[0] != "album.view.content" {
			t.Fatalf("album permissions were destroyed by a failed update: %+v", got.AlbumPermissions)
		}

		resp = doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"album_permissions": []map[string]any{}})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("clearing album permissions failed: %d %s", resp.StatusCode, resp.Body)
		}
		if got := secGetRole(t, env.adminToken, role.ID); len(got.AlbumPermissions) != 0 {
			t.Fatalf("expected album permissions cleared, got %+v", got.AlbumPermissions)
		}
	})
}

// TestPerPageBelowDefault verifies small per_page values are honored.
func TestPerPageBelowDefault(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	for i := 0; i < 3; i++ {
		secMakeUser(t, env.adminToken, fmt.Sprintf("pp%d_%s", i, suffix), nil, nil)
	}

	resp := doRequest(t, http.MethodGet, "/api/admin/users?per_page=2", env.adminToken, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list failed: %d %s", resp.StatusCode, resp.Body)
	}
	var body struct {
		Data []secUser `json:"data"`
		Meta struct {
			Pagination struct {
				PerPage int `json:"per_page"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	resp.decode(t, &body)
	if body.Meta.Pagination.PerPage != 2 || len(body.Data) != 2 {
		t.Fatalf("expected per_page=2 with 2 items, got per_page=%d items=%d", body.Meta.Pagination.PerPage, len(body.Data))
	}
}

package handlers

import (
	"fmt"
	"net/http"

	"github.com/camden-git/mediasysbackend/models"
)

// isSuperAdmin reports whether the user holds the Super Administrator role.
func isSuperAdmin(user *models.User) bool {
	if user == nil {
		return false
	}
	for _, role := range user.Roles {
		if role != nil && role.Name == models.SuperAdminRoleName {
			return true
		}
	}
	return false
}

// requestUser returns the authenticated user from the request context, writing a 500 if it is missing.
func requestUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ContextUserError", "User not found in context")
		return nil, false
	}
	return user, true
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// firstUngranted returns the first key the caller does not hold, ignoring keys listed in already.
func firstUngranted(keys, already []string, held func(string) bool) (string, bool) {
	for _, key := range keys {
		if containsString(already, key) {
			continue
		}
		if !held(key) {
			return key, true
		}
	}
	return "", false
}

func callerHasAlbumForAll(caller *models.User, perm string) bool {
	if caller.EffectivePermissions == nil {
		caller.RefreshEffectivePermissions()
	}
	return containsString(caller.EffectivePermissions.Album.ForAll, perm)
}

// globalGrantDenial returns a non-empty message if a non-super-admin caller tries to grant
// global permissions (not already in already) that they do not hold themselves.
func globalGrantDenial(caller *models.User, keys, already []string) string {
	if isSuperAdmin(caller) {
		return ""
	}
	if key, missing := firstUngranted(keys, already, caller.HasGlobalPermission); missing {
		return fmt.Sprintf("You cannot grant the global permission '%s' because you do not hold it", key)
	}
	return ""
}

// albumForAllGrantDenial is like globalGrantDenial for album permissions that apply to all albums.
func albumForAllGrantDenial(caller *models.User, keys, already []string) string {
	if isSuperAdmin(caller) {
		return ""
	}
	held := func(perm string) bool { return callerHasAlbumForAll(caller, perm) }
	if key, missing := firstUngranted(keys, already, held); missing {
		return fmt.Sprintf("You cannot grant the album permission '%s' for all albums because you do not hold it", key)
	}
	return ""
}

// albumGrantDenial is like globalGrantDenial for permissions on a single album.
func albumGrantDenial(caller *models.User, albumID uint, keys, already []string) string {
	if isSuperAdmin(caller) {
		return ""
	}
	held := func(perm string) bool { return caller.HasAlbumPermission(albumID, perm) }
	if key, missing := firstUngranted(keys, already, held); missing {
		return fmt.Sprintf("You cannot grant the album permission '%s' on album %d because you do not hold it", key, albumID)
	}
	return ""
}

// roleGrantDenial returns a non-empty message if a non-super-admin caller may not assign or
// remove the role, i.e. it is the Super Administrator role or carries permissions the caller lacks.
func roleGrantDenial(caller *models.User, role *models.Role) string {
	if isSuperAdmin(caller) {
		return ""
	}
	if role.Name == models.SuperAdminRoleName {
		return "Only a Super Administrator can assign or remove the Super Administrator role"
	}
	if msg := globalGrantDenial(caller, role.GlobalPermissions, nil); msg != "" {
		return fmt.Sprintf("You cannot manage role '%s': %s", role.Name, msg)
	}
	if msg := albumForAllGrantDenial(caller, role.GlobalAlbumPermissions, nil); msg != "" {
		return fmt.Sprintf("You cannot manage role '%s': %s", role.Name, msg)
	}
	for _, rap := range role.AlbumPermissions {
		if msg := albumGrantDenial(caller, rap.AlbumID, rap.Permissions, nil); msg != "" {
			return fmt.Sprintf("You cannot manage role '%s': %s", role.Name, msg)
		}
	}
	return ""
}

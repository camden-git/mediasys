package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/permissions"
	"github.com/camden-git/mediasysbackend/repository"
	"gorm.io/gorm"
)

type SetupHandler struct {
	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository
}

func NewSetupHandler(userRepo repository.UserRepository, roleRepo repository.RoleRepository) *SetupHandler {
	return &SetupHandler{UserRepo: userRepo, RoleRepo: roleRepo}
}

type FirstAdminPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SyncSuperAdminRole ensures the Super Administrator role exists and has all defined permissions
// This function is idempotent and safe to run on every application startup
func SyncSuperAdminRole(roleRepo repository.RoleRepository) error {
	fmt.Println("Syncing Super Administrator role...")

	// get all defined permissions from the static definitions
	var allGlobalPerms []string
	var allGlobalAlbumPerms []string
	for _, group := range permissions.DefinedPermissionGroups {
		for _, perm := range group.Permissions {
			if perm.Scope == permissions.ScopeGlobal {
				allGlobalPerms = append(allGlobalPerms, perm.Key)
			} else if perm.Scope == permissions.ScopeAlbum {
				allGlobalAlbumPerms = append(allGlobalAlbumPerms, perm.Key)
			}
		}
	}
	sort.Strings(allGlobalPerms)
	sort.Strings(allGlobalAlbumPerms)

	role, err := roleRepo.GetByName(models.SuperAdminRoleName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Printf("'%s' role not found, creating...\n", models.SuperAdminRoleName)
			newRole := &models.Role{
				Name:                   models.SuperAdminRoleName,
				GlobalPermissions:      allGlobalPerms,
				GlobalAlbumPermissions: allGlobalAlbumPerms,
			}
			if err := roleRepo.Create(newRole); err != nil {
				return fmt.Errorf("failed to create '%s' role: %w", models.SuperAdminRoleName, err)
			}
			fmt.Printf("'%s' role created successfully with all permissions.\n", models.SuperAdminRoleName)
			return nil
		}

		return fmt.Errorf("failed to query for '%s' role: %w", models.SuperAdminRoleName, err)
	}

	sort.Strings(role.GlobalPermissions)
	sort.Strings(role.GlobalAlbumPermissions)

	needsUpdate := !reflect.DeepEqual(role.GlobalPermissions, allGlobalPerms) ||
		!reflect.DeepEqual(role.GlobalAlbumPermissions, allGlobalAlbumPerms)

	if needsUpdate {
		fmt.Printf("'%s' role is outdated, updating permissions...\n", models.SuperAdminRoleName)
		role.GlobalPermissions = allGlobalPerms
		role.GlobalAlbumPermissions = allGlobalAlbumPerms
		if err := roleRepo.Update(role); err != nil {
			return fmt.Errorf("failed to update '%s' role permissions: %w", models.SuperAdminRoleName, err)
		}
		fmt.Printf("'%s' role permissions updated successfully.\n", models.SuperAdminRoleName)
	} else {
		fmt.Printf("'%s' role is up to date.\n", models.SuperAdminRoleName)
	}

	return nil
}

// CreateFirstAdmin handles the creation of the initial administrator user
// This endpoint should only be usable if no other users exist in the system!!
func (h *SetupHandler) CreateFirstAdmin(w http.ResponseWriter, r *http.Request) {
	count, err := h.UserRepo.CountAll()
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserCountError", "Database error while checking for existing users.")
		return
	}
	if count > 0 {
		WriteAPIError(w, http.StatusForbidden, "SetupAlreadyCompleted", "Setup has already been completed: users exist.")
		return
	}

	var payload FirstAdminPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if strings.TrimSpace(payload.Username) == "" || payload.Password == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Username and password are required")
		return
	}
	if msg := PasswordPolicyError(payload.Password); msg != "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", msg)
		return
	}

	adminUser := &models.User{
		Username: payload.Username,
	}
	if err := adminUser.SetPassword(payload.Password); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "AdminCreateError", "Failed to create first admin user: "+err.Error())
		return
	}

	if err := h.UserRepo.CreateFirstAdmin(adminUser, models.SuperAdminRoleName); err != nil {
		if errors.Is(err, repository.ErrSetupAlreadyCompleted) {
			WriteAPIError(w, http.StatusForbidden, "SetupAlreadyCompleted", "Setup has already been completed.")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AdminCreateError", "Failed to create first admin user: "+err.Error())
		}
		return
	}

	fmt.Printf("Successfully created initial admin user '%s' with Super Administrator role.\n", adminUser.Username)

	WriteAPIResponse(w, http.StatusCreated, map[string]string{"message": "Initial admin user created successfully. Please log in."})
}

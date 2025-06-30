package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/permissions"
	"github.com/camden-git/mediasysbackend/repository"
	"gorm.io/gorm"
)

type SetupHandler struct {
	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository
	DB       *gorm.DB
}

func NewSetupHandler(db *gorm.DB, userRepo repository.UserRepository, roleRepo repository.RoleRepository) *SetupHandler {
	return &SetupHandler{UserRepo: userRepo, RoleRepo: roleRepo, DB: db}
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
	var count int64
	if err := h.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		http.Error(w, "Database error while checking for existing users.", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Setup has already been completed: users exist.", http.StatusForbidden)
		return
	}

	var payload FirstAdminPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		var innerCount int64
		if err := tx.Model(&models.User{}).Count(&innerCount).Error; err != nil {
			return fmt.Errorf("failed to count existing users in transaction: %w", err)
		}
		if innerCount > 0 {
			return errors.New("setup already completed")
		}

		var superAdminRole models.Role
		err := tx.Where("name = ?", models.SuperAdminRoleName).First(&superAdminRole).Error
		if err != nil {
			return fmt.Errorf("could not find the '%s' role, which should have been auto-generated: %w", models.SuperAdminRoleName, err)
		}

		adminUser := &models.User{
			Username: payload.Username,
		}
		if err := adminUser.SetPassword(payload.Password); err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		if err := tx.Create(adminUser).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		userRole := models.UserRole{UserID: adminUser.ID, RoleID: superAdminRole.ID}
		if err := tx.Create(&userRole).Error; err != nil {
			return fmt.Errorf("failed to assign super admin role to user: %w", err)
		}

		fmt.Printf("Successfully created initial admin user '%s' with Super Administrator role.\n", adminUser.Username)
		return nil
	})

	if txErr != nil {
		if txErr.Error() == "setup already completed" {
			http.Error(w, "Setup has already been completed.", http.StatusForbidden)
		} else {
			http.Error(w, "Failed to create first admin user: "+txErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Initial admin user created successfully. Please log in."})
}

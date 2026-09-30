package handlers

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/permissions"
	"github.com/camden-git/mediasysbackend/repository"
	"gorm.io/gorm"
)

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

package permissions

// PermissionScope defines the context in which a permission applies
type PermissionScope string

const (
	ScopeGlobal PermissionScope = "global" // applies system-wide
	ScopeAlbum  PermissionScope = "album"  // applies to a specific album context
)

// PermissionDefinition describes a single, specific permission
type PermissionDefinition struct {
	Key         string          `json:"key"`         // unique key, e.g., "album.create"
	Name        string          `json:"name"`        // friendly name, e.g., "Create Album"
	Description string          `json:"description"` // detailed description of what the permission allows
	Scope       PermissionScope `json:"scope"`       // scope of the permission (global or album-specific)
}

// PermissionGroupDefinition groups related permissions
type PermissionGroupDefinition struct {
	Key         string                 `json:"key"`         // unique key for the group, e.g., "album"
	Name        string                 `json:"name"`        // friendly name for the group, e.g., "Album Management"
	Description string                 `json:"description"` // detailed description of the permission group
	Permissions []PermissionDefinition `json:"permissions"` // list of permissions within this group
}

// DefinedPermissionGroups holds all statically defined permission groups and their permissions
var DefinedPermissionGroups = []PermissionGroupDefinition{
	{
		Key:         "user",
		Name:        "User Management",
		Description: "Permissions related to managing user accounts.",
		Permissions: []PermissionDefinition{
			{
				Key:         "user.create",
				Name:        "Create User",
				Description: "Allows creating new user accounts.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "user.edit",
				Name:        "Edit User",
				Description: "Allows editing existing user accounts (e.g., username, roles, direct permissions).",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "user.delete",
				Name:        "Delete User",
				Description: "Allows deleting user accounts.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "user.list",
				Name:        "List Users",
				Description: "Allows viewing a list of user accounts.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "user.view",
				Name:        "View User Details",
				Description: "Allows viewing detailed information of a specific user.",
				Scope:       ScopeGlobal,
			},
		},
	},
	{
		Key:         "role",
		Name:        "Role Management",
		Description: "Permissions related to managing roles and their assigned permissions.",
		Permissions: []PermissionDefinition{
			{
				Key:         "role.create",
				Name:        "Create Role",
				Description: "Allows creating new roles.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.edit",
				Name:        "Edit Role",
				Description: "Allows editing existing roles (e.g., name, assigned permissions).",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.delete",
				Name:        "Delete Role",
				Description: "Allows deleting roles.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.list",
				Name:        "List Roles",
				Description: "Allows viewing a list of roles.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.view",
				Name:        "View Role Details",
				Description: "Allows viewing detailed information of a specific role.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.view.users",
				Name:        "View Users in Role",
				Description: "Allows viewing the list of users assigned to a specific role.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "role.edit.users",
				Name:        "Add/Remove Users from Role",
				Description: "Allows adding users to and removing users from a specific role.",
				Scope:       ScopeGlobal,
			},
		},
	},
	{
		Key:         "album",
		Name:        "Album Management",
		Description: "Permissions related to managing albums.",
		Permissions: []PermissionDefinition{
			{
				Key:         "album.create",
				Name:        "Create Album",
				Description: "Allows creating new albums.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "album.edit.general",
				Name:        "Edit Album Details",
				Description: "Allows editing album metadata like name, description, visibility (excluding content).",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "album.delete",
				Name:        "Delete Album",
				Description: "Allows deleting albums.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "album.list",
				Name:        "List Albums",
				Description: "Allows viewing the list of available albums.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "album.view.content",
				Name:        "View Album Content",
				Description: "Allows viewing the photos and videos within an album.",
				Scope:       ScopeAlbum,
			},
			// album-specific permissions that are typically assigned per-album rather than globally
			{
				Key:         "album.photo.upload",
				Name:        "Upload Photos to Album",
				Description: "Allows uploading new photos to a specific album.",
				Scope:       ScopeAlbum,
			},
			{
				Key:         "album.photo.delete",
				Name:        "Delete Photos from Album",
				Description: "Allows deleting photos from a specific album.",
				Scope:       ScopeAlbum,
			},
			{
				Key:         "album.photo.editmeta",
				Name:        "Edit Photo Metadata in Album",
				Description: "Allows editing metadata of photos within a specific album.",
				Scope:       ScopeAlbum,
			},
			{
				Key:         "album.manage.members",
				Name:        "Manage Album Members",
				Description: "Allows adding/removing users or changing their permissions for a specific album.",
				Scope:       ScopeAlbum,
			},
		},
	},
	{
		Key:         "system",
		Name:        "System Administration",
		Description: "High-level system administration permissions.",
		Permissions: []PermissionDefinition{
			{
				Key:         "system.settings.view",
				Name:        "View System Settings",
				Description: "Allows viewing system configuration and settings.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "system.settings.edit",
				Name:        "Edit System Settings",
				Description: "Allows modifying system configuration and settings.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "system.logs.view",
				Name:        "View System Logs",
				Description: "Allows accessing and viewing system logs.",
				Scope:       ScopeGlobal,
			},
		},
	},
	{
		Key:         "invite",
		Name:        "Invite Code Management",
		Description: "Permissions related to managing user registration invite codes.",
		Permissions: []PermissionDefinition{
			{
				Key:         "invite.create",
				Name:        "Create Invite Codes",
				Description: "Allows generating new invite codes.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "invite.list",
				Name:        "List Invite Codes",
				Description: "Allows viewing all existing invite codes.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "invite.view",
				Name:        "View Invite Code Details",
				Description: "Allows viewing the details of a specific invite code.",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "invite.edit",
				Name:        "Edit Invite Codes",
				Description: "Allows modifying existing invite codes (e.g., expiry, max uses, active status).",
				Scope:       ScopeGlobal,
			},
			{
				Key:         "invite.delete",
				Name:        "Delete Invite Codes",
				Description: "Allows deleting invite codes.",
				Scope:       ScopeGlobal,
			},
		},
	},
}

var (
	allPermissionKeysMap map[string]PermissionDefinition
	allPermissionKeys    []string
)

func init() {
	allPermissionKeysMap = make(map[string]PermissionDefinition)
	for _, group := range DefinedPermissionGroups {
		for _, perm := range group.Permissions {
			if _, exists := allPermissionKeysMap[perm.Key]; exists {
				// indicates a duplicate permission key definition, which should be avoided
			}
			allPermissionKeysMap[perm.Key] = perm
			allPermissionKeys = append(allPermissionKeys, perm.Key)
		}
	}
}

// GetAllPermissionDefinitions returns a map of all defined permissions, keyed by their unique string key
func GetAllPermissionDefinitions() map[string]PermissionDefinition {
	return allPermissionKeysMap
}

// GetAllPermissionKeys returns a slice of all unique permission string keys
func GetAllPermissionKeys() []string {
	// return a copy to prevent modification of the internal slice
	keys := make([]string, len(allPermissionKeys))
	copy(keys, allPermissionKeys)
	return keys
}

// IsValidPermissionKey checks if a given permission key is defined
func IsValidPermissionKey(key string) bool {
	_, ok := allPermissionKeysMap[key]
	return ok
}

// GetPermissionDefinition retrieves a specific permission definition by its key.
// Returns the definition and true if found, otherwise an empty definition and false.
func GetPermissionDefinition(key string) (PermissionDefinition, bool) {
	def, ok := allPermissionKeysMap[key]
	return def, ok
}

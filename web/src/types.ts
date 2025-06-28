export interface Album {
    id: number;
    name: string;
    slug: string;
    description?: string;
    location?: string;
    folder_path: string;
    banner_image_path?: string;
    zip_path?: string;
    zip_size?: number;
    zip_status: string;
    zip_last_generated_at?: number;
    zip_last_requested_at?: number;

    created_at: number;
    updated_at: number;
}

export interface FileInfo {
    name: string;
    path: string;
    is_dir: boolean;
    size: number;
    mod_time: number;
    thumbnail_path?: string;
    width?: number;
    height?: number;

    aperture?: number;
    shutter_speed?: string;
    iso?: number;
    focal_length?: number;
    lens_make?: string;
    lens_model?: string;
    camera_make?: string;
    camera_model?: string;
    taken_at?: number;

    thumbnail_status?: string;
    metadata_status?: string;
    detection_status?: string;
}

export interface DirectoryListing {
    path: string;
    files: FileInfo[];
    parent?: string;
}

// Corresponds to backend models.Role (simplified for frontend)
export interface Role {
    id: number;
    name: string;
    // Permissions might be fetched separately or included if small
    global_permissions?: string[];
    global_album_permissions?: string[];
    // album_permissions are likely too complex for a simple Role DTO here
}

// Corresponds to backend models.User (or a UserResponseDTO)
export interface User {
    id: number;
    username: string;
    roles?: Role[]; // Optional, might be just role IDs or full Role objects
    global_permissions?: string[];
    // album_permissions: UserAlbumPermission[]; // Likely fetched on demand
    created_at: string; // Assuming string format like from http.TimeFormat
    updated_at: string;
}

// For login API
export interface LoginPayload {
    username: string;
    password: string;
}

// For register API
export interface RegisterPayload {
    username: string;
    password: string;
    invite_code: string;
}

// Response from login or /me endpoint
export interface AuthResponse {
    token: string;
    user: User; // This User type should match what the backend sends
    expires_at: string; // ISO string
}

// For Admin User Management DTOs (align with backend UserResponseDTO)
export interface UserAlbumPermission {
    id: number;
    user_id: number;
    album_id: number;
    permissions: string[];
    created_at: string;
    updated_at: string;
}

export interface AdminUserResponse extends User {
    album_permissions: UserAlbumPermission[];
}

export interface UserSummary {
    id: number;
    username: string;
}

// For Admin Role Management DTOs (align with backend RoleResponseDTO)
export interface RoleAlbumPermission {
    id: number;
    role_id: number;
    album_id: number;
    permissions: string[];
    created_at: string;
    updated_at: string;
}
export interface AdminRoleResponse extends Role {
    global_album_permissions: string[];
    album_permissions: RoleAlbumPermission[];
    created_at: string;
    updated_at: string;
    users?: UserSummary[]; // Optional list of users
}
export interface UserSummary {
    id: number;
    username: string;
}

// For Admin Invite Code Management DTOs
export interface AdminInviteCodeResponse {
    id: number;
    code: string;
    expires_at?: string; // ISO string
    max_uses?: number;
    uses: number;
    is_active: boolean;
    created_by_user_id: number;
    created_at: string;
    updated_at: string;
}

// Payloads for creating/updating invite codes
export interface InviteCodeCreatePayload {
    expires_at?: string; // ISO 8601 format e.g., "2023-12-31T23:59:59Z" or null
    max_uses?: number; // Nullable for unlimited
    is_active?: boolean; // Defaults to true if not provided
}

export interface InviteCodeUpdatePayload {
    expires_at?: string | null; // Allow sending null to clear
    max_uses?: number | null; // Allow sending null to clear
    is_active?: boolean;
}

// Payloads for creating/updating Roles
export interface RoleAlbumPermissionCreate {
    album_id: number;
    permissions: string[];
}

export interface RoleCreatePayload {
    name: string;
    global_permissions: string[];
    global_album_permissions: string[];
    album_permissions: RoleAlbumPermissionCreate[];
}

export interface RoleAlbumPermissionInput {
    id?: number; // For existing permissions to update
    album_id: number;
    permissions: string[];
}

export interface RoleUpdatePayload {
    name?: string;
    global_permissions?: string[];
    global_album_permissions?: string[];
    album_permissions?: RoleAlbumPermissionInput[]; // Full replacement
}

// For Permission Definitions API
export interface PermissionDefinition {
    key: string;
    name: string;
    description: string;
    scope: 'global' | 'album';
}

export interface PermissionGroupDefinition {
    key: string;
    name: string;
    description: string;
    permissions: PermissionDefinition[];
}

// Stub for album selection in forms, expand as needed
export interface AlbumStub {
    id: number;
    name: string;
}

export interface UserCreatePayload {
    username: string;
    password?: string;
    role_ids: number[];
    global_permissions?: string[];
}

export interface UserUpdatePayload {
    username?: string;
    password?: string;
    role_ids?: number[];
    global_permissions?: string[];
}

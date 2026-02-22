import http from '../http';
import { Album, DirectoryListing } from '../../types';
import { User } from '../../types';
import { ApiResponse } from '../standard';

export interface AdminAlbumResponse extends Album {
    is_hidden: boolean;
    sort_order: string;
}

export interface CreateAlbumPayload {
    name: string;
    slug: string;
    folder_path: string;
    description?: string;
    is_hidden?: boolean;
    location?: string;
    sort_order?: string;
}

export interface UpdateAlbumPayload {
    name?: string;
    description?: string;
    is_hidden?: boolean;
    location?: string;
    sort_order?: string;
}

export const listAlbums = async (): Promise<AdminAlbumResponse[]> => {
    const response = await http.get<ApiResponse<AdminAlbumResponse[]>>('/admin/albums');
    return response.data.data;
};

export const getAlbum = async (id: number): Promise<AdminAlbumResponse> => {
    const response = await http.get<ApiResponse<AdminAlbumResponse>>(`/admin/albums/${id}`);
    return response.data.data;
};

export const createAlbum = async (payload: CreateAlbumPayload): Promise<AdminAlbumResponse> => {
    const response = await http.post<ApiResponse<AdminAlbumResponse>>('/admin/albums', payload);
    return response.data.data;
};

export const updateAlbum = async (id: number, payload: UpdateAlbumPayload): Promise<AdminAlbumResponse> => {
    const response = await http.put<ApiResponse<AdminAlbumResponse>>(`/admin/albums/${id}`, payload);
    return response.data.data;
};

export const deleteAlbum = async (id: number): Promise<void> => {
    await http.delete(`/admin/albums/${id}`);
};

export const uploadAlbumBanner = async (id: number, file: File): Promise<AdminAlbumResponse> => {
    const formData = new FormData();
    formData.append('banner_image', file);

    const response = await http.put<ApiResponse<AdminAlbumResponse>>(`/admin/albums/${id}/banner`, formData, {
        headers: {
            'Content-Type': 'multipart/form-data',
        },
    });
    return response.data.data;
};

export interface UploadResult {
    uploaded: number;
    failed: Array<{ path: string; error: string }>;
}

export const uploadAlbumImages = async (
    id: number,
    files: Array<{ file: File; relativePath?: string }>,
): Promise<UploadResult> => {
    const formData = new FormData();
    for (const item of files) {
        if (item.relativePath) {
            formData.append('relative_path', item.relativePath);
        }
        formData.append('files', item.file, item.relativePath || item.file.name);
    }
    const resp = await http.post<ApiResponse<UploadResult>>(`/admin/albums/${id}/upload`, formData);
    return resp.data.data;
};

export interface UploadAlbumImagesBatchOptions {
    batchSize?: number; // number of files per request
    concurrency?: number; // number of parallel requests
    requestTimeoutMs?: number; // per-request timeout; default disables timeout for large uploads
}

export const uploadAlbumImagesBatched = async (
    id: number,
    files: Array<{ file: File; relativePath?: string }>,
    options: UploadAlbumImagesBatchOptions = {},
): Promise<UploadResult> => {
    const batchSize = options.batchSize ?? 5;
    const concurrency = Math.max(1, options.concurrency ?? 3);
    const requestTimeoutMs = options.requestTimeoutMs ?? 0; // 0 = no timeout

    const batches: Array<Array<{ file: File; relativePath?: string }>> = [];
    for (let i = 0; i < files.length; i += batchSize) {
        batches.push(files.slice(i, i + batchSize));
    }

    let uploadedTotal = 0;
    const failedTotal: Array<{ path: string; error: string }> = [];
    let nextBatchIndex = 0;

    const runOne = async () => {
        while (true) {
            const myIndex = nextBatchIndex++;
            if (myIndex >= batches.length) return;
            const batch = batches[myIndex];

            const formData = new FormData();
            for (const item of batch) {
                if (item.relativePath) {
                    formData.append('relative_path', item.relativePath);
                }
                formData.append('files', item.file, item.relativePath || item.file.name);
            }
            const resp = await http.post<ApiResponse<UploadResult>>(`/admin/albums/${id}/upload`, formData, {
                timeout: requestTimeoutMs,
            });
            uploadedTotal += resp.data?.data?.uploaded ?? 0;
            if (resp.data?.data?.failed) {
                failedTotal.push(...resp.data.data.failed);
            }
        }
    };

    const workers: Promise<void>[] = [];
    for (let i = 0; i < concurrency; i++) {
        workers.push(runOne());
    }
    await Promise.all(workers);

    return { uploaded: uploadedTotal, failed: failedTotal };
};

export const listAlbumImages = async (id: number): Promise<DirectoryListing> => {
    const resp = await http.get<ApiResponse<DirectoryListing>>(`/admin/albums/${id}/images`);
    return resp.data.data;
};

export const deleteAlbumImage = async (id: number, imagePath: string): Promise<void> => {
    // imagePath should be full relative path (e.g., "album/folder/IMG_1234.jpg")
    await http.delete(`/admin/albums/${id}/images`, { params: { path: imagePath } });
};

export const requestAlbumZip = async (id: number): Promise<{ message: string }> => {
    const response = await http.post<ApiResponse<{ message: string }>>(`/admin/albums/${id}/zip`);
    return response.data.data;
};

export const downloadAlbumZip = async (id: number): Promise<Blob> => {
    const response = await http.get(`/admin/albums/${id}/zip`, {
        responseType: 'blob',
    });
    return response.data;
};

export interface AlbumUserPermissionResponse {
    user: User;
    permissions: string[];
    direct_permissions?: string[];
    inherited_permissions?: string[];
    role_contributions?: Array<{
        role_id: number;
        role_name: string;
        for_all_albums?: string[];
        album_specific?: string[];
    }>;
    user_album_permission?: {
        id: number;
        user_id: number;
        album_id: number;
        permissions: string[];
        created_at: string;
        updated_at: string;
    };
}

export interface AddUserToAlbumPayload {
    user_id: number;
    permissions: string[];
}

export interface UpdateUserAlbumPermissionsPayload {
    permissions: string[];
}

export const getAlbumUsers = async (albumId: number): Promise<AlbumUserPermissionResponse[]> => {
    const response = await http.get<ApiResponse<AlbumUserPermissionResponse[]>>(`/admin/albums/${albumId}/users`);
    return response.data.data;
};

export const getAvailableUsers = async (albumId: number): Promise<User[]> => {
    const response = await http.get<ApiResponse<User[]>>(`/admin/albums/${albumId}/users/available`);
    return response.data.data;
};

export const addUserToAlbum = async (albumId: number, payload: AddUserToAlbumPayload): Promise<any> => {
    const response = await http.post(`/admin/albums/${albumId}/users`, payload);
    return response.data.data;
};

export const updateUserAlbumPermissions = async (
    albumId: number,
    userId: number,
    payload: UpdateUserAlbumPermissionsPayload,
): Promise<any> => {
    const response = await http.put(`/admin/albums/${albumId}/users/${userId}`, payload);
    return response.data.data;
};

export const removeUserFromAlbum = async (albumId: number, userId: number): Promise<void> => {
    await http.delete(`/admin/albums/${albumId}/users/${userId}`);
};

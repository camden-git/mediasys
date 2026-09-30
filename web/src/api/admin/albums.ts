import http from '../http';
import { ApiError, isAbortError } from '../errors';
import { Album, AlbumBanner, DirectoryListing } from '../../types';
import { User } from '../../types';
import { ApiResponse } from '../standard';

export interface AdminAlbumResponse extends Omit<Album, 'banners'> {
    is_hidden: boolean;
    sort_order: string;
    banners: AlbumBanner[];
    zip_path?: string;
    zip_error?: string;
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

export const listAlbums = async (signal?: AbortSignal): Promise<AdminAlbumResponse[]> => {
    const response = await http.get<ApiResponse<AdminAlbumResponse[]>>('/admin/albums', { signal });
    return response.data.data;
};

export const getAlbumBySlug = async (slug: string, signal?: AbortSignal): Promise<AdminAlbumResponse> => {
    const response = await http.get<ApiResponse<AdminAlbumResponse>>(`/admin/albums/${encodeURIComponent(slug)}`, {
        signal,
    });
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

export const addAlbumBanner = async (id: number, file: File): Promise<AlbumBanner> => {
    const formData = new FormData();
    formData.append('banner_image', file);
    const response = await http.post<ApiResponse<AlbumBanner>>(`/admin/albums/${id}/banners`, formData);
    return response.data.data;
};

export const deleteAlbumBanner = async (id: number, bannerId: number): Promise<void> => {
    await http.delete(`/admin/albums/${id}/banners/${bannerId}`);
};

export const reorderAlbumBanners = async (id: number, bannerIds: number[]): Promise<void> => {
    await http.put(`/admin/albums/${id}/banners/order`, { banner_ids: bannerIds });
};

interface UploadResult {
    uploaded: number;
    failed: Array<{ path: string; error: string }>;
}

interface UploadAlbumImagesBatchOptions {
    batchSize?: number; // max number of files per request
    maxBatchBytes?: number; // max total bytes per request (default: 95 MB, below Cloudflare free plan 100 MB limit)
    concurrency?: number; // number of parallel requests
    requestTimeoutMs?: number; // per-request timeout
    maxRetries?: number; // per-batch retry count, default 3
    signal?: AbortSignal; // for cancellation
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

const uploadBatchWithRetry = async (
    id: number,
    batch: Array<{ file: File; relativePath?: string }>,
    { signal, timeout, maxRetries }: { signal?: AbortSignal; timeout: number; maxRetries: number },
): Promise<UploadResult> => {
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
        if (signal?.aborted) throw new ApiError('Request cancelled', { isAbort: true });
        try {
            const formData = new FormData();
            for (const item of batch) {
                if (item.relativePath) formData.append('relative_path', item.relativePath);
                formData.append('files', item.file, item.relativePath || item.file.name);
            }
            const resp = await http.post<ApiResponse<UploadResult>>(`/admin/albums/${id}/upload`, formData, {
                timeout,
                signal,
            });
            return resp.data.data;
        } catch (err: unknown) {
            const isClientError = err instanceof ApiError && err.isClientError;
            if (isAbortError(err) || isClientError || attempt === maxRetries) throw err;
            await sleep(Math.min(1000 * 2 ** attempt, 10_000)); // 1s, 2s, 4s … max 10s
        }
    }
    throw new Error('Max retries exceeded');
};

export const uploadAlbumImagesBatched = async (
    id: number,
    files: Array<{ file: File; relativePath?: string }>,
    options: UploadAlbumImagesBatchOptions = {},
): Promise<UploadResult> => {
    const batchSize = options.batchSize ?? Infinity;
    const maxBatchBytes = options.maxBatchBytes ?? 95 * 1024 * 1024;
    const concurrency = Math.max(1, options.concurrency ?? 3);
    const requestTimeoutMs = options.requestTimeoutMs ?? 5 * 60 * 1000; // 5 min default
    const maxRetries = options.maxRetries ?? 3;
    const signal = options.signal;

    const batches: Array<Array<{ file: File; relativePath?: string }>> = [];
    let current: Array<{ file: File; relativePath?: string }> = [];
    let currentBytes = 0;
    for (const item of files) {
        if (current.length > 0 && (current.length >= batchSize || currentBytes + item.file.size > maxBatchBytes)) {
            batches.push(current);
            current = [];
            currentBytes = 0;
        }
        current.push(item);
        currentBytes += item.file.size;
    }
    if (current.length > 0) batches.push(current);

    let uploadedTotal = 0;
    const failedTotal: Array<{ path: string; error: string }> = [];
    let nextBatchIndex = 0;

    const runOne = async () => {
        while (true) {
            if (signal?.aborted) return;
            const myIndex = nextBatchIndex++;
            if (myIndex >= batches.length) return;
            const batch = batches[myIndex];
            try {
                const result = await uploadBatchWithRetry(id, batch, { signal, timeout: requestTimeoutMs, maxRetries });
                uploadedTotal += result?.uploaded ?? 0;
                if (result?.failed) failedTotal.push(...result.failed);
            } catch (err: unknown) {
                if (isAbortError(err)) return;
                for (const item of batch) {
                    failedTotal.push({
                        path: item.relativePath || item.file.name,
                        error: err instanceof Error ? err.message : 'Upload failed',
                    });
                }
            }
        }
    };

    const workers: Promise<void>[] = [];
    for (let i = 0; i < concurrency; i++) {
        workers.push(runOne());
    }
    await Promise.allSettled(workers);

    return { uploaded: uploadedTotal, failed: failedTotal };
};

export const listAlbumImages = async (id: number, signal?: AbortSignal): Promise<DirectoryListing> => {
    const resp = await http.get<ApiResponse<DirectoryListing>>(`/admin/albums/${id}/images`, { signal });
    return resp.data.data;
};

export const deleteAlbumImage = async (id: number, imagePath: string): Promise<void> => {
    // imagePath should be full relative path (e.g., "album/folder/IMG_1234.jpg")
    await http.delete(`/admin/albums/${id}/images`, { params: { path: imagePath } });
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

export const getAlbumUsers = async (albumId: number, signal?: AbortSignal): Promise<AlbumUserPermissionResponse[]> => {
    const response = await http.get<ApiResponse<AlbumUserPermissionResponse[]>>(`/admin/albums/${albumId}/users`, {
        signal,
    });
    return response.data.data;
};

export const getAvailableUsers = async (albumId: number, signal?: AbortSignal): Promise<User[]> => {
    const response = await http.get<ApiResponse<User[]>>(`/admin/albums/${albumId}/users/available`, { signal });
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

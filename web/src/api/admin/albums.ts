import http from '../http';
import { Album } from '../../types';
import { User } from '../../types';

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
    const response = await http.get('/admin/albums');
    return response.data;
};

export const getAlbum = async (id: number): Promise<AdminAlbumResponse> => {
    const response = await http.get(`/admin/albums/${id}`);
    return response.data;
};

export const createAlbum = async (payload: CreateAlbumPayload): Promise<AdminAlbumResponse> => {
    const response = await http.post('/admin/albums', payload);
    return response.data;
};

export const updateAlbum = async (id: number, payload: UpdateAlbumPayload): Promise<AdminAlbumResponse> => {
    const response = await http.put(`/admin/albums/${id}`, payload);
    return response.data;
};

export const deleteAlbum = async (id: number): Promise<void> => {
    await http.delete(`/admin/albums/${id}`);
};

export const uploadAlbumBanner = async (id: number, file: File): Promise<AdminAlbumResponse> => {
    const formData = new FormData();
    formData.append('banner_image', file);

    const response = await http.put(`/admin/albums/${id}/banner`, formData, {
        headers: {
            'Content-Type': 'multipart/form-data',
        },
    });
    return response.data;
};

export const requestAlbumZip = async (id: number): Promise<{ message: string }> => {
    const response = await http.post(`/admin/albums/${id}/zip`);
    return response.data;
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
    const response = await http.get(`/admin/albums/${albumId}/users`);
    return response.data;
};

export const getAvailableUsers = async (albumId: number): Promise<User[]> => {
    const response = await http.get(`/admin/albums/${albumId}/users/available`);
    return response.data;
};

export const addUserToAlbum = async (albumId: number, payload: AddUserToAlbumPayload): Promise<any> => {
    const response = await http.post(`/admin/albums/${albumId}/users`, payload);
    return response.data;
};

export const updateUserAlbumPermissions = async (
    albumId: number,
    userId: number,
    payload: UpdateUserAlbumPermissionsPayload,
): Promise<any> => {
    const response = await http.put(`/admin/albums/${albumId}/users/${userId}`, payload);
    return response.data;
};

export const removeUserFromAlbum = async (albumId: number, userId: number): Promise<void> => {
    await http.delete(`/admin/albums/${albumId}/users/${userId}`);
};

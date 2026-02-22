import http from '../http';
import { AlbumGroup } from '../../types';
import { ApiResponse } from '../standard';

export interface GroupCreatePayload {
    name: string;
    slug: string;
    description?: string;
    is_hidden?: boolean;
}

export interface GroupUpdatePayload {
    name?: string;
    slug?: string;
    description?: string;
    is_hidden?: boolean;
}

export const listGroups = async (): Promise<AlbumGroup[]> => {
    const response = await http.get<ApiResponse<AlbumGroup[]>>('/admin/groups/');
    return response.data.data;
};

export const getGroup = async (id: number): Promise<AlbumGroup> => {
    const response = await http.get<ApiResponse<AlbumGroup>>(`/admin/groups/${id}`);
    return response.data.data;
};

export const createGroup = async (payload: GroupCreatePayload): Promise<AlbumGroup> => {
    const response = await http.post<ApiResponse<AlbumGroup>>('/admin/groups/', payload);
    return response.data.data;
};

export const updateGroup = async (id: number, payload: GroupUpdatePayload): Promise<AlbumGroup> => {
    const response = await http.put<ApiResponse<AlbumGroup>>(`/admin/groups/${id}`, payload);
    return response.data.data;
};

export const deleteGroup = async (id: number): Promise<void> => {
    await http.delete(`/admin/groups/${id}`);
};

export const setAlbumGroup = async (albumId: number, groupId: number | null): Promise<void> => {
    await http.put(`/admin/albums/${albumId}/group`, { group_id: groupId });
};

export const uploadGroupBanner = async (id: number, file: File): Promise<AlbumGroup> => {
    const formData = new FormData();
    formData.append('banner_image', file);
    const response = await http.put<ApiResponse<AlbumGroup>>(`/admin/groups/${id}/banner`, formData);
    return response.data.data;
};

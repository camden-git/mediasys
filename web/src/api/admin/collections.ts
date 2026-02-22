import http from '../http';
import { Collection, CollectionTagFilter } from '../../types';
import { ApiResponse } from '../standard';

export interface CollectionCreatePayload {
    name: string;
    slug: string;
    description?: string;
    is_public?: boolean;
}

export interface CollectionUpdatePayload {
    name?: string;
    slug?: string;
    description?: string;
    is_public?: boolean;
}

export const listCollections = async (): Promise<Collection[]> => {
    const response = await http.get<ApiResponse<Collection[]>>('/admin/collections/');
    return response.data.data;
};

export const getCollection = async (id: number): Promise<Collection> => {
    const response = await http.get<ApiResponse<Collection>>(`/admin/collections/${id}`);
    return response.data.data;
};

export const createCollection = async (payload: CollectionCreatePayload): Promise<Collection> => {
    const response = await http.post<ApiResponse<Collection>>('/admin/collections/', payload);
    return response.data.data;
};

export const updateCollection = async (id: number, payload: CollectionUpdatePayload): Promise<Collection> => {
    const response = await http.put<ApiResponse<Collection>>(`/admin/collections/${id}`, payload);
    return response.data.data;
};

export const deleteCollection = async (id: number): Promise<void> => {
    await http.delete(`/admin/collections/${id}`);
};

export const uploadCollectionBanner = async (id: number, file: File): Promise<Collection> => {
    const formData = new FormData();
    formData.append('banner_image', file);
    const response = await http.put<ApiResponse<Collection>>(`/admin/collections/${id}/banner`, formData);
    return response.data.data;
};

export const setCollectionFilters = async (
    id: number,
    filters: Array<{ tag_key: string; tag_value: string }>,
): Promise<Collection> => {
    const response = await http.put<ApiResponse<Collection>>(`/admin/collections/${id}/filters`, { filters });
    return response.data.data;
};

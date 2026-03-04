import http from '../http';
import { Collection, CollectionBanner } from '../../types';
import { ApiResponse } from '../standard';

export interface AdminCollectionResponse extends Omit<Collection, 'banners'> {
    banners: CollectionBanner[];
}

export interface CollectionCreatePayload {
    name: string;
    slug: string;
    description?: string;
    is_public?: boolean;
    filter_match?: 'all' | 'any';
}

export interface CollectionUpdatePayload {
    name?: string;
    slug?: string;
    description?: string;
    is_public?: boolean;
    filter_match?: 'all' | 'any';
}

export const listCollections = async (): Promise<AdminCollectionResponse[]> => {
    const response = await http.get<ApiResponse<AdminCollectionResponse[]>>('/admin/collections/');
    return response.data.data;
};

export const getCollection = async (id: number): Promise<AdminCollectionResponse> => {
    const response = await http.get<ApiResponse<AdminCollectionResponse>>(`/admin/collections/${id}`);
    return response.data.data;
};

export const createCollection = async (payload: CollectionCreatePayload): Promise<AdminCollectionResponse> => {
    const response = await http.post<ApiResponse<AdminCollectionResponse>>('/admin/collections/', payload);
    return response.data.data;
};

export const updateCollection = async (
    id: number,
    payload: CollectionUpdatePayload,
): Promise<AdminCollectionResponse> => {
    const response = await http.put<ApiResponse<AdminCollectionResponse>>(`/admin/collections/${id}`, payload);
    return response.data.data;
};

export const deleteCollection = async (id: number): Promise<void> => {
    await http.delete(`/admin/collections/${id}`);
};

export const addCollectionBanner = async (id: number, file: File): Promise<CollectionBanner> => {
    const formData = new FormData();
    formData.append('banner_image', file);
    const response = await http.post<ApiResponse<CollectionBanner>>(`/admin/collections/${id}/banners`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data.data;
};

export const deleteCollectionBanner = async (id: number, bannerId: number): Promise<void> => {
    await http.delete(`/admin/collections/${id}/banners/${bannerId}`);
};

export const reorderCollectionBanners = async (id: number, bannerIds: number[]): Promise<void> => {
    await http.put(`/admin/collections/${id}/banners/order`, { banner_ids: bannerIds });
};

export const setCollectionInheritBanners = async (id: number, inherit: boolean): Promise<AdminCollectionResponse> => {
    const response = await http.put<ApiResponse<AdminCollectionResponse>>(`/admin/collections/${id}/banners/inherit`, {
        inherit,
    });
    return response.data.data;
};

export const setCollectionFilters = async (
    id: number,
    filters: Array<{ tag_key: string; tag_value: string; negate: boolean }>,
): Promise<AdminCollectionResponse> => {
    const response = await http.put<ApiResponse<AdminCollectionResponse>>(`/admin/collections/${id}/filters`, {
        filters,
    });
    return response.data.data;
};

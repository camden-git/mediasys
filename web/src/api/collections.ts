import http from './http';
import { Collection, DirectoryListing } from '../types';
import { ApiResponse } from './standard';

export const listPublicCollections = async (): Promise<Collection[]> => {
    const response = await http.get<ApiResponse<Collection[]>>('/collections/');
    return response.data.data;
};

export const getPublicCollection = async (slug: string): Promise<Collection> => {
    const response = await http.get<ApiResponse<Collection>>(`/collections/${slug}`);
    return response.data.data;
};

export const getCollectionPhotos = async (slug: string, offset = 0, limit = 120): Promise<DirectoryListing> => {
    const response = await http.get<ApiResponse<DirectoryListing>>(`/collections/${slug}/photos`, {
        params: { offset, limit },
    });
    return response.data.data;
};

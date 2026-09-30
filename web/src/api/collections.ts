import http, { unwrap } from './http';
import { Collection, DirectoryListing } from '../types';
import { ApiResponse } from './standard';

export const listPublicCollections = async (signal?: AbortSignal): Promise<Collection[]> =>
    unwrap(await http.get<ApiResponse<Collection[]>>('/collections/', { signal }));

export const getPublicCollection = async (slug: string, signal?: AbortSignal): Promise<Collection> =>
    unwrap(await http.get<ApiResponse<Collection>>(`/collections/${encodeURIComponent(slug)}`, { signal }));

export const getCollectionPhotos = async (
    slug: string,
    offset = 0,
    limit = 120,
    signal?: AbortSignal,
): Promise<DirectoryListing> =>
    unwrap(
        await http.get<ApiResponse<DirectoryListing>>(`/collections/${encodeURIComponent(slug)}/photos`, {
            params: { offset, limit },
            signal,
        }),
    );

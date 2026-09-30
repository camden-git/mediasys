import http, { unwrap } from './http';
import { Album, DirectoryListing } from '../types';
import { ApiResponse } from './standard';

export interface ListingParams {
    offset?: number;
    limit?: number;
    min_rating?: number;
}

export const getAlbums = async (signal?: AbortSignal): Promise<Album[]> =>
    unwrap(await http.get<ApiResponse<Album[]>>('/albums', { signal }));

export const getAlbumDetails = async (identifier: string, signal?: AbortSignal): Promise<Album> =>
    unwrap(await http.get<ApiResponse<Album>>(`/albums/${encodeURIComponent(identifier)}`, { signal }));

export const getAlbumContents = async (
    identifier: string,
    params?: ListingParams,
    signal?: AbortSignal,
): Promise<DirectoryListing> =>
    unwrap(
        await http.get<ApiResponse<DirectoryListing>>(`/albums/${encodeURIComponent(identifier)}/contents`, {
            params,
            signal,
        }),
    );

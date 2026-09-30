import http, { unwrap } from './http';
import { AlbumGroup, DirectoryListing } from '../types';
import { ApiResponse } from './standard';
import type { ListingParams } from './albums';

export const getGroups = async (signal?: AbortSignal): Promise<AlbumGroup[]> =>
    unwrap(await http.get<ApiResponse<AlbumGroup[]>>('/groups', { signal }));

export const getGroup = async (slug: string, signal?: AbortSignal): Promise<AlbumGroup> =>
    unwrap(await http.get<ApiResponse<AlbumGroup>>(`/groups/${encodeURIComponent(slug)}`, { signal }));

export const getGroupPhotos = async (
    slug: string,
    params?: ListingParams,
    signal?: AbortSignal,
): Promise<DirectoryListing> =>
    unwrap(
        await http.get<ApiResponse<DirectoryListing>>(`/groups/${encodeURIComponent(slug)}/photos`, {
            params,
            signal,
        }),
    );

import http from '../http';
import { AlbumDefaultTag } from '../../types';
import { ApiResponse } from '../standard';

export const getAlbumDefaultTags = async (albumId: number): Promise<AlbumDefaultTag[]> => {
    const response = await http.get<ApiResponse<AlbumDefaultTag[]>>(`/admin/albums/${albumId}/default-tags`);
    return response.data.data;
};

export const setAlbumDefaultTags = async (
    albumId: number,
    tags: Array<{ tag_key: string; tag_value: string }>,
): Promise<AlbumDefaultTag[]> => {
    const response = await http.put<ApiResponse<AlbumDefaultTag[]>>(`/admin/albums/${albumId}/default-tags`, { tags });
    return response.data.data;
};

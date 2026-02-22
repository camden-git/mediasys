import http from '../http';
import { ImageTag, AlbumDefaultTag } from '../../types';
import { ApiResponse } from '../standard';

export const getImageTags = async (imagePath: string): Promise<ImageTag[]> => {
    const response = await http.get<ApiResponse<ImageTag[]>>('/admin/images/tags', {
        params: { path: imagePath },
    });
    return response.data.data;
};

export const addManualTag = async (imagePath: string, tagKey: string, tagValue: string): Promise<ImageTag[]> => {
    const response = await http.post<ApiResponse<ImageTag[]>>(
        '/admin/images/tags',
        {
            tag_key: tagKey,
            tag_value: tagValue,
        },
        {
            params: { path: imagePath },
        },
    );
    return response.data.data;
};

export const removeManualTag = async (imagePath: string, tagKey: string, tagValue: string): Promise<void> => {
    await http.delete('/admin/images/tags', {
        params: { path: imagePath },
        data: { tag_key: tagKey, tag_value: tagValue },
    });
};

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

import http, { unwrap } from './http';
import { FaceData, PersonImageResult, UntaggedFaceResult } from '../types';
import { ApiResponse } from './standard';

export const getFacesForImage = async (imagePath: string, signal?: AbortSignal): Promise<FaceData[]> =>
    unwrap(await http.get<ApiResponse<FaceData[]>>('/images/faces', { params: { path: imagePath }, signal }));

export const searchFacesByName = async (query: string, signal?: AbortSignal): Promise<PersonImageResult[]> =>
    unwrap(await http.get<ApiResponse<PersonImageResult[]>>('/search/faces', { params: { query }, signal }));

// Face management (admin)
export interface UntaggedFaceParams {
    limit?: number;
    min_quality?: number;
    min_confidence?: number;
    sort_by?: 'quality' | 'confidence' | 'created_at';
    sort_order?: 'asc' | 'desc';
    group_by_image?: boolean;
    album_id?: number;
}

export const getUntaggedFaces = async (
    params: UntaggedFaceParams = {},
    signal?: AbortSignal,
): Promise<UntaggedFaceResult[]> =>
    unwrap(
        await http.get<ApiResponse<UntaggedFaceResult[]>>('/faces/untagged', {
            params: { ...params, group_by_image: params.group_by_image || undefined },
            signal,
        }),
    );

export const tagFace = async (faceId: number, personId: number): Promise<void> => {
    await http.post(`/faces/${faceId}/tag`, { person_id: personId });
};

export const deleteFace = async (faceId: number): Promise<void> => {
    await http.delete(`/faces/${faceId}`);
};

export interface FaceSuggestion {
    suggested_person_id: number | null;
    suggested_person_name: string | null;
    suggestion_count: number;
    confidence: number;
}

export const suggestFace = async (faceId: number, signal?: AbortSignal): Promise<FaceSuggestion> =>
    unwrap(await http.get<ApiResponse<FaceSuggestion>>(`/faces/${faceId}/suggest`, { signal }));

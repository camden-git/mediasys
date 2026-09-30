import {
    Album,
    Alias,
    AlbumGroup,
    DirectoryListing,
    FaceData,
    LoginPayload,
    Person,
    PersonImageResult,
    RegisterPayload,
    UntaggedFaceResult,
    User,
    AuthResponse,
} from './types';
import { ApiErrorDetail } from './api/standard';
import { handleUnauthorized } from './api/unauthorized';

const getAuthToken = (): string | null => localStorage.getItem('authToken');

const apiClient = async (url: string, options: RequestInit = {}, signal?: AbortSignal): Promise<Response> => {
    const token = getAuthToken();
    const requestHeaders: Record<string, string> = {
        'Content-Type': 'application/json',
    };

    // spread existing headers from options if they exist
    if (options.headers) {
        if (options.headers instanceof Headers) {
            options.headers.forEach((value, key) => {
                requestHeaders[key] = value;
            });
        } else if (Array.isArray(options.headers)) {
            options.headers.forEach(([key, value]) => {
                requestHeaders[key] = value;
            });
        } else {
            for (const key in options.headers) {
                if (Object.prototype.hasOwnProperty.call(options.headers, key)) {
                    requestHeaders[key] = (options.headers as Record<string, string>)[key];
                }
            }
        }
    }

    if (token) {
        requestHeaders['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${import.meta.env.VITE_API_URL}${url}`, {
        ...options,
        headers: requestHeaders as HeadersInit, // cast to HeadersInit for fetch
        signal,
    });

    if (!response.ok) {
        let errorMessage = `HTTP error! status: ${response.status}`;
        let standardizedErrors: ApiErrorDetail[] | null = null;

        // The backend always returns errors as {"errors": [{code, status, detail}]}.
        try {
            const errorBody = await response.clone().json();
            if (Array.isArray(errorBody?.errors) && errorBody.errors.length > 0) {
                standardizedErrors = errorBody.errors;
                if (standardizedErrors![0]?.detail) {
                    errorMessage = standardizedErrors![0].detail;
                }
            }
        } catch {
            // ignore, fall back to the generic status message
        }

        handleUnauthorized(url, response.status);

        const error = new Error(errorMessage);
        (error as any).status = response.status;
        if (standardizedErrors) {
            (error as any).errors = standardizedErrors;
        }
        throw error;
    }
    return response;
};

export const getAlbums = async (): Promise<Album[]> => {
    const response = await apiClient(`/albums`);
    return (await response.json()) as Album[];
};

export const getAlbumDetails = async (identifier: string, signal?: AbortSignal): Promise<Album> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const response = await apiClient(`/albums/${encodedIdentifier}`, {}, signal);
    return (await response.json()) as Album;
};

export const getAlbumContents = async (
    identifier: string,
    params?: { offset?: number; limit?: number },
    signal?: AbortSignal,
): Promise<DirectoryListing> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const search = new URLSearchParams();
    if (params?.offset !== undefined) search.set('offset', String(params.offset));
    if (params?.limit !== undefined) search.set('limit', String(params.limit));
    const qs = search.toString();
    const response = await apiClient(`/albums/${encodedIdentifier}/contents${qs ? `?${qs}` : ''}`, {}, signal);
    return (await response.json()) as DirectoryListing;
};

// auth
export const loginUser = async (payload: LoginPayload): Promise<AuthResponse> => {
    const response = await apiClient('/auth/login', {
        method: 'POST',
        body: JSON.stringify(payload),
    });
    return (await response.json()) as AuthResponse;
};

export const registerUser = async (payload: RegisterPayload): Promise<{ message: string }> => {
    const response = await apiClient('/auth/register', {
        method: 'POST',
        body: JSON.stringify(payload),
    });
    return (await response.json()) as { message: string }; // Assuming backend returns a message on successful registration
};

export const getCurrentUser = async (): Promise<User> => {
    const response = await apiClient('/auth/me');
    return (await response.json()) as User;
};

export const getThumbnailUrl = (thumbnailPath: string): string => {
    if (/^https?:\/\//.test(thumbnailPath)) return thumbnailPath;
    return `${import.meta.env.VITE_BACKEND_URL}${thumbnailPath}`;
};

export const getBannerUrl = (bannerPath: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/${bannerPath}`;
};

// encodes each segment of an image path for use in a URL
const encodeImagePath = (imagePath: string): string =>
    imagePath.replace(/^\/+/, '').split('/').map(encodeURIComponent).join('/');

export const getOriginalImageUrl = (imagePath: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/originals/${encodeImagePath(imagePath)}`;
};

export const getAlbumDownloadUrl = (id: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/albums/${id}/zip`;
};

export const getPreviewImagePath = (imagePath: string): string => `/preview/${encodeImagePath(imagePath)}`;

export const getPreviewImageUrl = (imagePath: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}${getPreviewImagePath(imagePath)}`;
};

export const getFacesForImage = async (imagePath: string): Promise<FaceData[]> => {
    const response = await apiClient(`/images/faces?path=${encodeURIComponent(imagePath)}`);
    return (await response.json()) as FaceData[];
};

// Album Groups
export const getGroups = async (signal?: AbortSignal): Promise<AlbumGroup[]> => {
    const response = await apiClient('/groups', {}, signal);
    const body = (await response.json()) as { data: AlbumGroup[] };
    return body.data;
};

export const getGroup = async (slug: string, signal?: AbortSignal): Promise<AlbumGroup> => {
    const response = await apiClient(`/groups/${encodeURIComponent(slug)}`, {}, signal);
    const body = (await response.json()) as { data: AlbumGroup };
    return body.data;
};

export const getGroupPhotos = async (
    slug: string,
    params?: { offset?: number; limit?: number; min_rating?: number },
    signal?: AbortSignal,
): Promise<DirectoryListing> => {
    const search = new URLSearchParams();
    if (params?.offset !== undefined) search.set('offset', String(params.offset));
    if (params?.limit !== undefined) search.set('limit', String(params.limit));
    if (params?.min_rating !== undefined) search.set('min_rating', String(params.min_rating));
    const qs = search.toString();
    const response = await apiClient(`/groups/${encodeURIComponent(slug)}/photos${qs ? `?${qs}` : ''}`, {}, signal);
    const body = (await response.json()) as { data: DirectoryListing };
    return body.data;
};

// People
export const getPeople = async (signal?: AbortSignal): Promise<Person[]> => {
    const response = await apiClient('/people', {}, signal);
    return (await response.json()) as Person[];
};

export const getPersonById = async (id: number, signal?: AbortSignal): Promise<Person> => {
    const response = await apiClient(`/people/${id}`, {}, signal);
    return (await response.json()) as Person;
};

/** Admin variant of getPersonById that includes faces from hidden albums (requires people.manage). */
export const getPersonByIdAdmin = async (id: number, signal?: AbortSignal): Promise<Person> => {
    const response = await apiClient(`/people/${id}/admin`, {}, signal);
    return (await response.json()) as Person;
};

export interface PersonImagesPage {
    items: PersonImageResult[];
    total: number;
    offset: number;
    limit: number;
    has_more: boolean;
}

export const getPersonImages = async (
    personId: number,
    params: { offset?: number; limit?: number } = {},
    signal?: AbortSignal,
): Promise<PersonImagesPage> => {
    const qs = new URLSearchParams();
    if (params.offset !== undefined) qs.set('offset', String(params.offset));
    if (params.limit !== undefined) qs.set('limit', String(params.limit));
    const response = await apiClient(`/people/${personId}/images?${qs}`, {}, signal);
    return (await response.json()) as PersonImagesPage;
};

export const searchFacesByName = async (query: string, signal?: AbortSignal): Promise<PersonImageResult[]> => {
    const response = await apiClient(`/search/faces?query=${encodeURIComponent(query)}`, {}, signal);
    return (await response.json()) as PersonImageResult[];
};

export const searchPeople = async (q: string, limit = 5, signal?: AbortSignal): Promise<Person[]> => {
    const response = await apiClient(`/people/search?q=${encodeURIComponent(q)}&limit=${limit}`, {}, signal);
    return (await response.json()) as Person[];
};

// People management (admin)
export const createPerson = async (name: string): Promise<Person> => {
    const response = await apiClient('/people', {
        method: 'POST',
        body: JSON.stringify({ primary_name: name }),
    });
    return (await response.json()) as Person;
};

export const updatePerson = async (id: number, name: string): Promise<Person> => {
    const response = await apiClient(`/people/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ primary_name: name }),
    });
    return (await response.json()) as Person;
};

export const deletePerson = async (id: number): Promise<void> => {
    await apiClient(`/people/${id}`, { method: 'DELETE' });
};

export const addPersonAlias = async (personId: number, name: string): Promise<Alias> => {
    const response = await apiClient(`/people/${personId}/aliases`, {
        method: 'POST',
        body: JSON.stringify({ name }),
    });
    return (await response.json()) as Alias;
};

export const deletePersonAlias = async (personId: number, aliasId: number): Promise<void> => {
    await apiClient(`/people/${personId}/aliases/${aliasId}`, { method: 'DELETE' });
};

export const getPersonAliases = async (personId: number, signal?: AbortSignal): Promise<Alias[]> => {
    const response = await apiClient(`/people/${personId}/aliases`, {}, signal);
    return (await response.json()) as Alias[];
};

/** Returns the URL for a person's 128×128 key photo thumbnail. */
export const getPersonKeyPhotoUrl = (personId: number): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/people/${personId}/key-photo.jpg`;
};

/** Sets (or clears) the key photo for a person. Pass null to clear. */
export const setPersonKeyPhoto = async (personId: number, faceId: number | null): Promise<Person> => {
    const response = await apiClient(`/people/${personId}/key-photo`, {
        method: 'PUT',
        body: JSON.stringify({ face_id: faceId ?? 0 }),
    });
    const body = (await response.json()) as { data: Person };
    return body.data;
};

// Face management (admin)
export interface UntaggedFaceParams {
    limit?: number;
    min_quality?: number;
    min_confidence?: number;
    sort_by?: 'quality' | 'confidence' | 'created_at';
    sort_order?: 'asc' | 'desc';
    group_by_image?: boolean;
}

export const getUntaggedFaces = async (
    params: UntaggedFaceParams = {},
    signal?: AbortSignal,
): Promise<UntaggedFaceResult[]> => {
    const qs = new URLSearchParams();
    if (params.limit != null) qs.set('limit', String(params.limit));
    if (params.min_quality != null) qs.set('min_quality', String(params.min_quality));
    if (params.min_confidence != null) qs.set('min_confidence', String(params.min_confidence));
    if (params.sort_by) qs.set('sort_by', params.sort_by);
    if (params.sort_order) qs.set('sort_order', params.sort_order);
    if (params.group_by_image) qs.set('group_by_image', 'true');
    const response = await apiClient(`/faces/untagged?${qs}`, {}, signal);
    return ((await response.json()) ?? []) as UntaggedFaceResult[];
};

export const tagFace = async (faceId: number, personId: number): Promise<void> => {
    await apiClient(`/faces/${faceId}/tag`, {
        method: 'POST',
        body: JSON.stringify({ person_id: personId }),
    });
};

export const deleteFace = async (faceId: number): Promise<void> => {
    await apiClient(`/faces/${faceId}`, { method: 'DELETE' });
};

export const suggestFace = async (
    faceId: number,
): Promise<{
    suggested_person_id: number | null;
    suggested_person_name: string | null;
    suggestion_count: number;
    confidence: number;
}> => {
    const response = await apiClient(`/faces/${faceId}/suggest`);
    return await response.json();
};

export const getAlbumContentsWithRating = async (
    identifier: string,
    params?: { offset?: number; limit?: number; min_rating?: number },
    signal?: AbortSignal,
): Promise<DirectoryListing> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const search = new URLSearchParams();
    if (params?.offset !== undefined) search.set('offset', String(params.offset));
    if (params?.limit !== undefined) search.set('limit', String(params.limit));
    if (params?.min_rating !== undefined) search.set('min_rating', String(params.min_rating));
    const qs = search.toString();
    const response = await apiClient(`/albums/${encodedIdentifier}/contents${qs ? `?${qs}` : ''}`, {}, signal);
    return (await response.json()) as DirectoryListing;
};

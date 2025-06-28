import { Album, DirectoryListing, LoginPayload, RegisterPayload, User, AuthResponse } from './types';

const getAuthToken = (): string | null => localStorage.getItem('authToken');

const apiClient = async (url: string, options: RequestInit = {}): Promise<Response> => {
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
    });

    if (!response.ok) {
        let errorMessage = `HTTP error! status: ${response.status}`;
        try {
            const errorBody = await response.json();
            if (errorBody && errorBody.message) {
                errorMessage = errorBody.message;
            } else if (typeof errorBody === 'string') {
                errorMessage = errorBody;
            }
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
        } catch (e) {
            // ignore if error body is not JSON or empty
        }
        const error = new Error(errorMessage);
        (error as any).status = response.status;
        throw error;
    }
    return response;
};

export const getAlbums = async (): Promise<Album[]> => {
    const response = await apiClient(`/albums`);
    return (await response.json()) as Album[];
};

export const getAlbumDetails = async (identifier: string): Promise<Album> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const response = await apiClient(`/albums/${encodedIdentifier}`);
    return (await response.json()) as Album;
};

export const getAlbumContents = async (identifier: string): Promise<DirectoryListing> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const response = await apiClient(`/albums/${encodedIdentifier}/contents`);
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
    return `${import.meta.env.VITE_BACKEND_URL}${thumbnailPath}`;
};

export const getBannerUrl = (bannerPath: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/${bannerPath}`;
};

export const getOriginalImageUrl = (imagePath: string): string => {
    const path = imagePath.startsWith('/') ? imagePath : `/${imagePath}`;
    return `${import.meta.env.VITE_BACKEND_URL}${path}`;
};

export const getAlbumDownloadUrl = (id: string): string => {
    return `${import.meta.env.VITE_BACKEND_URL}/albums/${id}/zip`;
};

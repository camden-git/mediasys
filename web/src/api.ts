import { Album, DirectoryListing } from './types';

export const getAlbums = async (): Promise<Album[]> => {
    const response = await fetch(`${import.meta.env.VITE_API_URL}/albums`);
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    return (await response.json()) as Album[];
};

export const getAlbumDetails = async (identifier: string): Promise<Album> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const response = await fetch(`${import.meta.env.VITE_API_URL}/albums/${encodedIdentifier}`);
    if (!response.ok) {
        if (response.status === 404) {
            throw new Error(`Album not found.`);
        }
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    return (await response.json()) as Album;
};

export const getAlbumContents = async (identifier: string): Promise<DirectoryListing> => {
    const encodedIdentifier = encodeURIComponent(identifier);
    const response = await fetch(`${import.meta.env.VITE_API_URL}/albums/${encodedIdentifier}/contents`);
    if (!response.ok) {
        if (response.status === 404) {
            throw new Error(`Album contents not found (album or folder missing).`);
        }
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    return (await response.json()) as DirectoryListing;
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

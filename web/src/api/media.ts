// URL builders for binary assets served by the backend (not JSON, so they don't go through the http client).

const backendUrl = (): string => String(import.meta.env.VITE_BACKEND_URL ?? '').replace(/\/+$/, '');

/** Encodes each segment of a slash separated path for use in a URL, dropping leading slashes. */
export const encodePath = (path: string): string =>
    path.replace(/^\/+/, '').split('/').map(encodeURIComponent).join('/');

const assetUrl = (path: string): string => `${backendUrl()}/${encodePath(path)}`;

export const getThumbnailUrl = (thumbnailPath: string): string => {
    if (/^https?:\/\//.test(thumbnailPath)) return thumbnailPath;
    return assetUrl(thumbnailPath);
};

export const getBannerUrl = (bannerPath: string): string => assetUrl(bannerPath);

export const getOriginalImageUrl = (imagePath: string): string => `${backendUrl()}/originals/${encodePath(imagePath)}`;

export const getAlbumDownloadUrl = (id: string): string => `${backendUrl()}/albums/${encodeURIComponent(id)}/zip`;

export const getPreviewImagePath = (imagePath: string): string => `/preview/${encodePath(imagePath)}`;

export const getPreviewImageUrl = (imagePath: string): string => `${backendUrl()}${getPreviewImagePath(imagePath)}`;

export const getFaceThumbnailUrl = (faceId: number): string =>
    `${backendUrl()}/faces/${encodeURIComponent(String(faceId))}/thumbnail.jpg`;

/** Returns the URL for a person's key photo thumbnail; the face id busts caches when the key photo changes. */
export const getPersonKeyPhotoUrl = (personId: number, keyPhotoFaceId: number): string =>
    `${backendUrl()}/people/${encodeURIComponent(String(personId))}/key-photo.jpg?v=${keyPhotoFaceId}`;

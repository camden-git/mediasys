import { useAlbumContextStore, useAlbumId, useAlbumName, useAlbumSlug, useAlbumData } from './useAlbumContextStore';
import { AdminAlbumResponse } from '../api/admin/albums';

export const useAlbumContext = () => {
    const album = useAlbumContextStore((s) => s.data);
    const setAlbum = useAlbumContextStore((s) => s.setAlbum);
    const clearAlbum = useAlbumContextStore((s) => s.clearAlbum);

    return {
        album,
        setAlbum,
        clearAlbum,
        // Legacy compatibility
        isLoading: false,
        error: null as string | null,
        setIsLoading: (_: boolean) => {},
        setError: (_: string | null) => {},
    };
};

export { useAlbumId, useAlbumName, useAlbumSlug, useAlbumData };
export type { AdminAlbumResponse };

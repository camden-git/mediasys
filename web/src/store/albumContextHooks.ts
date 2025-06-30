import { useStoreState, useStoreActions } from './hooks';
import { AdminAlbumResponse } from '../api/admin/albums';

export const useAlbumContext = () => {
    const album = useStoreState((state) => state.albumContext.data);
    const isLoading = useStoreState((state) => state.albumContext.isLoading);
    const error = useStoreState((state) => state.albumContext.error);

    const setAlbum = useStoreActions((actions) => actions.albumContext.setAlbum);
    const setIsLoading = useStoreActions((actions) => actions.albumContext.setIsLoading);
    const setError = useStoreActions((actions) => actions.albumContext.setError);
    const clearAlbum = useStoreActions((actions) => actions.albumContext.clearAlbum);

    return {
        album,
        isLoading,
        error,
        setAlbum,
        setIsLoading,
        setError,
        clearAlbum,
    };
};

// convenience hooks for specific album properties - these are guaranteed to be non-null
// when used in components rendered by the router
export const useAlbumId = (): number => useStoreState((state) => state.albumContext.data!.id);
export const useAlbumName = (): string => useStoreState((state) => state.albumContext.data!.name);
export const useAlbumSlug = (): string => useStoreState((state) => state.albumContext.data!.slug);
export const useAlbumData = (): AdminAlbumResponse => useStoreState((state) => state.albumContext.data!);

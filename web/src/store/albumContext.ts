import { Action, action } from 'easy-peasy';
import { AdminAlbumResponse } from '../api/admin/albums';

export interface AlbumContextStore {
    // state
    data: AdminAlbumResponse | null;
    isLoading: boolean;
    error: string | null;

    // actions
    setAlbum: Action<AlbumContextStore, AdminAlbumResponse | null>;
    setIsLoading: Action<AlbumContextStore, boolean>;
    setError: Action<AlbumContextStore, string | null>;
    clearAlbum: Action<AlbumContextStore>;
}

const albumContextStore: AlbumContextStore = {
    data: null,
    isLoading: false,
    error: null,

    setAlbum: action((state, album) => {
        state.data = album;
        state.error = null;
    }),

    setIsLoading: action((state, isLoading) => {
        state.isLoading = isLoading;
    }),

    setError: action((state, error) => {
        state.error = error;
        state.data = null;
    }),

    clearAlbum: action((state) => {
        state.data = null;
        state.isLoading = false;
        state.error = null;
    }),
};

export default albumContextStore;

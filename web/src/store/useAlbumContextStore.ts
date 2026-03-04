import { create } from 'zustand';
import { AdminAlbumResponse } from '../api/admin/albums';

interface AlbumContextState {
    data: AdminAlbumResponse | null;
    setAlbum: (album: AdminAlbumResponse | null) => void;
    clearAlbum: () => void;
}

export const useAlbumContextStore = create<AlbumContextState>()((set) => ({
    data: null,

    setAlbum: (album) => set({ data: album }),

    clearAlbum: () => set({ data: null }),
}));

export const useAlbumId = () => useAlbumContextStore((s) => s.data!.id);
export const useAlbumName = () => useAlbumContextStore((s) => s.data!.name);
export const useAlbumSlug = () => useAlbumContextStore((s) => s.data!.slug);
export const useAlbumData = (): AdminAlbumResponse => useAlbumContextStore((s) => s.data!);

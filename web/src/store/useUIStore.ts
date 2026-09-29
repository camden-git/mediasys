import { create } from 'zustand';

export interface FlashMessage {
    key: string;
    id?: number;
    type: 'success' | 'error' | 'info' | 'warning';
    title?: string;
    message: string;
}

interface UIState {
    flashes: FlashMessage[];
    addFlash: (flash: FlashMessage) => void;
    clearFlashes: (key: string) => void;
    removeFlash: (id: number) => void;
    clearAndAddHttpError: (params: { error: Error; key: string }) => void;
}

export const useUIStore = create<UIState>()((set) => ({
    flashes: [],

    addFlash: (payload) =>
        set((state) => ({
            flashes: [...state.flashes.filter((f) => f.key !== payload.key), { ...payload, id: Date.now() }],
        })),

    clearFlashes: (key) =>
        set((state) => ({
            flashes: state.flashes.filter((f) => f.key !== key),
        })),

    removeFlash: (id) =>
        set((state) => ({
            flashes: state.flashes.filter((f) => f.id !== id),
        })),

    clearAndAddHttpError: ({ error, key }) =>
        set((state) => ({
            flashes: [
                ...state.flashes.filter((f) => f.key !== key),
                {
                    key,
                    type: 'error' as const,
                    title: 'Error',
                    message: error.message || 'An error occurred',
                    id: Date.now(),
                },
            ],
        })),
}));

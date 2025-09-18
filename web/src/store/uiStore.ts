import { Action, action } from 'easy-peasy';

export interface FlashMessage {
    key: string;
    type: 'success' | 'error' | 'info' | 'warning';
    title: string;
    message: string;
}

export interface UIStore {
    flashes: FlashMessage[];
    addFlash: Action<UIStore, FlashMessage>;
    clearFlashes: Action<UIStore, string>;
    clearAndAddHttpError: Action<UIStore, { error: Error; key: string }>;
}

const uiStore: UIStore = {
    flashes: [],

    addFlash: action((state, payload) => {
        state.flashes = state.flashes.filter((flash) => flash.key !== payload.key);

        state.flashes.push(payload);
    }),

    clearFlashes: action((state, key) => {
        state.flashes = state.flashes.filter((flash) => flash.key !== key);
    }),

    clearAndAddHttpError: action((state, { error, key }) => {
        state.flashes = state.flashes.filter((flash) => flash.key !== key);

        const errorsArr = (error as any)?.errors as Array<{ detail?: string }> | undefined;
        const detail = Array.isArray(errorsArr) && errorsArr[0]?.detail ? errorsArr[0].detail : undefined;

        state.flashes.push({
            key,
            type: 'error',
            title: 'Error',
            message: detail || error.message || 'An error occurred',
        });
    }),
};

export default uiStore;

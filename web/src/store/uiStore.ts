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

        state.flashes.push({
            key,
            type: 'error',
            title: 'Error',
            message: error.message || 'An error occurred',
        });
    }),
};

export default uiStore;

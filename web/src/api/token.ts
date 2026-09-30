const STORAGE_KEY = 'authToken';

/** The single place that reads and writes the persisted auth token. */
export const getAuthToken = (): string | null => localStorage.getItem(STORAGE_KEY);

export const setAuthToken = (token: string | null): void => {
    if (token) {
        localStorage.setItem(STORAGE_KEY, token);
    } else {
        localStorage.removeItem(STORAGE_KEY);
    }
};

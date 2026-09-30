import { getAuthToken } from './token';

type UnauthorizedHandler = () => void;

let handler: UnauthorizedHandler | null = null;

export const registerUnauthorizedHandler = (cb: UnauthorizedHandler) => {
    handler = cb;
};

/**
 * Called when an API response comes back 401. Auth endpoints are excluded: a failed login is an expected 401,
 * PUT /auth/me answers 401 for a wrong current password, and GET /auth/me is handled by the auth store itself.
 */
export const handleUnauthorized = (url: string | undefined, status: number | undefined) => {
    if (status !== 401 || !getAuthToken()) return;
    if (url?.includes('/auth/')) return;
    handler?.();
};

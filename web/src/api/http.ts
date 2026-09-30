import axios, { AxiosError, AxiosInstance, AxiosResponse } from 'axios';
import { handleUnauthorized } from './unauthorized';
import { ApiError, ApiErrorDetail, messageForStatus } from './errors';
import { getAuthToken } from './token';
import type { ApiResponse } from './standard';

/**
 * The one HTTP client for the JSON API (axios: per-request timeouts, upload support and AbortSignal
 * cancellation, all behind the interceptors below). Every request gets the auth header and counts
 * towards the progress bar, and every failure rejects with an ApiError.
 */

type ProgressCbs = { onStart: () => void; onComplete: () => void };
let _progressCbs: ProgressCbs | null = null;
let _inFlight = 0;
export const registerProgressCallbacks = (cb: ProgressCbs) => {
    _progressCbs = cb;
};

const progressStart = () => {
    _inFlight++;
    _progressCbs?.onStart();
};

// Only complete the progress bar once every in-flight request has settled.
const progressDone = () => {
    _inFlight = Math.max(0, _inFlight - 1);
    if (_inFlight === 0) _progressCbs?.onComplete();
};

const http: AxiosInstance = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
    timeout: 20000,
    headers: {
        Accept: 'application/json',
    },
});

http.interceptors.request.use((req) => {
    progressStart();

    const token = getAuthToken();
    if (token) {
        req.headers.Authorization = `Bearer ${token}`;
    }
    // Content-Type is left to axios: JSON for object bodies, multipart (with boundary) for FormData,
    // and nothing at all for bodiless requests.
    return req;
});

/** Parses the standard `{errors: [{code, status, detail}]}` body, tolerating JSON delivered as a string. */
const parseErrorDetails = (body: unknown): ApiErrorDetail[] => {
    let data = body;
    if (typeof data === 'string') {
        try {
            data = JSON.parse(data);
        } catch {
            return []; // e.g. an HTML page from a proxy
        }
    }
    const errors = (data as { errors?: unknown } | null | undefined)?.errors;
    return Array.isArray(errors) ? (errors as ApiErrorDetail[]).filter((e) => typeof e?.detail === 'string') : [];
};

// Endpoint-specific wording for the auth forms when the server gave no detail.
const authFallbackMessage = (url: string | undefined, status: number | undefined): string | undefined => {
    const lowerUrl = (url ?? '').toLowerCase();
    if (lowerUrl.includes('/auth/login') && status === 401) {
        return 'No account matching those credentials could be found.';
    }
    if (lowerUrl.includes('/auth/register') && (status === 400 || status === 403)) {
        return 'Registration failed. Please verify your input and invite code.';
    }
    return undefined;
};

const toApiError = (error: unknown): ApiError => {
    if (error instanceof ApiError) return error;

    if (axios.isCancel(error)) {
        return new ApiError('Request cancelled', { isAbort: true, cause: error });
    }

    const axiosError = error as AxiosError;
    const status = axiosError.response?.status;
    const errors = parseErrorDetails(axiosError.response?.data);

    let message: string | undefined = errors[0]?.detail;
    if (!message) message = authFallbackMessage(axiosError.config?.url, status);
    if (!message) {
        message = axiosError.code === 'ECONNABORTED' ? messageForStatus(408) : messageForStatus(status);
    }

    return new ApiError(message, { status, errors, cause: error });
};

http.interceptors.response.use(
    (resp: AxiosResponse) => {
        progressDone();
        return resp;
    },
    (error: unknown) => {
        progressDone();
        const apiError = toApiError(error);
        handleUnauthorized((error as AxiosError).config?.url, apiError.status);
        throw apiError;
    },
);

/** Unwraps the `{data: ...}` envelope of a successful response. */
export const unwrap = <T>(resp: AxiosResponse<ApiResponse<T>>): T => resp.data.data;

export default http;

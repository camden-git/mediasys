import axios from 'axios';

export interface ApiErrorDetail {
    code: string;
    status: string;
    detail: string;
}

interface ApiErrorInit {
    status?: number;
    errors?: ApiErrorDetail[];
    isAbort?: boolean;
    cause?: unknown;
}

/**
 * Every failed request rejects with an ApiError. `errors` holds the parsed `{errors: [...]}` body
 * (empty for non-JSON responses such as a proxy's HTML 413/502 page) and `isAbort` is true when the
 * request was cancelled, so callers and retry loops can tell cancellation from failure.
 */
export class ApiError extends Error {
    readonly status: number | undefined;
    readonly errors: ApiErrorDetail[];
    readonly isAbort: boolean;
    readonly cause: unknown;

    constructor(message: string, { status, errors = [], isAbort = false, cause }: ApiErrorInit = {}) {
        super(message);
        this.cause = cause;
        // `name` matches the DOM convention so `err.name === 'AbortError'` checks keep working too
        this.name = isAbort ? 'AbortError' : 'ApiError';
        this.status = status;
        this.errors = errors;
        this.isAbort = isAbort;
    }

    /** True for a response with a 4xx status (a retry will not help). */
    get isClientError(): boolean {
        return this.status !== undefined && this.status >= 400 && this.status < 500;
    }
}

export const isApiError = (error: unknown): error is ApiError => error instanceof ApiError;

/** True for a cancelled request, whether it surfaced as an ApiError, an axios cancel or a DOM AbortError. */
export const isAbortError = (error: unknown): boolean => {
    if (error instanceof ApiError) return error.isAbort;
    if (axios.isCancel(error)) return true;
    return error instanceof Error && error.name === 'AbortError';
};

/** The message of a caught value, or the fallback when it isn't an Error. */
export const errorMessage = (error: unknown, fallback: string): string =>
    error instanceof Error && error.message ? error.message : fallback;

/** The HTTP status of a failed request, if it came from the server. */
export const errorStatus = (error: unknown): number | undefined => (isApiError(error) ? error.status : undefined);

/** Human-readable message for responses whose body isn't the standard `{errors: [...]}` shape. */
export const messageForStatus = (status: number | undefined): string => {
    switch (status) {
        case undefined:
            return 'Unable to reach the server. Check your connection and try again.';
        case 400:
            return 'The request was invalid.';
        case 401:
            return 'You need to sign in to do that.';
        case 403:
            return 'You do not have permission to do that.';
        case 404:
            return 'The requested resource was not found.';
        case 408:
            return 'The request timed out.';
        case 413:
            return 'The upload is too large.';
        case 429:
            return 'Too many requests. Please wait a moment and try again.';
        case 502:
        case 503:
        case 504:
            return 'The server is temporarily unavailable. Please try again shortly.';
        default:
            return status >= 500 ? 'The server ran into a problem. Please try again.' : `Request failed (${status}).`;
    }
};

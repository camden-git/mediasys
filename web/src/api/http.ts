import axios, { AxiosInstance, AxiosResponse } from 'axios';
import store from '../store';
import { ApiErrorDetail, ApiErrorResponse, httpErrorToHuman } from './standard';

const getAuthToken = (): string | null => localStorage.getItem('authToken');

const http: AxiosInstance = axios.create({
    timeout: 20000,
    headers: {
        Accept: 'application/json',
    },
});

http.interceptors.request.use((req) => {
    if (!req.url?.endsWith('/resources')) {
        store.getActions().progress.startContinuous();
    }

    // Add auth token if available
    const token = getAuthToken();
    if (token) {
        req.headers.Authorization = `Bearer ${token}`;
    }

    // Construct the full URL like the original fetch implementation
    if (req.url && !req.url.startsWith('http')) {
        req.url = `${import.meta.env.VITE_API_URL}${req.url}`;
    }

    // Ensure multipart form-data requests are not forced to JSON
    if (req.data instanceof FormData) {
        // Let the browser set the proper multipart boundary
        if (req.headers && 'Content-Type' in req.headers) {
            delete (req.headers as any)['Content-Type'];
        }
    } else if (!req.headers['Content-Type']) {
        req.headers['Content-Type'] = 'application/json';
    }

    return req;
});

http.interceptors.response.use(
    (resp: AxiosResponse) => {
        if (!resp.request?.url?.endsWith('/resources')) {
            store.getActions().progress.setComplete();
        }

        return resp;
    },
    (error) => {
        store.getActions().progress.setComplete();

        const parsePayload = (payload: unknown): ApiErrorResponse | undefined => {
            if (!payload) {
                return undefined;
            }

            if (typeof payload === 'string') {
                try {
                    return JSON.parse(payload) as ApiErrorResponse;
                } catch {
                    return { detail: payload };
                }
            }

            if (typeof payload === 'object') {
                return payload as ApiErrorResponse;
            }

            return undefined;
        };

        const parsedResponsePayload = parsePayload(error.response?.data);
        const parsedRequestPayload =
            !parsedResponsePayload && typeof error?.request?.responseText === 'string'
                ? parsePayload(error.request.responseText)
                : undefined;

        let standardizedErrors: ApiErrorDetail[] | null = null;
        const candidatePayload = parsedResponsePayload || parsedRequestPayload;
        if (candidatePayload?.errors && Array.isArray(candidatePayload.errors) && candidatePayload.errors.length > 0) {
            standardizedErrors = candidatePayload.errors as ApiErrorDetail[];
        }

        const normalizedError = {
            ...error,
            response: error.response
                ? {
                      ...error.response,
                      data: parsedResponsePayload ?? error.response.data,
                  }
                : {
                      data: candidatePayload,
                  },
        };

        let errorMessage = httpErrorToHuman(normalizedError);

        // Endpoint-aware fallbacks if extraction failed
        const status = error.response?.status as number | undefined;
        const urlStr: string | undefined = error?.config?.url;
        const lowerUrl = (urlStr || '').toLowerCase();
        if ((!errorMessage || errorMessage.startsWith('HTTP error')) && status) {
            if (lowerUrl.includes('/auth/login') && status === 401) {
                errorMessage = 'No account matching those credentials could be found.';
            }
            if (lowerUrl.includes('/auth/register') && (status === 400 || status === 403)) {
                errorMessage = 'Registration failed. Please verify your input and invite code.';
            }
        }

        if (!errorMessage) {
            errorMessage = `HTTP error! status: ${status || 'unknown'}`;
        }

        const customError = new Error(errorMessage);
        (customError as any).status = status;
        if (standardizedErrors) {
            (customError as any).errors = standardizedErrors;
        }
        throw customError;
    },
);

export default http;

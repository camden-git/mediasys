import axios, { AxiosInstance, AxiosResponse } from 'axios';
import store from '../store';

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

        // Handle error response
        let errorMessage = `HTTP error! status: ${error.response?.status || 'unknown'}`;
        let standardizedErrors: Array<{ code: string; status: string; detail: string }> | null = null;

        const tryExtractFrom = (payload: any) => {
            if (!payload) return;
            if (payload.errors && Array.isArray(payload.errors)) {
                standardizedErrors = payload.errors;
                if (standardizedErrors && standardizedErrors.length > 0) {
                    const first = standardizedErrors[0];
                    if (first?.detail) {
                        errorMessage = first.detail;
                    }
                }
                return;
            }
            if (payload.detail && typeof payload.detail === 'string') {
                errorMessage = payload.detail;
                return;
            }
            if (payload.message && typeof payload.message === 'string') {
                errorMessage = payload.message;
                return;
            }
        };

        if (error.response?.data) {
            const data = error.response.data;
            if (typeof data === 'string') {
                // Try parsing string JSON
                try {
                    const parsed = JSON.parse(data);
                    tryExtractFrom(parsed);
                    if (!standardizedErrors && typeof parsed === 'string') {
                        errorMessage = parsed;
                    }
                } catch {
                    errorMessage = data;
                }
            } else if (typeof data === 'object') {
                tryExtractFrom(data);
            }
        }

        // Some environments expose the raw response as text on request
        if (!standardizedErrors && typeof (error?.request?.responseText) === 'string') {
            try {
                const parsed = JSON.parse(error.request.responseText);
                tryExtractFrom(parsed);
                if (!standardizedErrors && typeof parsed === 'string') {
                    errorMessage = parsed;
                }
            } catch {
                // ignore
            }
        }

        // Endpoint-aware fallbacks if extraction failed
        const status = error.response?.status as number | undefined;
        const urlStr: string | undefined = error?.config?.url;
        const lowerUrl = (urlStr || '').toLowerCase();
        if (errorMessage.startsWith('HTTP error!') && status) {
            if (lowerUrl.includes('/auth/login') && status === 401) {
                errorMessage = 'No account matching those credentials could be found.';
            }
            if (lowerUrl.includes('/auth/register') && (status === 400 || status === 403)) {
                // Keep generic but user-friendly register error if backend detail was not parsed
                errorMessage = 'Registration failed. Please verify your input and invite code.';
            }
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

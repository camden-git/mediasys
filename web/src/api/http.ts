import axios, { AxiosInstance, AxiosResponse } from 'axios';
import { handleUnauthorized } from './unauthorized';
import { ApiErrorDetail, httpErrorToHuman } from './standard';

const getAuthToken = (): string | null => localStorage.getItem('authToken');

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
    timeout: 20000,
    headers: {
        Accept: 'application/json',
    },
});

http.interceptors.request.use((req) => {
    progressStart();

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
        progressDone();

        return resp;
    },
    (error) => {
        progressDone();

        // The backend always returns errors as {"errors": [{code, status, detail}]}, so if the
        // response body came back as a JSON string for some reason, parse it before extracting.
        let responseData = error.response?.data;
        if (typeof responseData === 'string') {
            try {
                responseData = JSON.parse(responseData);
            } catch {
                responseData = undefined;
            }
        }

        const standardizedErrors: ApiErrorDetail[] | null =
            responseData?.errors && Array.isArray(responseData.errors) && responseData.errors.length > 0
                ? (responseData.errors as ApiErrorDetail[])
                : null;

        const normalizedError = {
            ...error,
            response: error.response ? { ...error.response, data: responseData } : undefined,
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

        handleUnauthorized(urlStr, status);

        const customError = new Error(errorMessage);
        (customError as any).status = status;
        if (standardizedErrors) {
            (customError as any).errors = standardizedErrors;
        }
        throw customError;
    },
);

export default http;

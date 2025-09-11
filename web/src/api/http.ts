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

        if (error.response?.data) {
            if (error.response.data.message) {
                errorMessage = error.response.data.message;
            } else if (typeof error.response.data === 'string') {
                errorMessage = error.response.data;
            }
        }

        const customError = new Error(errorMessage);
        (customError as any).status = error.response?.status;
        throw customError;
    },
);

export default http;

import axios from 'axios';
import { describe, expect, it } from 'vitest';
import { ApiError, errorMessage, errorStatus, isAbortError, isApiError, messageForStatus } from './errors';

describe('ApiError', () => {
    it('defaults to a non-abort error with no details', () => {
        const err = new ApiError('failed');
        expect(err.name).toBe('ApiError');
        expect(err.status).toBeUndefined();
        expect(err.errors).toEqual([]);
        expect(err.isAbort).toBe(false);
        expect(err.isClientError).toBe(false);
    });

    it('is named AbortError for cancelled requests', () => {
        const err = new ApiError('cancelled', { isAbort: true });
        expect(err.name).toBe('AbortError');
        expect(err.isAbort).toBe(true);
    });

    it('treats only 4xx statuses as client errors', () => {
        expect(new ApiError('x', { status: 399 }).isClientError).toBe(false);
        expect(new ApiError('x', { status: 400 }).isClientError).toBe(true);
        expect(new ApiError('x', { status: 499 }).isClientError).toBe(true);
        expect(new ApiError('x', { status: 500 }).isClientError).toBe(false);
    });
});

describe('error helpers', () => {
    it('isApiError narrows on ApiError instances only', () => {
        expect(isApiError(new ApiError('x'))).toBe(true);
        expect(isApiError(new Error('x'))).toBe(false);
        expect(isApiError('x')).toBe(false);
    });

    it('isAbortError recognises ApiError aborts, axios cancels and DOM AbortErrors', () => {
        expect(isAbortError(new ApiError('x', { isAbort: true }))).toBe(true);
        expect(isAbortError(new ApiError('x', { status: 500 }))).toBe(false);
        expect(isAbortError(new axios.CanceledError('cancelled'))).toBe(true);
        const dom = new Error('aborted');
        dom.name = 'AbortError';
        expect(isAbortError(dom)).toBe(true);
        expect(isAbortError(new Error('plain'))).toBe(false);
        expect(isAbortError(null)).toBe(false);
    });

    it('errorMessage uses the error message or the fallback', () => {
        expect(errorMessage(new Error('real'), 'fallback')).toBe('real');
        expect(errorMessage(new Error(''), 'fallback')).toBe('fallback');
        expect(errorMessage('string', 'fallback')).toBe('fallback');
    });

    it('errorStatus only reads the status from ApiErrors', () => {
        expect(errorStatus(new ApiError('x', { status: 404 }))).toBe(404);
        expect(errorStatus(new Error('x'))).toBeUndefined();
    });
});

describe('messageForStatus', () => {
    it('explains an unreachable server when there is no status', () => {
        expect(messageForStatus(undefined)).toMatch(/unable to reach the server/i);
    });

    it('has specific messages for common statuses', () => {
        expect(messageForStatus(401)).toMatch(/sign in/i);
        expect(messageForStatus(403)).toMatch(/permission/i);
        expect(messageForStatus(404)).toMatch(/not found/i);
        expect(messageForStatus(413)).toMatch(/too large/i);
        expect(messageForStatus(429)).toMatch(/too many requests/i);
        expect(messageForStatus(503)).toMatch(/temporarily unavailable/i);
    });

    it('falls back by status class', () => {
        expect(messageForStatus(418)).toBe('Request failed (418).');
        expect(messageForStatus(500)).toMatch(/ran into a problem/i);
    });
});

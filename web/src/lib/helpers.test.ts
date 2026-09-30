import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/errors';
import { randomInt, safeRedirectPath, withRateLimitMessage } from './helpers';

describe('randomInt', () => {
    it('stays within the inclusive bounds', () => {
        for (let i = 0; i < 200; i++) {
            const n = randomInt(3, 5);
            expect(Number.isInteger(n)).toBe(true);
            expect(n).toBeGreaterThanOrEqual(3);
            expect(n).toBeLessThanOrEqual(5);
        }
    });

    it('returns the only possible value when min equals max', () => {
        expect(randomInt(7, 7)).toBe(7);
    });
});

describe('safeRedirectPath', () => {
    it('keeps same-origin paths with search and hash', () => {
        expect(safeRedirectPath({ pathname: '/admin/albums', search: '?a=1', hash: '#x' }, '/')).toBe(
            '/admin/albums?a=1#x',
        );
        expect(safeRedirectPath({ pathname: '/album/foo' }, '/')).toBe('/album/foo');
    });

    it('falls back for non-objects and missing pathnames', () => {
        expect(safeRedirectPath(undefined, '/admin')).toBe('/admin');
        expect(safeRedirectPath('/evil', '/admin')).toBe('/admin');
        expect(safeRedirectPath({}, '/admin')).toBe('/admin');
    });

    it('rejects absolute and protocol-relative URLs (open redirects)', () => {
        expect(safeRedirectPath({ pathname: 'https://evil.example' }, '/admin')).toBe('/admin');
        expect(safeRedirectPath({ pathname: '//evil.example' }, '/admin')).toBe('/admin');
        expect(safeRedirectPath({ pathname: '/\\evil.example' }, '/admin')).toBe('/admin');
    });

    it('never redirects back into the auth pages', () => {
        expect(safeRedirectPath({ pathname: '/auth/login' }, '/admin')).toBe('/admin');
    });

    it('ignores non-string search and hash values', () => {
        expect(safeRedirectPath({ pathname: '/x', search: 5, hash: null }, '/')).toBe('/x');
    });
});

describe('withRateLimitMessage', () => {
    it('swaps the message for a friendly one on HTTP 429', () => {
        const err = withRateLimitMessage(new ApiError('raw', { status: 429 }));
        expect(err.message).toMatch(/too many attempts/i);
    });

    it('passes other errors through unchanged', () => {
        const original = new ApiError('nope', { status: 500 });
        expect(withRateLimitMessage(original)).toBe(original);
    });

    it('wraps non-Error values in an Error', () => {
        const err = withRateLimitMessage('boom');
        expect(err).toBeInstanceOf(Error);
        expect(err.message).toBe('boom');
    });
});

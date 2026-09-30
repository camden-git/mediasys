/**
 * Generates a random integer between min and max (inclusive)
 */
export function randomInt(min: number, max: number): number {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Returns a same-origin relative path from a router `from` location, or the fallback.
 * Rejects absolute / protocol-relative URLs to avoid open redirects.
 */
export function safeRedirectPath(from: unknown, fallback: string): string {
    if (!from || typeof from !== 'object') return fallback;
    const { pathname, search, hash } = from as { pathname?: unknown; search?: unknown; hash?: unknown };
    if (typeof pathname !== 'string' || !/^\/(?![/\\])/.test(pathname) || pathname.startsWith('/auth/')) {
        return fallback;
    }
    return `${pathname}${typeof search === 'string' ? search : ''}${typeof hash === 'string' ? hash : ''}`;
}

/**
 * Swaps the message for a friendly one when the request was rate limited (HTTP 429).
 */
export function withRateLimitMessage(err: unknown): Error {
    const error = err instanceof Error ? err : new Error(String(err));
    if ((error as Error & { status?: number }).status === 429) {
        return new Error('Too many attempts. Please wait a few minutes and try again.');
    }
    return error;
}

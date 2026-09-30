import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { bytesToString, formatTimeRemaining, getTimeRemaining, getUrgencyLevel } from './formatters';

describe('bytesToString', () => {
    it('returns "0 Bytes" for zero, negative and non-finite input', () => {
        expect(bytesToString(0)).toBe('0 Bytes');
        expect(bytesToString(-5)).toBe('0 Bytes');
        expect(bytesToString(Number.NaN)).toBe('0 Bytes');
        expect(bytesToString(Number.POSITIVE_INFINITY)).toBe('0 Bytes');
    });

    it('uses 1024-based units', () => {
        expect(bytesToString(512)).toBe('512 Bytes');
        expect(bytesToString(1024)).toBe('1 KiB');
        expect(bytesToString(1536)).toBe('1.5 KiB');
        expect(bytesToString(1024 * 1024)).toBe('1 MiB');
        expect(bytesToString(5 * 1024 ** 3)).toBe('5 GiB');
    });

    it('respects the decimals argument', () => {
        expect(bytesToString(1234567, 0)).toBe('1 MiB');
        expect(bytesToString(1234567, 3)).toBe('1.177 MiB');
    });
});

describe('time remaining helpers', () => {
    const now = new Date('2026-01-01T00:00:00Z');

    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(now);
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    const inFuture = (ms: number) => new Date(now.getTime() + ms).toISOString();

    it('reports an expired date as expired with zeroed fields', () => {
        const remaining = getTimeRemaining(inFuture(-1000));
        expect(remaining.isExpired).toBe(true);
        expect(remaining.total).toBe(0);
        expect(formatTimeRemaining(remaining)).toBe('Expired');
    });

    it('splits the remaining time into days, hours, minutes and seconds', () => {
        const ms = ((2 * 24 + 3) * 60 * 60 + 4 * 60 + 5) * 1000;
        expect(getTimeRemaining(inFuture(ms))).toMatchObject({
            days: 2,
            hours: 3,
            minutes: 4,
            seconds: 5,
            isExpired: false,
        });
    });

    it('formats using only the largest relevant units', () => {
        const fmt = (ms: number) => formatTimeRemaining(getTimeRemaining(inFuture(ms)));
        expect(fmt(((2 * 24 + 3) * 60 * 60 + 4 * 60 + 5) * 1000)).toBe('2d 3h 4m');
        expect(fmt((3 * 60 * 60 + 4 * 60 + 5) * 1000)).toBe('3h 4m 5s');
        expect(fmt((4 * 60 + 5) * 1000)).toBe('4m 5s');
        expect(fmt(5000)).toBe('5s');
    });

    it('classifies urgency', () => {
        const hour = 60 * 60 * 1000;
        expect(getUrgencyLevel(inFuture(-1))).toBe('critical');
        expect(getUrgencyLevel(inFuture(30 * 60 * 1000))).toBe('critical');
        expect(getUrgencyLevel(inFuture(5 * hour))).toBe('warning');
        expect(getUrgencyLevel(inFuture(48 * hour))).toBe('normal');
    });
});

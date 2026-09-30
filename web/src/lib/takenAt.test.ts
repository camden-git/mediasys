import { describe, expect, it } from 'vitest';
import { formatTakenAt, formatTakenAtDate } from './takenAt';

describe('takenAt formatting', () => {
    // 2024-03-05 14:30:00 UTC
    const ts = Date.UTC(2024, 2, 5, 14, 30) / 1000;

    it('returns null for missing or zero timestamps', () => {
        expect(formatTakenAt(undefined)).toBeNull();
        expect(formatTakenAt(0)).toBeNull();
        expect(formatTakenAtDate(undefined)).toBeNull();
        expect(formatTakenAtDate(0)).toBeNull();
    });

    it('returns null for timestamps outside the representable date range', () => {
        expect(formatTakenAt(Number.MAX_VALUE)).toBeNull();
        expect(formatTakenAtDate(Number.MAX_VALUE)).toBeNull();
    });

    it('formats in UTC regardless of the local timezone', () => {
        expect(formatTakenAt(ts)).toContain('March 5, 2024');
        expect(formatTakenAt(ts)).toMatch(/2:30\sPM/);
        expect(formatTakenAtDate(ts)).toBe('Mar 5, 2024');
    });

    it('keeps the wall-clock date at a UTC day boundary', () => {
        const lateNight = Date.UTC(2024, 11, 31, 23, 59) / 1000;
        expect(formatTakenAtDate(lateNight)).toBe('Dec 31, 2024');
    });
});

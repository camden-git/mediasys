import { describe, expect, it } from 'vitest';
import { computeLayout, LayoutOptions } from './galleryLayout';
import { FileInfo } from '../types';

const img = (name: string, width?: number, height?: number): FileInfo => ({
    name,
    path: `/${name}`,
    is_dir: false,
    size: 1,
    mod_time: 0,
    width,
    height,
});

const options: LayoutOptions = { containerWidth: 1000, targetRowHeight: 200, boxSpacing: 10 };

describe('computeLayout', () => {
    it('returns no rows for empty input or an unusable container width', () => {
        expect(computeLayout([], options)).toEqual([]);
        expect(computeLayout([img('a', 100, 100)], { ...options, containerWidth: 0 })).toEqual([]);
        expect(computeLayout([img('a', 100, 100)], { ...options, containerWidth: -10 })).toEqual([]);
    });

    it('places every image in exactly one row, preserving order', () => {
        const images = Array.from({ length: 25 }, (_, i) => img(`img-${i}`, 1500 + (i % 3) * 500, 1000));
        const rows = computeLayout(images, options);
        expect(rows.flatMap((r) => r.items.map((i) => i.name))).toEqual(images.map((i) => i.name));
        expect(rows.map((r) => r.rowIndex)).toEqual(rows.map((_, i) => i));
    });

    it('fills full rows to the container width', () => {
        const images = Array.from({ length: 12 }, (_, i) => img(`img-${i}`, 1500, 1000));
        const rows = computeLayout(images, options);
        expect(rows.length).toBeGreaterThan(1);
        // every row except the last is justified: item widths + spacing == containerWidth
        for (const row of rows.slice(0, -1)) {
            const widths = row.items.reduce((sum, i) => sum + (i.width! / i.height!) * row.height, 0);
            const total = widths + (row.items.length - 1) * options.boxSpacing;
            expect(total).toBeCloseTo(options.containerWidth, 3);
        }
    });

    it('does not stretch a short last row past the target height by default', () => {
        const rows = computeLayout([img('a', 1000, 1000)], options);
        expect(rows).toHaveLength(1);
        expect(rows[0].height).toBe(options.targetRowHeight);
    });

    it('stretches the last row when requested (up to the max height ratio)', () => {
        const rows = computeLayout([img('a', 4000, 1000)], { ...options, stretchLastRow: true });
        expect(rows).toHaveLength(1);
        expect(rows[0].height).toBeGreaterThan(options.targetRowHeight);
        expect(rows[0].height).toBeLessThanOrEqual(options.targetRowHeight * 2.5);
    });

    it('treats images without dimensions as 3:2', () => {
        const [row] = computeLayout([img('a')], { ...options, containerWidth: 300, targetRowHeight: 400 });
        // a lone image is capped at the target height; 300 / 1.5 = 200 fits under it
        expect(row.height).toBeCloseTo(200, 3);
    });

    it('never produces a row height below 1px', () => {
        const rows = computeLayout([img('wide', 1_000_000, 1)], options);
        expect(rows[0].height).toBeGreaterThanOrEqual(1);
    });
});

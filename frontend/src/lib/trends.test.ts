import {describe, expect, it} from 'vitest';
import {computeTrendPaths} from './trends';

describe('computeTrendPaths', () => {
    it('不足两个点时返回空 path', () => {
        const r = computeTrendPaths([{date: 'a', footprint: 1, cleanable: 0, user: 1}]);
        expect(r.footprint).toBe('');
        expect(r.cleanable).toBe('');
        expect(r.max).toBe(0);
    });

    it('两个点生成 M 开头、L 连接的折线 path', () => {
        const r = computeTrendPaths([
            {date: '2026-08-10', footprint: 100, cleanable: 10, user: 90},
            {date: '2026-08-11', footprint: 200, cleanable: 20, user: 180},
        ]);
        expect(r.footprint.startsWith('M')).toBe(true);
        expect(r.footprint).toContain('L');
        expect(r.cleanable).not.toBe('');
        expect(r.max).toBe(200);
    });

    it('最高点映射到图表顶部（y 更小）', () => {
        const r = computeTrendPaths([
            {date: 'a', footprint: 100, cleanable: 0, user: 100},
            {date: 'b', footprint: 200, cleanable: 0, user: 200},
        ]);
        const [, second] = r.footprint.split('L');
        const y2 = Number(second.split(',')[1]);
        // 图表高度 240、边距 34：最高点 y ≈ 34，最低点 y ≈ 206
        expect(y2).toBeLessThan(50);
    });

    it('x 轴标签仅落在采样点与最后一个点', () => {
        const points = Array.from({length: 13}, (_, i) => ({date: `2026-08-${String(i + 1).padStart(2, '0')}`, footprint: 100 + i, cleanable: 0, user: 100 + i}));
        const r = computeTrendPaths(points);
        const labels = (r.labels.match(/<text/g) ?? []).length;
        // 13 个点，step=2，采样 0/2/4/6/8/10 + 末尾 12 → 7 个标签
        expect(labels).toBe(7);
    });
});

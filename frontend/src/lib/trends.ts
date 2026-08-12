import type { Point } from './types';

// 计算趋势折线（总占用/可清理）的 SVG path、x 轴标签与最大值（纯函数，可单测）。
// footprint 为空字符串表示点数不足，调用方应提示"数据积累中"。
export function computeTrendPaths(
    points: Point[],
    W = 760, H = 240, PAD = 34,
): { footprint: string; cleanable: string; labels: string; max: number } {
    if (points.length < 2) {
        return { footprint: '', cleanable: '', labels: '', max: 0 };
    }
    const max = Math.max(1, ...points.map(p => p.footprint));
    const xs = (i: number) => PAD + i * (W - 2 * PAD) / (points.length - 1);
    const ys = (v: number) => H - PAD - (v / max) * (H - 2 * PAD);
    const path = (pick: (p: Point) => number) =>
        points.map((p, i) => `${i ? 'L' : 'M'}${xs(i).toFixed(1)},${ys(pick(p)).toFixed(1)}`).join(' ');
    const step = Math.max(1, Math.floor(points.length / 6));
    const labels = points.map((p, i) => (i % step === 0 || i === points.length - 1)
        ? `<text x="${xs(i)}" y="${H - PAD + 16}" text-anchor="middle" font-size="10" fill="var(--muted)">${p.date.slice(5)}</text>` : '').join('');
    return { footprint: path(p => p.footprint), cleanable: path(p => p.cleanable), labels, max };
}

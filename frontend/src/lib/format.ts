// 字节与时间的展示格式化（纯函数，可单测）
export function hb(n: number): string {
    if (n < 1024) return `${n} B`;
    // 后缀表与 Go 端 humanBytes 一致（K/M/G/T/P/E），覆盖 int64 全范围
    const units = ['K', 'M', 'G', 'T', 'P', 'E'];
    let v = n;
    let i = -1;
    do { v /= 1024; i++; } while (v >= 1024 && i < units.length - 1);
    return `${v.toFixed(1)} ${units[i]}B`;
}

// 将 RFC3339 时间压缩为 "YYYY-MM-DD HH:MM"（与 CLI 输出一致）
export function fmtTime(ts: string): string {
    return ts.slice(0, 16).replace('T', ' ');
}

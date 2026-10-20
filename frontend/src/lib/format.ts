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

// shortPath 把用户主目录前缀换成 `~`，缩短长路径的展示宽度。
// 列表里 `/Users/wei/Library/Application Support/JoyCoder` 会被截断成
// `/Users/wei/Library/Application Supp...`，有效信息全丢了；`~/Library/...`
// 能多显示一截。
//
// 只替换**主目录本身或它的子路径**：`/Users/wei2/...` 不能被误当成 `/Users/wei`。
// 分隔符兼容 Windows 反斜杠，且**保留原分隔符**（展示的是真实路径形状，
// 不应把 `~\AppData\Roaming` 改写成 `~/AppData/Roaming`）。
// home 为空时原样返回（不猜测）。
// 注意：这只是展示，不能把结果拿去当路径用（后端需要绝对路径）。
export function shortPath(p: string, home: string): string {
    if (!p || !home) return p;
    // 去掉末尾分隔符再比，避免 `/Users/wei/` 与 `/Users/wei` 判不相等。
    const trim = (s: string) => s.replace(/[\\/]+$/, '');
    const tp = trim(p);
    const th = trim(home);
    if (!th) return p;
    if (tp === th) return '~';
    // 前缀必须紧跟分隔符，否则 `/Users/wei2` 会被当成 `/Users/wei` 的子路径。
    const sep = tp.charAt(th.length);
    if (tp.startsWith(th) && (sep === '/' || sep === '\\')) {
        return '~' + tp.slice(th.length);
    }
    return p;
}

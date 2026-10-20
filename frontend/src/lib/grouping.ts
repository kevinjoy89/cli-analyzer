// 工具列表的 AI 优先分组纯逻辑（add-ai-focused-grouping）。
// category 非空（agent / ai-ide）归 AI 段，其余（通用 CLI）归「其他」段；
// 过滤按名称与别名，两段独立应用。DOM 无关，供 render 与 vitest 共用。
import type { Tool } from './types';

// 搜索过滤：工具名或任一别名包含关键词（大小写不敏感，函数内归一，
// 调用方无需预处理）。
export function matchesFilter(t: Tool, q: string): boolean {
    const ql = q.trim().toLowerCase();
    if (!ql) return true;
    return (
        t.name.toLowerCase().includes(ql) ||
        (t.displayName ? t.displayName.toLowerCase().includes(ql) : false) ||
        t.aliases.some((a) => a.toLowerCase().includes(ql))
    );
}

export interface ToolGroup {
    label: 'ai' | 'others';
    tools: Tool[];
    footprintBytes: number;
}

// 按当前搜索词把工具分成 AI 段与其他段，并计算两段合计占用。
// 段为空时 tools 为空数组、合计为 0（渲染方据此决定是否渲染段）。
export function groupTools(
    tools: Tool[],
    q: string,
): { ai: ToolGroup; others: ToolGroup } {
    const ai: Tool[] = [];
    const others: Tool[] = [];
    for (const t of tools) {
        if (!matchesFilter(t, q)) continue;
        // 已卸载且无占用的工具不应出现在列表中（即使在内置回收站中未清空）
        if (t.uninstalled && t.footprintBytes === 0) continue;
        if (t.category) ai.push(t);
        else others.push(t);
    }
    const sum = (list: Tool[]) =>
        list.reduce((acc, t) => acc + t.footprintBytes, 0);
    return {
        ai: { label: 'ai', tools: ai, footprintBytes: sum(ai) },
        others: {
            label: 'others',
            tools: others,
            footprintBytes: sum(others),
        },
    };
}

// pathInside 报告 p 是否严格位于 dir 之内（p ≠ dir，无越界）。
// 用于父子同选去重：子路径被所选父路径包含时随父项处置，不再重复
// 计入确认合计（与后端 cleaner 的 isInside 同口径）。分隔符无关：
// Windows 反斜杠路径归一为正斜杠后比较（大小写敏感，实际处置由
// 后端权威判定，这里只保证确认合计口径一致）。
export function pathInside(dir: string, p: string): boolean {
    if (!dir || !p) return false;
    const dn = dir.replace(/\\/g, "/");
    const pn = p.replace(/\\/g, "/");
    if (dn === pn) return false;
    return pn.startsWith(dn.endsWith("/") ? dn : dn + "/");
}

// FormSubtotals 是形态占用小计（add-form-factor-split）：桌面端（应用包 +
// 桌面端数据目录）与 CLI 各自合计。
export interface FormSubtotals {
    desktop: number;
    cli: number;
}

// formSubtotals 按极大路径口径分形态合计：父目录与子目录并存时不重复
// 计入（与后端 cleanableBytes 的 sumMaximal 同口径）。form 缺省按 CLI 计。
export function formSubtotals(
    dirs: Array<{ path: string; bytes: number; form?: string }>,
): FormSubtotals {
    const maximal = keepMaximalPaths(dirs);
    let desktop = 0;
    let cli = 0;
    for (const d of maximal) {
        if (d.form === 'desktop') desktop += d.bytes;
        else cli += d.bytes;
    }
    return { desktop, cli };
}

// keepMaximalPaths 只保留极大路径：某个条目的路径严格位于另一条目之内时，
// 它随外层条目一起处置，不再单独计入合计。这是「可处置合计」与「形态小计」
// 共用的唯一去重口径（后端 cleaner 的 isInside 同义）。
// 抽成函数而不是各处内联：CLI 与 GUI 的合计必须一致，两套手写实现会分叉。
export function keepMaximalPaths<T extends { path: string }>(items: T[]): T[] {
    return items.filter(
        (it) => !items.some((o) => o.path !== it.path && pathInside(o.path, it.path)),
    );
}

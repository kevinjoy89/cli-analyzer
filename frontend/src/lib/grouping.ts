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

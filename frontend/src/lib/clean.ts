import type { ScanResult } from './types';

// 清理成功后从扫描快照移除已清理项（含子项），重算各工具与总计。
// 原地修改并返回 result，供 UI 立即刷新，无需等待整体重扫。
export function applyCleanLocally(result: ScanResult, ids: string[]): ScanResult {
    const idSet = new Set(ids);
    for (const t of result.tools) {
        let toolFreed = 0;
        // 整项清理
        for (const c of t.cleanables) {
            if (idSet.has(c.id)) { toolFreed += c.bytes; c.bytes = 0; c.sub = []; }
        }
        // 子项清理
        for (const c of t.cleanables) {
            if (!(c.sub ?? []).length) continue;
            let subFreed = 0;
            c.sub = c.sub.filter(s => { if (idSet.has(s.id)) { subFreed += s.bytes; return false; } return true; });
            c.bytes -= subFreed;
            toolFreed += subFreed;
        }
        t.cleanables = t.cleanables.filter(c => c.bytes > 0);
        t.cleanableBytes = t.cleanables.reduce((a, c) => a + c.bytes, 0);
        t.footprintBytes = Math.max(0, t.footprintBytes - toolFreed);
        t.userBytes = Math.max(0, t.footprintBytes - t.cleanableBytes);
    }
    result.tools = result.tools.filter(t => t.cleanableBytes > 0 || t.userBytes > 0 || t.binaries.length > 0);
    result.totals.footprintBytes = result.tools.reduce((a, t) => a + t.footprintBytes, 0);
    result.totals.cleanableBytes = result.tools.reduce((a, t) => a + t.cleanableBytes, 0);
    result.totals.userBytes = result.tools.reduce((a, t) => a + t.userBytes, 0);
    return result;
}

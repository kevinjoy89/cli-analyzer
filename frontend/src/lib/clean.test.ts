import {describe, expect, it} from 'vitest';
import {applyCleanLocally} from './clean';
import type {ScanResult, Tool} from './types';

// mkTool 便捷构造一个带单个 cleanable 的工具
function mkTool(cleanable: {id: string; bytes: number; sub?: {id: string; path: string; bytes: number}[]}, userBytes = 100): Tool {
    const bytes = cleanable.bytes;
    return {
        name: 'npm', aliases: [], installer: 'npm', version: '', updatedAt: '', homepage: '', description: '',
        binaries: [], dataDirs: [],
        cleanables: [{
            id: cleanable.id, tool: 'npm', path: '/a', bytes: cleanable.bytes, tier: 'safe',
            kind: 'cache', keep: '', desc: '', sub: cleanable.sub ?? [],
        }],
        footprintBytes: bytes + userBytes, cleanableBytes: bytes, userBytes,
    };
}

function mkResult(tools: Tool[]): ScanResult {
    const totals = {footprintBytes: 0, cleanableBytes: 0, userBytes: 0};
    for (const t of tools) {
        totals.footprintBytes += t.footprintBytes;
        totals.cleanableBytes += t.cleanableBytes;
        totals.userBytes += t.userBytes;
    }
    return {scannedAt: '', scanTimeMs: 0, platform: '', goVersion: '', tools, totals, roots: {}, walkErrors: 0};
}

describe('applyCleanLocally', () => {
    it('移除整个可清理项并重算 totals', () => {
        const res = mkResult([mkTool({id: 'npm|cache|/a', bytes: 100})]);
        applyCleanLocally(res, ['npm|cache|/a']);
        expect(res.tools.length).toBe(1); // 仍有 userBytes，工具保留
        expect(res.tools[0].cleanables.length).toBe(0);
        expect(res.tools[0].cleanableBytes).toBe(0);
        expect(res.totals.cleanableBytes).toBe(0);
        expect(res.totals.footprintBytes).toBe(100); // footprint -= 清理的 100
    });

    it('移除子项并减小父级可清理量', () => {
        const res = mkResult([mkTool({
            id: 'npm|cache|/a', bytes: 300,
            sub: [
                {id: 'npm|cache|/a::/a/x', path: '/a/x', bytes: 100},
                {id: 'npm|cache|/a::/a/y', path: '/a/y', bytes: 200},
            ],
        }, 0)]);
        applyCleanLocally(res, ['npm|cache|/a::/a/x']);
        expect(res.tools[0].cleanables[0].bytes).toBe(200);
        expect(res.tools[0].cleanables[0].sub.length).toBe(1);
        expect(res.totals.cleanableBytes).toBe(200);
    });

    it('工具完全清空后整行移除', () => {
        const res = mkResult([mkTool({id: 'npm|cache|/a', bytes: 50}, 0)]);
        applyCleanLocally(res, ['npm|cache|/a']);
        expect(res.tools.length).toBe(0);
        expect(res.totals.footprintBytes).toBe(0);
        expect(res.totals.cleanableBytes).toBe(0);
    });

    it('未知 id 不影响结果', () => {
        const res = mkResult([mkTool({id: 'npm|cache|/a', bytes: 100})]);
        applyCleanLocally(res, ['does-not-exist']);
        expect(res.tools[0].cleanableBytes).toBe(100);
        expect(res.totals.cleanableBytes).toBe(100);
    });
});

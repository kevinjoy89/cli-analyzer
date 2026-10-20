import {describe, expect, it} from 'vitest';
import type {Tool} from './types';

// 双形态工具的版本展示规则（render.ts versionCell 的口径）：
// 两种形态都有版本 → 分行列出「CLI x」/「桌面端 y」；只有一侧 → 单行。
// 不展示「版本不一致」文案，也不做悬浮提示——两个值直接摆出来即可。
// 真实场景：kimi CLI 2.1.0 vs 桌面 1.0.2。

function mkTool(p: Partial<Tool>): Tool {
    return {
        name: 'x', aliases: [], installer: 'other', version: '',
        updatedAt: '', homepage: '', description: '',
        binaries: [], dataDirs: [], cleanables: [],
        footprintBytes: 0, cleanableBytes: 0, userBytes: 0,
        ...p,
    } as Tool;
}

// versionLines 复刻 versionCell 的显示决策（返回行数组）。
function versionLines(t: Tool): string[] {
    const cli = t.cliVersion || '';
    const desk = t.desktopVersion || '';
    if (cli && desk) {
        return [`CLI ${cli}`, `桌面端 ${desk}`];
    }
    return [cli || desk || t.version || '—'];
}

describe('版本列：双形态分行展示', () => {
    it('CLI 与桌面端都有版本：分两行，各带来源前缀', () => {
        const t = mkTool({version: '2.1.0', cliVersion: '2.1.0', desktopVersion: '1.0.2'});
        expect(versionLines(t)).toEqual(['CLI 2.1.0', '桌面端 1.0.2']);
    });

    it('两版本相同时也分行（不隐藏来源）', () => {
        const t = mkTool({version: '1.0.2', cliVersion: '1.0.2', desktopVersion: '1.0.2'});
        expect(versionLines(t)).toEqual(['CLI 1.0.2', '桌面端 1.0.2']);
    });

    it('只有桌面端（无 CLI 二进制）：单行，不带前缀', () => {
        const t = mkTool({version: '5.7.7', desktopVersion: '5.7.7'});
        expect(versionLines(t)).toEqual(['5.7.7']);
    });

    it('只有 CLI 版本：单行', () => {
        const t = mkTool({version: '0.1.15', cliVersion: '0.1.15'});
        expect(versionLines(t)).toEqual(['0.1.15']);
    });

    it('完全未知：显示破折号', () => {
        expect(versionLines(mkTool({}))).toEqual(['—']);
    });
});

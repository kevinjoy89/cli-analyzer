import { describe, expect, it } from 'vitest';
import { formSubtotals, groupTools, matchesFilter, pathInside } from './grouping';
import type { Tool } from './types';

// 与 Go JSON 契约同构的最小 Tool 构造（grouping 只消费 name/aliases/
// category/footprintBytes）。
function tool(
    name: string,
    category?: string,
    footprintBytes = 0,
    aliases?: string[],
    uninstalled?: boolean,
): Tool {
    return {
        name,
        aliases: aliases ?? [name + '-alias'],
        installer: 'other',
        version: '',
        updatedAt: '',
        homepage: '',
        description: '',
        category,
        uninstalled,
        binaries: [],
        dataDirs: [],
        cleanables: [],
        footprintBytes,
        cleanableBytes: 0,
        userBytes: 0,
    };
}

describe('matchesFilter', () => {
    it('空关键词匹配一切', () => {
        expect(matchesFilter(tool('opencode', 'agent'), '')).toBe(true);
    });
    it('命中工具名或别名（大小写不敏感）', () => {
        expect(matchesFilter(tool('opencode', 'agent'), 'OPEN')).toBe(true);
        expect(matchesFilter(tool('kimi', 'agent', 0, ['kimi-code']), 'kimi-code')).toBe(true);
        expect(matchesFilter(tool('git'), 'npm')).toBe(false);
    });
    it('命中 displayName（大小写不敏感）', () => {
        const t = tool('code', 'ai-ide');
        t.displayName = 'Visual Studio Code';
        expect(matchesFilter(t, 'studio')).toBe(true);
        expect(matchesFilter(t, 'visual')).toBe(true);
    });
});

describe('groupTools', () => {
    const tools = [
        tool('nodejs', undefined, 17),
        tool('uv', undefined, 10),
        tool('opencode', 'agent', 12),
        tool('code', 'ai-ide', 2),
        tool('trae-cn', 'ai-ide', 1),
    ];

    it('category 非空归 AI 段，其余归其他段', () => {
        const { ai, others } = groupTools(tools, '');
        expect(ai.tools.map((t) => t.name)).toEqual(['opencode', 'code', 'trae-cn']);
        expect(others.tools.map((t) => t.name)).toEqual(['nodejs', 'uv']);
    });

    it('两段合计各自独立计算', () => {
        const { ai, others } = groupTools(tools, '');
        expect(ai.footprintBytes).toBe(15);
        expect(others.footprintBytes).toBe(27);
    });

    it('过滤同时作用于两段', () => {
        const { ai, others } = groupTools(tools, 'opencode');
        expect(ai.tools).toHaveLength(1);
        expect(others.tools).toHaveLength(0);
    });

    it('搜索只命中其他段时 AI 段为空', () => {
        const { ai, others } = groupTools(tools, 'nodejs');
        expect(ai.tools).toHaveLength(0);
        expect(others.tools.map((t) => t.name)).toEqual(['nodejs']);
    });

    it('空输入返回两个空段', () => {
        const { ai, others } = groupTools([], '');
        expect(ai.tools).toHaveLength(0);
        expect(others.footprintBytes).toBe(0);
    });

    it('已卸载且无占用的工具被排除在列表外', () => {
        const list = [
            tool('marscode', 'ai-ide', 0, undefined, true),
            tool('trae-cn', 'ai-ide', 10, undefined, true),
            tool('claude', 'agent', 20),
        ];
        const { ai } = groupTools(list, '');
        expect(ai.tools.map((t) => t.name)).toEqual(['trae-cn', 'claude']);
        expect(ai.footprintBytes).toBe(30);
    });
});

describe('pathInside', () => {
    it('unix 路径：严格子路径命中', () => {
        expect(pathInside('/home/u/.kimi-code', '/home/u/.kimi-code/bin')).toBe(true);
        expect(pathInside('/home/u/.kimi-code', '/home/u/.kimi-code')).toBe(false);
        expect(pathInside('/home/u/.kimi-code', '/home/u/.kimi-code-x')).toBe(false);
        expect(pathInside('/home/u/.kimi-code', '/home/u/.npm')).toBe(false);
    });
    it('带尾斜杠与 Windows 反斜杠路径', () => {
        expect(pathInside('/home/u/.kimi-code/', '/home/u/.kimi-code/bin')).toBe(true);
        expect(pathInside('C:\\Users\\u\\.kimi-code', 'C:\\Users\\u\\.kimi-code\\bin')).toBe(true);
    });
});

describe('formSubtotals', () => {
    it('按 form 分形态合计', () => {
        const s = formSubtotals([
            { path: '/a/app.app', bytes: 300, form: 'desktop' },
            { path: '/a/data', bytes: 100, form: 'desktop' },
            { path: '/b/cli-data', bytes: 50 },
        ]);
        expect(s.desktop).toBe(400);
        expect(s.cli).toBe(50);
    });
    it('父子目录极大路径去重', () => {
        const s = formSubtotals([
            { path: '/home/u/.kimi-code', bytes: 556 },
            { path: '/home/u/.kimi-code/bin', bytes: 353 },
            { path: '/a/Kimi Code.app', bytes: 295, form: 'desktop' },
        ]);
        expect(s.cli).toBe(556); // bin 被父目录包含，不重复计入
        expect(s.desktop).toBe(295);
    });
    it('空输入返回零', () => {
        expect(formSubtotals([])).toEqual({ desktop: 0, cli: 0 });
    });
});

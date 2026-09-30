import { describe, expect, it } from 'vitest';
import { groupTools, matchesFilter, pathInside } from './grouping';
import type { Tool } from './types';

// 与 Go JSON 契约同构的最小 Tool 构造（grouping 只消费 name/aliases/
// category/footprintBytes）。
function tool(name: string, category?: string, footprintBytes = 0, aliases?: string[]): Tool {
    return {
        name,
        aliases: aliases ?? [name + '-alias'],
        installer: 'other',
        version: '',
        updatedAt: '',
        homepage: '',
        description: '',
        category,
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

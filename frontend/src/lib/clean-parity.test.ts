import {describe, expect, it} from 'vitest';
import {keepMaximalPaths} from './grouping';
import type {Cleanable, Tool} from './types';

// CLI `clean --dry-run --json` 与 GUI 确认弹窗必须给出**同一个合计**。
//
// 两边的去重口径各有一份实现：Go 的 `internal/cleaner.isInside` 与 TS 的
// `pathInside`。口径一旦分叉，用户就会看到「弹窗说 7.9G，执行后只回收
// 3.9G」。这里测的是 **GUI 真实代码**（render.ts 的 selectedItems 就调用
// keepMaximalPaths），不是复刻；Go 侧由 internal/cleaner 的测试覆盖同一组
// 场景，两边用例一一对应（见 cleaner_subsumed_test.go）。

function mkCleanable(path: string, bytes: number): Cleanable {
    return {
        id: path, tool: 'x', path, bytes, tier: 'safe', kind: 'cache',
        keep: '', desc: '', sub: [],
    } as unknown as Cleanable;
}

const total = (items: Cleanable[]) =>
    keepMaximalPaths(items).reduce((acc, it) => acc + it.bytes, 0);

describe('可处置合计口径（GUI 真实实现）', () => {
    const cases: Array<{name: string; items: Cleanable[]; want: number}> = [
        {
            name: '互不相交：全额相加',
            items: [mkCleanable('/a', 100), mkCleanable('/b', 200)],
            want: 300,
        },
        {
            name: '父子并存：只计父项',
            items: [mkCleanable('/a', 100), mkCleanable('/a/child', 40)],
            want: 100,
        },
        {
            name: '三层嵌套：只计最外层',
            items: [
                mkCleanable('/a', 100),
                mkCleanable('/a/b', 60),
                mkCleanable('/a/b/c', 30),
            ],
            want: 100,
        },
        {
            name: '前缀相似但不是父子：都计入',
            // /a 与 /ab 是同级目录，前缀匹配不能误判为包含
            items: [mkCleanable('/a', 100), mkCleanable('/ab', 200)],
            want: 300,
        },
        {
            name: '子项先选、父项后选：仍只计父项',
            items: [mkCleanable('/a/b', 40), mkCleanable('/a', 100)],
            want: 100,
        },
    ];

    for (const c of cases) {
        it(c.name, () => {
            expect(total(c.items)).toBe(c.want);
        });
    }

    // pathInside 的分隔符语义：Windows 反斜杠路径先归一为正斜杠再比较。
    // 这是 GUI 的口径；Go 侧走 filepath.Rel（平台感知），两者在各自平台上
    // 对同一份输入给出相同结论——见 internal/cleaner 的对应用例。
    it('Windows 反斜杠路径：GUI 归一后判为父子', () => {
        const items = [mkCleanable('C:\\x', 100), mkCleanable('C:\\x\\y', 50)];
        expect(total(items)).toBe(100);
    });
});

// 真实快照形状的回归：全选时合计不能超过总占用。
describe('全选合计不超过总占用', () => {
    it('父子子目录规则不会让合计超过 footprint', () => {
        const tool = {
            name: 'marscode',
            aliases: [], installer: 'other', version: '', updatedAt: '',
            homepage: '', description: '', binaries: [], dataDirs: [],
            cleanables: [
                mkCleanable('/home/.marscode', 3_900_000_000),
                mkCleanable('/home/.marscode/.ckg', 3_800_000_000),
            ],
            footprintBytes: 3_900_000_000,
            cleanableBytes: 3_900_000_000,
            userBytes: 0,
        } as unknown as Tool;

        const sum = total(tool.cleanables);
        expect(sum).toBeLessThanOrEqual(tool.footprintBytes);
        expect(sum).toBe(3_900_000_000);
    });
});

import {describe, expect, it} from 'vitest';
import {keepMaximalPaths, pathInside} from './grouping';

// 确认弹窗的「N 项」与用户勾选数可能不一致：被上层目录包含的条目随父项
// 一起处置，不重复计。这里锁定弹窗标注的判定与后端同口径——否则用户勾了
// 2 项看到「处置 1 项」会以为漏了东西。
//
// 真实数据来自实机 joycoder：父 ~/Library/Application Support/JoyCoder
// (22.4MB) 与子 .../CachedData (10.1MB) 同时可勾选。

interface Item {
    path: string;
    bytes: number;
    subsumed?: boolean;
}

// markSubsumed 复刻 render.ts allSelectedItems 的标注逻辑：
// 对**未去重**的勾选集合逐项判断是否被另一项包含。
function markSubsumed(items: Item[]): Item[] {
    return items.map((it) => ({
        ...it,
        subsumed: items.some((o) => o.path !== it.path && pathInside(o.path, it.path)),
    }));
}

describe('确认弹窗：被包含条目标注', () => {
    it('父子同选：子项被标注，弹窗显示 2 行但只处置 1 项', () => {
        const all = markSubsumed([
            {path: '/Users/wei/Library/Application Support/JoyCoder', bytes: 23491424},
            {path: '/Users/wei/Library/Application Support/JoyCoder/CachedData', bytes: 10576864},
        ]);
        // 弹窗要展示全部勾选项（2 行），其中 1 行带标记
        expect(all.length).toBe(2);
        expect(all.filter((i) => i.subsumed).length).toBe(1);
        expect(all[1].subsumed).toBe(true);
        // 实际处置项 = 去重后的数量
        const active = keepMaximalPaths(all);
        expect(active.length).toBe(1);
        expect(active[0].bytes).toBe(23491424);
    });

    it('互不相交时不标注，处置项数 = 勾选数', () => {
        const all = markSubsumed([
            {path: '/a/.npm', bytes: 100},
            {path: '/a/.cache', bytes: 200},
        ]);
        expect(all.filter((i) => i.subsumed).length).toBe(0);
        expect(keepMaximalPaths(all).length).toBe(2);
    });

    it('只勾子项时不标注（父项未选）', () => {
        const all = markSubsumed([
            {path: '/Users/wei/Library/Application Support/JoyCoder/CachedData', bytes: 10576864},
        ]);
        expect(all.filter((i) => i.subsumed).length).toBe(0);
        expect(keepMaximalPaths(all).length).toBe(1);
    });

    it('三层嵌套：内层两个都被标注，只处置最外层', () => {
        const all = markSubsumed([
            {path: '/a', bytes: 100},
            {path: '/a/b', bytes: 60},
            {path: '/a/b/c', bytes: 30},
        ]);
        expect(all.filter((i) => i.subsumed).length).toBe(2);
        expect(keepMaximalPaths(all).length).toBe(1);
    });

    it('前缀相似但不是父子：都不标注', () => {
        const all = markSubsumed([
            {path: '/a', bytes: 100},
            {path: '/ab', bytes: 200},
        ]);
        expect(all.filter((i) => i.subsumed).length).toBe(0);
    });
});

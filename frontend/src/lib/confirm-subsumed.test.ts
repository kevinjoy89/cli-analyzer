import {describe, expect, it} from 'vitest';
import {pathInside} from './grouping';

// 渲染层回归（曾出过的 bug）：
//   确认弹窗曾把「勾了 2 项只处置 1 项」静静吞掉 —— 因为 selectedItems()
//   在弹窗之前就去重了，弹窗拿到的 items 里没有子项，标注判定永远为空。
//   修复后：弹窗用 allSelectedItems()（全量 + subsumed 标记）渲染，
//   用 selectedItems()（去重后）算合计与处置。
//
// 这里断言的是「标注判定」这个口径本身；render.ts 的 allSelectedItems
// 与 flows.ts 的 showConfirmModal 共用同一套 pathInside 语义。

interface Item {
    path: string;
    subsumed?: boolean;
}

// markSubsumed 复刻 render.ts allSelectedItems 的标注逻辑。
function markSubsumed(items: Item[]): Item[] {
    return items.map((it) => ({
        ...it,
        subsumed: items.some((o) => o.path !== it.path && pathInside(o.path, it.path)),
    }));
}

// visibleRows 复刻弹窗渲染的行数：**全部**勾选项都要出现。
function visibleRows(items: Item[]): number {
    return markSubsumed(items).length;
}

describe('确认弹窗：subsumed 标注回归', () => {
    it('父子同选：两行都渲染，子项被标 subsumed', () => {
        const all = markSubsumed([
            {path: '/Users/wei/Library/Application Support/JoyCoder'},
            {path: '/Users/wei/Library/Application Support/JoyCoder/CachedData'},
        ]);
        // 回归点：曾经只有 1 行（去重早于弹窗）
        expect(visibleRows(all)).toBe(2);
        expect(all.filter((i) => i.subsumed).length).toBe(1);
        expect(all[1].subsumed).toBe(true);
        expect(all[0].subsumed).toBeFalsy();
    });

    it('互不相交：两行都不标', () => {
        const all = markSubsumed([{path: '/a/.npm'}, {path: '/a/.cache'}]);
        expect(visibleRows(all)).toBe(2);
        expect(all.filter((i) => i.subsumed).length).toBe(0);
    });

    it('只勾子项：不标（父项未选）', () => {
        const all = markSubsumed([{path: '/a/parent/child'}]);
        expect(all.filter((i) => i.subsumed).length).toBe(0);
    });

    it('三层嵌套：内层两个都标', () => {
        const all = markSubsumed([{path: '/a'}, {path: '/a/b'}, {path: '/a/b/c'}]);
        expect(all.filter((i) => i.subsumed).length).toBe(2);
    });
});

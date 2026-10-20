// 版本单元格的 HTML 拼装（纯函数，可单测）。
//
// 双形态工具的两个版本分行展示：
//
//   CLI 2.1.0
//   桌面端 1.0.2
//
// 不做悬浮提示，也不写「版本不一致」文案——把两个值直接摆出来，
// 是否相同用户一眼就能看出来。
//
// 抽成纯函数而不是内联在 render.ts：渲染层的 bug（曾因 flex 折行导致
// 桌面端版本被拆成两行）用纯函数测试能直接断言 HTML 结构。
import {esc} from '../dom';
import type {Tool} from './types';

export interface VersionLabels {
    cli: string;
    desktop: string;
}

export function versionCellHtml(tool: Tool, labels: VersionLabels): string {
    const cli = tool.cliVersion || '';
    const desk = tool.desktopVersion || '';

    // 两种形态都有版本：分行列，各自带来源前缀
    if (cli && desk) {
        return (
            `<span class="ver ver-stack">` +
            `<span class="ver-line">${esc(labels.cli)} ${esc(cli)}</span>` +
            `<span class="ver-line">${esc(labels.desktop)} ${esc(desk)}</span>` +
            `</span>`
        );
    }
    // 只有一种形态（或无形态区分）：单行，不加前缀
    return `<span class="ver">${esc(cli || desk || tool.version || '—')}</span>`;
}

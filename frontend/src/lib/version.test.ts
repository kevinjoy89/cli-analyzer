import {describe, expect, it} from 'vitest';
import {versionCellHtml} from './version';
import type {Tool} from './types';

// 渲染层回归（曾出过的 bug）：
//   1. 双版本两行渲染 —— 曾因 flex + white-space:normal 导致长版本号折行，
//      且因去重位置错导致第二行根本不出现。
//   2. 这里断言的是**生成的 HTML 结构**，不是「某个数字存在」——结构对了
//      样式才有作用对象。

function mkTool(p: Partial<Tool>): Tool {
    return {
        name: 'x', aliases: [], installer: 'other', version: '',
        updatedAt: '', homepage: '', description: '',
        binaries: [], dataDirs: [], cleanables: [],
        footprintBytes: 0, cleanableBytes: 0, userBytes: 0,
        ...p,
    } as Tool;
}

const labels = {cli: 'CLI', desktop: '桌面端'};

describe('versionCellHtml', () => {
    it('两种形态都有版本：两个 .ver-line 分行', () => {
        const html = versionCellHtml(
            mkTool({version: '2.1.0', cliVersion: '2.1.0', desktopVersion: '1.0.2'}),
            labels,
        );
        // 外层容器带 ver-stack（样式依赖它做分行）
        expect(html).toContain('class="ver ver-stack"');
        // 恰好两个 .ver-line
        expect(html.match(/class="ver-line"/g)?.length).toBe(2);
        expect(html).toContain('CLI 2.1.0');
        expect(html).toContain('桌面端 1.0.2');
    });

    it('长版本号完整保留（不截断、不折行靠样式保证）', () => {
        const html = versionCellHtml(
            mkTool({version: '0.156.1', cliVersion: '0.156.1', desktopVersion: '26.1013.31256'}),
            labels,
        );
        expect(html).toContain('桌面端 26.1013.31256');
    });

    it('两版本相同也分行（不隐藏来源）', () => {
        const html = versionCellHtml(
            mkTool({version: '1.0.2', cliVersion: '1.0.2', desktopVersion: '1.0.2'}),
            labels,
        );
        expect(html.match(/class="ver-line"/g)?.length).toBe(2);
    });

    it('只有桌面端：单行且不带 ver-stack', () => {
        const html = versionCellHtml(mkTool({version: '5.7.7', desktopVersion: '5.7.7'}), labels);
        expect(html).not.toContain('ver-stack');
        expect(html).toContain('5.7.7');
    });

    it('完全未知：显示破折号', () => {
        expect(versionCellHtml(mkTool({}), labels)).toContain('—');
    });

    it('HTML 转义：版本串里的尖括号不会被当标签', () => {
        const html = versionCellHtml(mkTool({cliVersion: '<b>1</b>', desktopVersion: '<i>2</i>'}), labels);
        expect(html).not.toContain('<b>');
        expect(html).toContain('&lt;b&gt;');
    });
});

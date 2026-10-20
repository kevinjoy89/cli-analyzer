import {describe, expect, it} from 'vitest';
import {shortPath} from './format';

// 渲染层回归（曾出过的 bug）：
//   内置回收站列表里的 `/Users/wei/Library/Application Support/JoyCoder`
//   被 CSS 截断成 `/Users/wei/Library/Application Supp...` —— 恰好把能标识
//   这个目录的部分切掉了。改为 `~/Library/Application Support/JoyCoder`。

describe('shortPath（路径简写回归）', () => {
    const home = '/Users/wei';

    it('主目录子路径换成 ~（回收站列表曾截断的那一条）', () => {
        expect(shortPath('/Users/wei/Library/Application Support/JoyCoder', home)).toBe(
            '~/Library/Application Support/JoyCoder',
        );
    });

    it('主目录本身换成 ~', () => {
        expect(shortPath('/Users/wei', home)).toBe('~');
    });

    it('主目录外保持不变', () => {
        expect(shortPath('/Applications/WorkBuddy.app', home)).toBe('/Applications/WorkBuddy.app');
        expect(shortPath('/opt/homebrew/lib/node_modules/npm', home)).toBe(
            '/opt/homebrew/lib/node_modules/npm',
        );
    });

    it('前缀相似但不是子路径：不能误替换', () => {
        expect(shortPath('/Users/wei2/foo', home)).toBe('/Users/wei2/foo');
        expect(shortPath('/Users/weibull', home)).toBe('/Users/weibull');
    });

    it('Windows 反斜杠路径：保留原分隔符', () => {
        expect(shortPath('C:\\Users\\wei\\AppData\\Roaming\\X', 'C:\\Users\\wei')).toBe(
            '~\\AppData\\Roaming\\X',
        );
    });

    it('home 为空时原样返回（不猜测）', () => {
        expect(shortPath('/Users/wei/foo', '')).toBe('/Users/wei/foo');
    });

    it('空路径原样返回', () => {
        expect(shortPath('', home)).toBe('');
    });

    it('末尾斜杠不影响判定', () => {
        expect(shortPath('/Users/wei/foo', '/Users/wei/')).toBe('~/foo');
    });
});

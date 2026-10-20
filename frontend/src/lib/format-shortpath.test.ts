import {describe, expect, it} from 'vitest';
import {shortPath} from './format';

// 路径简写：把主目录前缀换成 `~`。
// 动机：列表里 `/Users/wei/Library/Application Support/JoyCoder` 被截断成
// `/Users/wei/Library/Application Supp...`，有效信息全丢；`~/Library/...`
// 能多显示一截。

describe('shortPath', () => {
    const home = '/Users/wei';

    it('主目录下的子路径换成 ~', () => {
        expect(shortPath('/Users/wei/Library/Application Support/JoyCoder', home))
            .toBe('~/Library/Application Support/JoyCoder');
    });

    it('主目录本身换成 ~', () => {
        expect(shortPath('/Users/wei', home)).toBe('~');
    });

    it('主目录外的路径保持不变', () => {
        expect(shortPath('/Applications/WorkBuddy.app', home)).toBe('/Applications/WorkBuddy.app');
        expect(shortPath('/opt/homebrew/lib/node_modules/npm', home)).toBe('/opt/homebrew/lib/node_modules/npm');
    });

    it('前缀相似但不是主目录：不能误替换', () => {
        // /Users/wei2 不是 /Users/wei 的子路径
        expect(shortPath('/Users/wei2/foo', home)).toBe('/Users/wei2/foo');
        expect(shortPath('/Users/weibull', home)).toBe('/Users/weibull');
    });

    it('Windows 反斜杠路径也能识别', () => {
        expect(shortPath('C:\\Users\\wei\\AppData\\Roaming\\X', 'C:\\Users\\wei'))
            .toBe('~\\AppData\\Roaming\\X');
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

// 守护 lib/labels.ts 与 internal/i18n/locales/*.json 不漂移：
// 映射表里登记的每个键必须存在于全部三种语言的字典中，否则新类型漏登记
// 时界面会直接展示英文内部名（如 logs），此测试会立即失败。
import {describe, expect, it} from 'vitest';
import en from '../../../internal/i18n/locales/en.json';
import tw from '../../../internal/i18n/locales/zh-TW.json';
import zh from '../../../internal/i18n/locales/zh-CN.json';
import {KIND_LABEL_KEY, ROOT_LABEL_KEY, aliasMeta} from './labels';

const DICTS: Array<{lang: string; dict: Record<string, string>}> = [
    {lang: 'zh-CN', dict: zh},
    {lang: 'zh-TW', dict: tw},
    {lang: 'en', dict: en},
];

describe('labels 与 locale 字典一致性', () => {
    it('KIND_LABEL_KEY 的每个键在三种语言字典中都存在', () => {
        for (const key of Object.values(KIND_LABEL_KEY)) {
            for (const {lang, dict} of DICTS) {
                expect(dict[key], `kind 键 ${key} 在 ${lang} 中缺失`).toBeTruthy();
            }
        }
    });

    it('ROOT_LABEL_KEY 的每个键在三种语言字典中都存在', () => {
        for (const key of Object.values(ROOT_LABEL_KEY)) {
            for (const {lang, dict} of DICTS) {
                expect(dict[key], `root 键 ${key} 在 ${lang} 中缺失`).toBeTruthy();
            }
        }
    });

    it('字典中不应存在孤儿键（映射表删除后残留）', () => {
        const mapped = new Set([...Object.values(KIND_LABEL_KEY), ...Object.values(ROOT_LABEL_KEY)]);
        for (const {lang, dict} of DICTS) {
            for (const key of Object.keys(dict)) {
                if (key.startsWith('ui.kindLabel.') || key.startsWith('ui.root.')) {
                    expect(mapped.has(key), `${key} 在 ${lang} 中残留但映射表未使用`).toBe(true);
                }
            }
        }
    });
});

describe('aliasMeta 别名行展示判定', () => {
    it('家族合并工具 →「包含工具」', () => {
        expect(aliasMeta({family: 'nodejs', aliases: ['node', 'npm', 'npx', 'corepack', 'node-gyp']})).toEqual({labelKey: 'ui.bundledTools'});
    });

    it('普通工具 ≤3 条别名 →「别名」', () => {
        expect(aliasMeta({aliases: ['claude-code']})).toEqual({labelKey: 'ui.aliases'});
        expect(aliasMeta({aliases: ['a', 'b', 'c']})).toEqual({labelKey: 'ui.aliases'});
    });

    it('普通工具 >3 条别名 → 不展示（shims 噪音）', () => {
        expect(aliasMeta({aliases: ['python', 'pip', 'pytest', 'pipx']})).toBeNull();
    });

    it('无别名 → 不展示', () => {
        expect(aliasMeta({aliases: []})).toBeNull();
        expect(aliasMeta({})).toBeNull();
    });
});

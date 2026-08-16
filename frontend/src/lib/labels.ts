// 内部标识 → 本地化键名 的映射表。
// 与 internal/i18n/locales/*.json 同步维护：labels.test.ts 会校验两边不漂移
// （映射表里的每个键必须存在于全部三种语言的字典中），防止新类型漏登记
// 导致界面直接展示英文内部名。
import {t} from './i18n';

// 数据根 → 本地化标签（未在表内的根显示原始值）
export const ROOT_LABEL_KEY: Record<string, string> = {
    'xdg-config': 'ui.root.config',
    'xdg-cache': 'ui.root.cache',
    'xdg-data': 'ui.root.data',
    'xdg-state': 'ui.root.state',
    'appdata': 'ui.root.appdata',
    'localappdata': 'ui.root.localappdata',
    'home': 'ui.root.home',
    'macos-application-support': 'ui.root.macappsupport',
    'macos-caches': 'ui.root.maccaches',
    'macos-preferences': 'ui.root.macprefs',
};

// 类型短标签（回收站条目 / 详情页 / 确认弹窗共用）。后端 JSON 里 kind 是
// 内部英文标识（cache/old-version/data/logs…），不能直接展示；未收录的
// 类型回退原始值，保证新类型不显示空白。
export const KIND_LABEL_KEY: Record<string, string> = {
    'cache': 'ui.kindLabel.cache',
    'old-version': 'ui.kindLabel.oldVersion',
    'backup': 'ui.kindLabel.backup',
    'download': 'ui.kindLabel.download',
    'toolchain': 'ui.kindLabel.toolchain',
    'data': 'ui.kindLabel.data',
    'config': 'ui.kindLabel.config',
    'state': 'ui.kindLabel.state',
    'install': 'ui.kindLabel.install',
    'logs': 'ui.kindLabel.logs',
};

export function orphanRootLabel(root: string): string {
    const k = ROOT_LABEL_KEY[root];
    return k ? t(k) : root;
}

export function kindLabel(kind: string): string {
    const k = KIND_LABEL_KEY[kind];
    return k ? t(k) : kind;
}

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

// AI 语义类型（add-ai-tool-catalog）：会话/数据库/索引/运行时等。session-db
// 是「仅报告」类型：后端不为其生成可处置项，前端对这类子项禁用勾选。
export const AI_KIND_LABEL_KEY: Record<string, string> = {
    'session': 'ui.kindLabel.session',
    'session-db': 'ui.kindLabel.session-db',
    'index': 'ui.kindLabel.index',
    'runtime': 'ui.kindLabel.runtime',
    'plugins': 'ui.kindLabel.plugins',
    'artifacts': 'ui.kindLabel.artifacts',
    'telemetry': 'ui.kindLabel.telemetry',
    'checkpoint': 'ui.kindLabel.checkpoint',
    'workspace-data': 'ui.kindLabel.workspace-data',
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
    ...AI_KIND_LABEL_KEY,
};

// AI 编目类别（agent / ai-ide）
export const CATEGORY_LABEL_KEY: Record<string, string> = {
    'agent': 'ui.category.agent',
    'ai-ide': 'ui.category.ai-ide',
};

export function orphanRootLabel(root: string): string {
    const k = ROOT_LABEL_KEY[root];
    return k ? t(k) : root;
}

export function kindLabel(kind: string): string {
    const k = KIND_LABEL_KEY[kind];
    return k ? t(k) : kind;
}

export function categoryLabel(category: string): string {
    const k = CATEGORY_LABEL_KEY[category];
    return k ? t(k) : category;
}

// 会话数据库「仅报告」判定：与后端 subKind 的 isSessionDBFile 保持一致。
export function isSessionDBKind(kind: string): boolean {
    return kind === 'session-db';
}

// 全局状态（原 main.ts 顶层 let）与前端接口类型。
// 消费方用 `import * as state from './state'` 后 `state.selected = x` 读写。
import type { ReminderConfig, ScanResult, Tool, TrendsResult } from './lib/types';

// ---- 与 Go JSON 契约对应的前端接口类型（契约单一来源 lib/types.ts 之外的前端私有类型）----
export interface CleanReport { dryRun: boolean; deleted: string[]; freedBytes: number; skipped: string[]; errors: string[] }
export interface TrashInfoData { items: number; totalBytes: number; earliestExpiresAt: string }
export interface TrashItem { id: string; original: string; tool: string; kind: string; bytes: number; trashedAt: string; expiresAt: string }
export interface TrashConfig { retentionDays: number; expireAction: string; useTrash: boolean }
export interface UpdateConfig { checkUpdates?: boolean; lastCheckAt?: string; ignoredVersion?: string }
export interface UpdateResult { current: string; latest: string; updateAvailable: boolean; assetName?: string; downloadURL?: string; releaseURL?: string; installSource?: string; error?: string }
export interface UpdateDownloaded { path: string; releaseURL: string; executablePath: string; installSource: string }
export interface UninstallStartInfo { tool: string; installer?: string; blocked?: boolean; stale?: boolean; blockedReason?: string; officialCommand?: string; runnable?: boolean; footprint?: number; userBytes?: number; error?: string }
export interface UninstallResidueItem { path: string; bytes: number; tier: string; kind: string }
export interface UninstallStatus { running: boolean; done: boolean; output: string; error?: string }
export type UpdateProgress = { downloaded: number; total: number };

// ---- 工具升级（upgrade）----
export interface UpgradeCheckResult { name: string; installer?: string; current?: string; latest?: string; detected: boolean; hasUpdate: boolean; command?: string; runnable: boolean; error?: string }
export interface UpgradeStatus { running: boolean; done: boolean; output: string; error?: string }

// ---- 扫描 / 界面状态 ----
export let result: ScanResult | null = null;
export let probing = false; // 健康探测进行中（版本列显示 …）
export let selected: string | null = null;
export let appVersion = '';
export let filterText = '';
export let selectedCleanIds = new Set<string>();
export let expandedCleanIds = new Set<string>(); // cleanable ids whose sub-breakdown is expanded
export type SortKey = 'name' | 'version' | 'footprint' | 'cleanable' | 'user';
export let sortKey: SortKey = 'footprint';
export let sortDir: 1 | -1 = -1;

// ---- 主题 ----
export type ThemeMode = 'system' | 'light' | 'dark';
export let themeMode: ThemeMode = 'system';
export const THEME_META: Record<ThemeMode, {icon: string; labelKey: string}> = {
    system: {icon: '◐', labelKey: 'ui.themeSystem'},
    light: {icon: '☀', labelKey: 'ui.themeLight'},
    dark: {icon: '☾', labelKey: 'ui.themeDark'},
};

// ---- i18n / 平台 ----
export let isMac = false;

// ---- 未认领数据视图 ----
export let panelView: 'tools' | 'orphans' = 'tools';
export let orphanSel = new Set<string>(); // 已勾选的未认领路径

// ---- 内置回收站 ----
export let trashItems: TrashItem[] = [];

// ---- 更新流程状态 ----
export let updateState: 'idle' | 'downloading' | 'downloaded' = 'idle';
export let lastUpdateResult: UpdateResult | null = null;
export let downloadPoll: number | null = null;
export let lastShownPct = 0;

// ---- 卸载流程状态 ----
export let uninstallPoll: number | null = null;

// ---- 升级流程状态 ----
export let upgradePoll: number | null = null;

// ---- 待清理提醒 ----
export let reminderTools: Tool[] = [];

// 引用类型的赋值语义：Set 内部变更无需 setter（引用不变）；
// 原始值/整体替换走以下 setter，保证跨模块可写。
export function setResult(v: ScanResult | null) { result = v; }
export function setProbing(v: boolean) { probing = v; }
export function setSelected(v: string | null) { selected = v; }
export function setAppVersion(v: string) { appVersion = v; }
export function setFilterText(v: string) { filterText = v; }
export function setSortKey(v: SortKey) { sortKey = v; }
export function setSortDir(v: 1 | -1) { sortDir = v; }
export function setThemeMode(v: ThemeMode) { themeMode = v; }
export function setIsMac(v: boolean) { isMac = v; }
export function setPanelView(v: 'tools' | 'orphans') { panelView = v; }
export function setTrashItems(v: TrashItem[]) { trashItems = v; }
export function setUpdateState(v: 'idle' | 'downloading' | 'downloaded') { updateState = v; }
export function setLastUpdateResult(v: UpdateResult | null) { lastUpdateResult = v; }
export function setDownloadPoll(v: number | null) { downloadPoll = v; }
export function setLastShownPct(v: number) { lastShownPct = v; }
export function setUninstallPoll(v: number | null) { uninstallPoll = v; }
export function setUpgradePoll(v: number | null) { upgradePoll = v; }
export function setReminderTools(v: Tool[]) { reminderTools = v; }

// 类型再导出：消费方从 state 拿全量类型，main.ts 不再重复声明
export type { ReminderConfig, ScanResult, Tool, TrendsResult };

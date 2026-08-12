import './style.css';

import {CancelDownload, CheckForUpdates, Clean, DownloadUpdate, GetDownloadProgress, GetLanguage, GetLastResult, GetReminderConfig, GetTranslations, GetTrashConfig, GetUninstallStatus, GetUpdateStatus, GetTrends, GetUpdateConfig, GetVersion, IgnoreVersion, InstallUpdate, OpenURL, PurgeNow, Restore, Scan, SetLanguage, SetLanguagePreference, SetReminderConfig, SetTheme, SetTrashConfig, SetUpdateConfig, TrashInfo, TrashList, UninstallBlocked, UninstallResidue, UninstallRunOfficial, UninstallStart, UninstallTrashResidues} from '../wailsjs/go/gui/ScannerService';
import {Environment, EventsOn, Quit} from '../wailsjs/runtime/runtime';
import {applyCleanLocally} from './lib/clean';
import {hb} from './lib/format';
import {activeLocale, fmtTime, normalizeNavigator, setDict, t} from './lib/i18n';
import {computeTrendPaths} from './lib/trends';
import {Cleanable, Grower, Point, ReminderConfig, ScanResult, Tool, TrendsResult} from './lib/types';

// ---- types mirroring the Go JSON contract ----
interface CleanReport { dryRun: boolean; deleted: string[]; freedBytes: number; skipped: string[]; errors: string[] }
interface TrashInfoData { items: number; totalBytes: number; earliestExpiresAt: string }
interface TrashItem { id: string; original: string; tool: string; kind: string; bytes: number; trashedAt: string; expiresAt: string }
interface TrashConfig { retentionDays: number; expireAction: string; useTrash: boolean }
interface UpdateConfig { checkUpdates?: boolean; lastCheckAt?: string; ignoredVersion?: string }
interface UpdateResult { current: string; latest: string; updateAvailable: boolean; assetName?: string; downloadURL?: string; releaseURL?: string; installSource?: string; error?: string }
interface UpdateDownloaded { path: string; releaseURL: string; executablePath: string; installSource: string }

// ---- state ----
let result: ScanResult | null = null;
let selected: string | null = null;
let appVersion = '';
let filterText = '';
let selectedCleanIds = new Set<string>();
let expandedCleanIds = new Set<string>(); // cleanable ids whose sub-breakdown is expanded
type SortKey = 'name' | 'version' | 'footprint' | 'cleanable' | 'user';
let sortKey: SortKey = 'footprint';
let sortDir: 1 | -1 = -1;

// ---- theme ----
type ThemeMode = 'system' | 'light' | 'dark';
let themeMode: ThemeMode = 'system';
const THEME_META: Record<ThemeMode, {icon: string; labelKey: string}> = {
    system: {icon: '◐', labelKey: 'ui.themeSystem'},
    light: {icon: '☀', labelKey: 'ui.themeLight'},
    dark: {icon: '☾', labelKey: 'ui.themeDark'},
};

function systemDark(): boolean {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme(mode: ThemeMode) {
    themeMode = mode;
    const resolved = mode === 'system' ? (systemDark() ? 'dark' : 'light') : mode;
    document.documentElement.setAttribute('data-theme', resolved);
    const meta = THEME_META[mode];
    el('themeIcon').textContent = meta.icon;
    el('themeBtn').title = t('ui.themeBtnTitle', {label: t(meta.labelKey)});
    // 同步 Windows 原生标题栏/菜单栏主题（macOS/Linux 由系统与 CSS 处理）
    SetTheme(mode);
}

// ---- i18n ----
let isMac = false;
// 解析生效语言：显式配置优先，否则 navigator.language → 回退 zh-CN
async function resolveLocale(): Promise<string> {
    let cfgLang = '';
    try { cfgLang = await GetLanguage(); } catch { /* 默认 auto */ }
    if (cfgLang === 'zh-CN' || cfgLang === 'zh-TW' || cfgLang === 'en') return cfgLang;
    const nav = (typeof navigator !== 'undefined' && navigator.language) ? navigator.language : '';
    return normalizeNavigator(nav);
}

// 从 Go 侧拉取字典并设置为当前语言（GUI/Go 双端同一份语言文件）
async function initI18n(): Promise<string> {
    const locale = await resolveLocale();
    try {
        const raw = JSON.parse(await GetTranslations(locale));
        setDict(raw.locale, raw.dict);
        await SetLanguage(raw.locale); // 握手：同步 Go 侧错误/弹窗语言
    } catch { /* 字典拉取失败时维持空字典，t() 返回键名 */ }
    return activeLocale();
}

// 语言切换后：重扫静态标签 + 重渲染所有依赖文案的界面
function applyI18n() {
    document.querySelectorAll<HTMLElement>('[data-i18n]').forEach(n => {
        // 防御：data-i18n 只应出现在叶子节点；带子元素的节点需用内层 span 包裹
        if (n.querySelector(':scope > *')) {
            console.warn('data-i18n on non-leaf node, skipping', n.dataset.i18n);
            return;
        }
        n.textContent = t(n.dataset.i18n!);
    });
    document.querySelectorAll<HTMLElement>('[data-i18n-title]').forEach(n => {
        n.title = t(n.dataset.i18nTitle!);
    });
    document.querySelectorAll<HTMLInputElement>('[data-i18n-placeholder]').forEach(n => {
        n.placeholder = t(n.dataset.i18nPlaceholder!);
    });
    applyTheme(themeMode);
    renderSummary(); renderToolList();
    renderDetail(); // 无条件重渲染：无选中时的空态也是 JS 生成的，必须覆盖
    refreshTrashInfo(); refreshReminder();
    if (!el('prefsModal').classList.contains('hidden')) openPrefs();
    if (!el('updateModal').classList.contains('hidden') && updateState === 'idle' && lastUpdateResult?.updateAvailable) {
        showUpdateAvailable(lastUpdateResult);
    }
}

// ---- helpers ----
function el<T extends HTMLElement>(id: string): T {
    const e = document.getElementById(id);
    if (!e) throw new Error(`missing element #${id}`);
    return e as T;
}

function showToast(msg: string, isError = false) {
    const t = el<HTMLDivElement>('toast');
    t.textContent = msg;
    t.className = 'toast' + (isError ? ' error' : '');
    window.clearTimeout((t as unknown as { _t?: number })._t);
    (t as unknown as { _t?: number })._t = window.setTimeout(() => { t.className = 'toast hidden'; }, 2800);
}

function setScanning(busy: boolean, label = '') {
    const s = el('scanState');
    s.className = 'scan-state' + (busy ? ' busy' : '');
    s.textContent = busy ? label : '';
    (el('rescanBtn') as HTMLButtonElement).disabled = busy;
}

// ---- summary ----
function renderSummary() {
    if (!result) return;
    el('sumFootprint').textContent = hb(result.totals.footprintBytes);
    el('sumCleanable').textContent = hb(result.totals.cleanableBytes);
    el('sumUser').textContent = hb(result.totals.userBytes);
    el('sumTools').textContent = String(result.tools.length);
    el('lastScan').textContent = result.scannedAt ? t('ui.lastScan', {time: fmtTime(result.scannedAt)}) : '';
    const status = el('statusInfo');
    status.innerHTML = '';
    const parts: Array<[string, string]> = [
        [t('ui.scanTime'), result.scanTimeMs > 0 ? `${(result.scanTimeMs / 1000).toFixed(1)} s` : t('ui.scanCache')],
        [t('ui.walkErrors'), String(result.walkErrors)],
        [t('ui.platform'), result.platform],
    ];
    for (const [k, v] of parts) {
        const span = document.createElement('span');
        span.textContent = `${k}: ${v}`;
        status.appendChild(span);
    }
}

// ---- tool list ----
// Only 总占用 / 可清理 are sortable; the rest are plain column labels.
const COLUMNS: Array<{key: SortKey; labelKey: string; sortable: boolean}> = [
    {key: 'name', labelKey: 'ui.colTool', sortable: false},
    {key: 'version', labelKey: 'ui.colVersion', sortable: false},
    {key: 'footprint', labelKey: 'ui.colFootprint', sortable: true},
    {key: 'cleanable', labelKey: 'ui.colCleanable', sortable: true},
    {key: 'user', labelKey: 'ui.colUser', sortable: true},
];

function sortTools(tools: Tool[]): Tool[] {
    const dir = sortDir;
    const val = (t: Tool, k: SortKey): number | string => {
        switch (k) {
            case 'name': return t.name;
            case 'version': return t.version;
            case 'footprint': return t.footprintBytes;
            case 'cleanable': return t.cleanableBytes;
            case 'user': return t.userBytes;
        }
    };
    return tools.slice().sort((x, y) => {
        const vx = val(x, sortKey);
        const vy = val(y, sortKey);
        if (typeof vx === 'string' && typeof vy === 'string') return vx.localeCompare(vy) * dir;
        return (Number(vx) - Number(vy)) * dir;
    });
}

function renderToolList() {
    const list = el('toolList');
    list.innerHTML = '';
    if (!result) {
        list.innerHTML = '<div class="empty">' + esc(t('ui.noData')) + '</div>';
        return;
    }
    const q = filterText.toLowerCase();
    const tools = sortTools(result.tools.filter(t => !q || t.name.toLowerCase().includes(q) || t.aliases.some(a => a.toLowerCase().includes(q))));
    const head = document.createElement('div');
    head.className = 'tool-header';
    head.innerHTML = COLUMNS.map(({key, labelKey, sortable}) => {
        if (!sortable) return `<span class="h">${esc(t(labelKey))}</span>`;
        const arrow = key === sortKey ? (sortDir === -1 ? ' ▼' : ' ▲') : '';
        return `<button class="h" data-key="${key}">${esc(t(labelKey))}${arrow}</button>`;
    }).join('');
    list.appendChild(head);
    head.querySelectorAll<HTMLButtonElement>('button.h').forEach(btn => {
        btn.onclick = () => {
            const k = btn.dataset.key as SortKey;
            if (k === sortKey) {
                sortDir = sortDir === -1 ? 1 : -1;
            } else {
                sortKey = k;
                sortDir = (k === 'name') ? 1 : -1;
            }
            renderToolList();
        };
    });
    for (const tool of tools) {
        const row = document.createElement('div');
        row.className = 'tool-row' + (tool.name === selected ? ' selected' : '');
        const clean = tool.cleanableBytes > 0 ? `<span class="cleanable-flag">✓</span>` : '';
        row.innerHTML = `
            <span class="tool-name" title="${esc(tool.name)}">${esc(tool.name)}</span>
            <span class="ver">${esc(tool.version || '—')}</span>
            <span class="num">${hb(tool.footprintBytes)}</span>
            <span class="num clean">${hb(tool.cleanableBytes)}${clean}</span>
            <span class="num user">${hb(tool.userBytes)}</span>`;
        row.onclick = () => { selected = tool.name; selectedCleanIds.clear(); expandedCleanIds.clear(); renderToolList(); renderDetail(); };
        list.appendChild(row);
    }
}

// ---- detail ----
function renderDetail() {
    const body = el('detailBody');
    if (!result || !selected) {
        body.innerHTML = '<div class="empty">' + esc(t('ui.selectTool')) + '</div>';
        return;
    }
    const tool = result.tools.find(x => x.name === selected);
    if (!tool) return;

    const installer = tool.installer ? `<span class="badge installer">${esc(tool.installer)}</span>` : '';
    const metaItems: string[] = [];
    // 安装来源：本地化名称（未知来源回退原始值）
    let sourceName = '';
    if (tool.installer) {
        const localized = t('ui.installer.' + tool.installer);
        sourceName = localized !== 'ui.installer.' + tool.installer ? localized : tool.installer;
    }
    if (sourceName) metaItems.push(`<span class="meta-item">${esc(t('ui.installerSource'))} <b>${esc(sourceName)}</b></span>`);
    if (tool.version) metaItems.push(`<span class="meta-item">${esc(t('ui.version'))} <b>${esc(tool.version)}</b></span>`);
    if (tool.updatedAt) metaItems.push(`<span class="meta-item">${esc(t('ui.updatedAt'))} <b>${fmtTime(tool.updatedAt)}</b></span>`);
    const metaHtml = (tool.description || tool.homepage || metaItems.length)
        ? `<div class="detail-meta">
            ${tool.description ? `<div class="meta-desc">${esc(tool.description)}</div>` : ''}
            <div class="meta-row">${metaItems.join('')}${tool.homepage ? `<a class="meta-link" id="hpLink" href="#">${esc(t('ui.homepage'))}</a>` : ''}</div>
          </div>`
        : '';
    const binaries = tool.binaries.length
        ? `<div class="detail-list">${tool.binaries.map(b =>
            `<div class="detail-item"><span class="badge safe">bin</span><span class="path" title="${esc(b.real)}">${esc(b.real)}</span><span class="size">${hb(b.size)}</span></div>`).join('')}</div>`
        : '<div class="detail-list"><div class="detail-item"><span class="muted">' + esc(t('ui.noBinary')) + '</span></div></div>';

    const dataDirs = tool.dataDirs.length
        ? `<div class="detail-list">${tool.dataDirs.map(d =>
            `<div class="detail-item"><span class="badge ${d.tier}">${d.tier}</span><span class="path" title="${esc(d.path)}">${esc(d.path)}</span><span class="size ${d.tier}">${hb(d.bytes)}</span><span class="keep">${esc(d.kind)}</span></div>`).join('')}</div>`
        : '';

    const cleanables = tool.cleanables.filter(c => c.tier === 'safe');
    const cleanHtml = cleanables.length
        ? `<div class="detail-list">${cleanables.map(c => {
            const checked = selectedCleanIds.has(c.id) ? 'checked' : '';
            const expanded = expandedCleanIds.has(c.id);
            const hasSub = (c.sub ?? []).length > 0;
            const toggle = hasSub
                ? `<button class="sub-toggle${expanded ? ' expanded' : ''}" data-id="${esc(c.id)}" title="${esc(t('ui.expandCollapse'))}"><span class="st-caret"></span></button>`
                : `<span class="sub-toggle spacer"></span>`;
            return `<div class="clean-block">
                <div class="detail-item clean-row">
                    ${toggle}
                    <input type="checkbox" data-id="${esc(c.id)}" ${checked}/>
                    <span class="badge safe">${esc(c.kind)}</span>
                    <span class="path" title="${esc(c.path)}">${esc(c.path)}</span>
                    <span class="size clean">${hb(c.bytes)}</span>
                    <span class="keep">${c.keep ? esc(c.keep) : ''}</span>
                </div>${expanded ? subRows(c, selectedCleanIds.has(c.id)) : ''}
            </div>`;
        }).join('')}</div>`
        : '<div class="detail-list"><div class="detail-item"><span class="muted">' + esc(t('ui.noCleanable')) + '</span></div></div>';

    body.innerHTML = `
        <div class="detail-head"><h2>${esc(tool.name)}</h2>${installer}</div>
        ${metaHtml}
        <div class="detail-section"><h4>${esc(t('ui.sectionBinaries'))}</h4>${binaries}</div>
        ${dataDirs ? `<div class="detail-section"><h4>${esc(t('ui.sectionDataDirs'))}</h4>${dataDirs}</div>` : ''}
        <div class="detail-section"><h4>${esc(t('ui.sectionSafe'))}</h4>${cleanHtml}</div>
        <div class="detail-actions">
            <button id="cleanBtn" class="btn danger">${esc(t('ui.cleanSelected'))}</button>
            <button id="uninstallBtn" class="btn" title="${esc(t('un.guiUninstall'))}">${esc(t('un.guiUninstall'))}</button>
            <span class="sel-info" id="selInfo">${esc(t('ui.selectedCount', {n: selectedCleanIds.size}))}</span>
        </div>`;

    const hp = body.querySelector<HTMLAnchorElement>('#hpLink');
    if (hp) hp.onclick = (e) => { e.preventDefault(); OpenURL(tool.homepage); };

    body.querySelectorAll<HTMLInputElement>('input[type="checkbox"]').forEach(cb => {
        cb.onchange = () => {
            const id = cb.dataset.id!;
            const parentId = cb.dataset.parent;
            if (parentId) {
                // child: selecting it deselects the whole-dir parent
                if (cb.checked) { selectedCleanIds.delete(parentId); selectedCleanIds.add(id); }
                else selectedCleanIds.delete(id);
            } else {
                // whole dir: selecting it deselects all its children
                if (cb.checked) {
                    const c = cleanables.find(x => x.id === id);
                    if (c) for (const sid of subIdsOf(c)) selectedCleanIds.delete(sid);
                    selectedCleanIds.add(id);
                } else {
                    selectedCleanIds.delete(id);
                }
            }
            renderDetail();
        };
    });
    body.querySelectorAll<HTMLButtonElement>('button.sub-toggle').forEach(btn => {
        btn.onclick = () => {
            const id = btn.dataset.id!;
            if (expandedCleanIds.has(id)) expandedCleanIds.delete(id); else expandedCleanIds.add(id);
            renderDetail();
        };
    });
    const cleanBtn = el<HTMLButtonElement>('cleanBtn');
    cleanBtn.disabled = selectedCleanIds.size === 0;
    cleanBtn.onclick = () => {
        if (selectedCleanIds.size === 0) return;
        showConfirmModal(selectedItems(tool));
    };
    el<HTMLButtonElement>('uninstallBtn').onclick = () => startUninstall(tool.name);
    // 黑名单工具禁用卸载按钮（UninstallBlocked 来自服务层）
    UninstallBlocked(tool.name).then(raw => {
        try {
            const b = JSON.parse(raw) as {blocked: boolean};
            el<HTMLButtonElement>('uninstallBtn').disabled = !!b.blocked;
        } catch { /* 忽略 */ }
    });
}

function esc(s: string): string {
    return s.replace(/[&<>"']/g, c => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c]!));
}

// One level of size breakdown for a cleanable (its direct children), rendered
// as indented rows below the checkbox row. Non-empty children are individually
// selectable; when the parent (whole dir) is checked they are disabled. Capped
// to keep wide dirs readable.
const SUB_CAP = 20;
function subRows(c: Cleanable, parentSelected: boolean): string {
    const sub = c.sub ?? [];
    if (!sub.length) return '';
    const shown = sub.slice(0, SUB_CAP);
    const more = sub.length - shown.length;
    const base = c.path.endsWith('/') ? c.path : c.path + '/';
    const rows = shown.map(s => {
        const rel = s.path.startsWith(base) ? s.path.slice(base.length) : s.path;
        const selectable = !!(s.id && s.bytes > 0);
        if (!selectable) {
            return `<div class="detail-item sub-row">
                <span class="sub-path" title="${esc(s.path)}">${esc(rel)}</span>
                <span class="size muted">0 B</span>
            </div>`;
        }
        const checked = selectedCleanIds.has(s.id) ? 'checked' : '';
        const disabled = parentSelected ? 'disabled' : '';
        return `<div class="detail-item sub-row">
            <input type="checkbox" class="sub-cb" data-id="${esc(s.id)}" data-parent="${esc(c.id)}" ${checked} ${disabled}/>
            <span class="sub-path" title="${esc(s.path)}">${esc(rel)}</span>
            <span class="size clean">${hb(s.bytes)}</span>
        </div>`;
    }).join('');
    const moreHtml = more > 0 ? `<div class="sub-more">${esc(t('ui.moreSubitems', {n: more}))}</div>` : '';
    return `<div class="sub-list">${rows}${moreHtml}</div>`;
}

// subIdsOf returns the selectable sub-entry ids under a cleanable.
function subIdsOf(c: Cleanable): string[] {
    return (c.sub ?? []).map(s => s.id).filter(Boolean);
}

// selectedItems resolves the currently checked ids (whole cleanables and/or
// their children) into a flat list for the confirm dialog and Clean call.
interface PickItem { id: string; path: string; bytes: number; kind: string }
function selectedItems(t: Tool): PickItem[] {
    const out: PickItem[] = [];
    for (const c of t.cleanables) {
        if (c.tier !== 'safe') continue;
        if (selectedCleanIds.has(c.id)) {
            out.push({id: c.id, path: c.path, bytes: c.bytes, kind: c.kind});
            continue;
        }
        for (const s of c.sub ?? []) {
            if (s.id && selectedCleanIds.has(s.id)) out.push({id: s.id, path: s.path, bytes: s.bytes, kind: c.kind});
        }
    }
    return out;
}

// ---- confirm modal ----
function showConfirmModal(items: PickItem[]) {
    const total = items.reduce((a, c) => a + c.bytes, 0);
    el('modalTitle').textContent = t('ui.confirmClean', {n: items.length, size: hb(total)});
    el('modalBody').innerHTML = items.map(c =>
        `<div class="clean-row">
            <span class="badge safe">${esc(c.kind)}</span>
            <span class="path" style="flex:1;font-family:var(--mono);font-size:11px;white-space:normal;word-break:break-all">${esc(c.path)}</span>
            <span class="size" style="font-family:var(--mono);color:var(--clean);white-space:nowrap">${hb(c.bytes)}</span>
        </div>`).join('')
        + '<div class="warn">' + esc(t('ui.safeOnly')) + '</div>';
    el('modal').classList.remove('hidden');
    el('modalConfirm').onclick = async () => {
        el('modal').classList.add('hidden');
        const ids = items.map(c => c.id);
        try {
            const rep: CleanReport = JSON.parse(await Clean(ids, false));
            const del = (rep.deleted ?? []).length;
            const skipped = (rep.skipped ?? []).length;
            const hasErr = (rep.errors ?? []).length > 0;
            const extra = skipped ? t('ui.cleanSkipped', {n: skipped}) : '';
            showToast(t('ui.cleanDone', {del, size: hb(rep.freedBytes), extra}), hasErr);
            selectedCleanIds.clear();
            if (del > 0 && result) {
                // 本地立即刷新详情，无需等整体重扫
                applyCleanLocally(result, ids);
                renderSummary(); renderToolList(); renderDetail(); refreshReminder(); refreshTrashInfo();
            }
            rescan();
        } catch (e) {
            showToast(t('ui.cleanFailed', {err: String(e)}), true);
        }
    };
    el('modalCancel').onclick = () => el('modal').classList.add('hidden');
}

// ---- built-in trash ----
let trashItems: TrashItem[] = [];

// 刷新状态栏常驻的回收站占用信息（空回收站显示 0 B）
async function refreshTrashInfo() {
    try {
        const info: TrashInfoData = JSON.parse(await TrashInfo());
        const total = info.totalBytes || 0;
        el('trashInfo').textContent = total > 0 ? t('ui.trashInfo', {size: hb(total), n: info.items}) : t('ui.trashInfoEmpty');
        el('trashBtn').title = info.earliestExpiresAt
            ? t('ui.trashBtnFull', {size: hb(total), n: info.items, time: fmtTime(info.earliestExpiresAt)})
            : t('ui.trashBtnTitle');
    } catch { el('trashInfo').textContent = t('ui.trashInfoEmpty'); }
}

function openTrashPanel() {
    el('trashModal').classList.remove('hidden');
    refreshTrashList();
}

async function refreshTrashList() {
    const body = el('trashBody');
    try {
        const parsed = JSON.parse(await TrashList());
        if (parsed.error) { body.innerHTML = `<div class="warn">${esc(parsed.error)}</div>`; return; }
        trashItems = parsed;
    } catch (e) {
        body.innerHTML = `<div class="warn">${esc(t('ui.trashReadFailed', {err: String(e)}))}</div>`;
        return;
    }
    if (!trashItems.length) { body.innerHTML = '<div class="empty">' + esc(t('ui.trashEmpty')) + '</div>'; return; }
    body.innerHTML = trashItems.map(it => `
        <div class="trash-item">
            <div class="trash-head">
                <span class="badge safe">${esc(it.kind)}</span>
                <span class="trash-tool">${esc(it.tool)}</span>
                <span class="size clean">${hb(it.bytes)}</span>
                <span class="trash-exp">${esc(t('ui.trashExp', {time: fmtTime(it.trashedAt), time2: fmtTime(it.expiresAt)}))}</span>
            </div>
            <div class="trash-path" title="${esc(it.original)}">${esc(it.original)}</div>
            <div class="trash-actions">
                <button class="btn small" data-restore="${esc(it.id)}">${esc(t('ui.restore'))}</button>
                <button class="btn small danger" data-purge="${esc(it.id)}">${esc(t('ui.purge'))}</button>
            </div>
        </div>`).join('');
    body.querySelectorAll<HTMLButtonElement>('[data-restore]').forEach(btn => {
        btn.onclick = async () => {
            const r = JSON.parse(await Restore(btn.dataset.restore!));
            if (r.error) showToast(t('ui.restoreFailed', {err: r.error}), true);
            else {
                showToast(t('ui.restored', {path: r.restored}));
                rescan(); // 后台重扫，让主界面详情反映恢复的文件
            }
            refreshTrashList(); refreshTrashInfo();
        };
    });
    body.querySelectorAll<HTMLButtonElement>('[data-purge]').forEach(btn => {
        btn.onclick = async () => {
            const r = JSON.parse(await PurgeNow([btn.dataset.purge!]));
            if ((r.errors ?? []).length) showToast(t('ui.purgeFailed', {err: r.errors.join('; ')}), true);
            else { showToast(t('ui.purged')); rescan(); }
            refreshTrashList(); refreshTrashInfo();
        };
    });
}

// ---- update state ----
type UpdateProgress = { downloaded: number; total: number };
let updateState: 'idle' | 'downloading' | 'downloaded' = 'idle';
let lastUpdateResult: UpdateResult | null = null;

// 下载进度改用轮询（GetDownloadProgress）：macOS WKWebView 对跨桥事件不可靠，
// 事件推进度在 Mac 上完全不显示，轮询与 GetUpdateStatus 同一套路（已验证有效）。
let downloadPoll: number | null = null;
let lastShownPct = 0;
function stopDownloadPoll() {
    if (downloadPoll !== null) { window.clearInterval(downloadPoll); downloadPoll = null; }
}
function startDownloadPoll() {
    stopDownloadPoll();
    lastShownPct = 0;
    downloadPoll = window.setInterval(async () => {
        try {
            const raw = await GetDownloadProgress();
            if (!raw) return; // 已结束或未开始，由完成事件兜底
            const p = JSON.parse(raw) as UpdateProgress;
            if (updateState !== 'downloading') return;
            const bar = el('updBar') as HTMLElement;
            const pct = el('updPct');
            // 未开始或仍为 0%：保持初始 "0%"，不显示 0 字节、不切准备中文案（避免闪烁）
            if (!p.total || p.downloaded <= 0) return;
            const n = Math.min(100, Math.round((p.downloaded / p.total) * 100));
            if (n < 1) return; // 不足 1% 仍保持 0%
            // 单调防线：进度只进不退（双保险，Go 侧已有同样守卫）
            if (n < lastShownPct) return;
            lastShownPct = n;
            bar.style.width = n + '%';
            pct.textContent = `${n}%（${hb(p.downloaded)} / ${hb(p.total)}）`;
        } catch { /* 单次轮询失败忽略 */ }
    }, 200);
}

// 发现新版本：展示 当前/最新 版本与 [下载] [忽略该版本] [稍后]
function showUpdateAvailable(res: UpdateResult) {
    if (!res.updateAvailable) return;
    lastUpdateResult = res;
    updateState = 'idle';
    const body = el('updateBody');
    body.innerHTML = `
        <p class="update-versions">${t('upd.versions', {current: res.current, latest: res.latest})}</p>
        <p class="muted">${esc(t('upd.manualInstall'))}</p>
        <div class="update-actions">
            <button class="btn" id="updLater">${esc(t('upd.later'))}</button>
            <button class="btn" id="updIgnore">${esc(t('upd.ignore'))}</button>
            <button class="btn primary" id="updDownload">${esc(t('upd.download'))}</button>
        </div>`;
    el('updateModal').classList.remove('hidden');
    el('updLater').onclick = () => el('updateModal').classList.add('hidden');
    el('updIgnore').onclick = async () => {
        await IgnoreVersion(res.latest);
        el('updateModal').classList.add('hidden');
        showToast(t('upd.ignored', {version: res.latest}));
    };
    el('updDownload').onclick = () => startDownload();
}

// 开始下载：进度条 + 取消；进度由 update:progress 事件驱动
function startDownload() {
    updateState = 'downloading';
    const body = el('updateBody');
    body.innerHTML = `
        <p class="update-versions">${t('upd.downloading', {name: esc(lastUpdateResult?.assetName ?? '')})}</p>
        <div class="progress"><div id="updBar" class="progress-bar" style="width:0%"></div></div>
        <p id="updPct" class="muted">0%</p>
        <div class="update-actions"><button class="btn" id="updCancel">${esc(t('upd.cancel'))}</button></div>`;
    el('updCancel').onclick = () => { CancelDownload(); };
    DownloadUpdate().then(err => {
        if (err) { showToast(t('upd.startFailed', {err}), true); el('updateModal').classList.add('hidden'); return; }
        startDownloadPoll(); // 进度轮询（事件在 macOS 不可靠）
    });
}

// 下载完成（已通过校验）：[立即安装] [稍后]；压缩包产物额外展示当前二进制路径
function showUpdateDownloaded(d: UpdateDownloaded) {
    updateState = 'downloaded';
    stopDownloadPoll();
    const isArchive = d.installSource === 'tarball' || d.installSource === 'portable';
    const body = el('updateBody');
    body.innerHTML = `
        <p class="update-versions">${esc(t('upd.downloaded'))}</p>
        <p class="muted">${isArchive ? esc(t('upd.archiveSaved')) : esc(t('upd.installerSaved'))}<code>${esc(d.path)}</code></p>
        ${isArchive ? `<p class="muted">${esc(t('upd.binaryPath'))}<code>${esc(d.executablePath)}</code></p>` : ''}
        <div class="update-actions">
            <button class="btn" id="updLater">${esc(t('upd.later'))}</button>
            <button class="btn primary" id="updInstall">${esc(t('upd.installNow'))}</button>
        </div>`;
    el('updLater').onclick = () => el('updateModal').classList.add('hidden');
    // 点击后 Go 侧先打开安装包再退出应用（design D7）
    el('updInstall').onclick = () => { InstallUpdate(); };
}

// 校验失败：安全优先，不给安装入口，仅提供 Release 页面链接
function showUpdateVerifyFailed(err: string, releaseURL: string) {
    updateState = 'idle';
    const body = el('updateBody');
    body.innerHTML = `
        <p class="warn">${esc(t('upd.verifyFailed'))}</p>
        <p class="muted">${esc(err)}</p>
        <div class="update-actions">
            <button class="btn" id="updClose">${esc(t('upd.close'))}</button>
            <button class="btn" id="updRelease">${esc(t('upd.releasePage'))}</button>
        </div>`;
    el('updClose').onclick = () => el('updateModal').classList.add('hidden');
    el('updRelease').onclick = () => OpenURL(releaseURL || 'https://github.com/kevinjoy89/cli-analyzer/releases');
}

// 手动检查（Help 菜单「检查更新…」）：不受 24h 缓存限制
async function manualCheck() {
    let res: UpdateResult;
    try { res = JSON.parse(await CheckForUpdates()); }
    catch (e) { showToast(t('upd.checkFailed', {err: String(e)}), true); return; }
    if (res.error) { showToast(res.error, true); return; }
    if (res.updateAvailable) showUpdateAvailable(res);
    else showToast(t('upd.upToDate', {version: res.latest}));
}

// ---- uninstall ----
interface UninstallStartInfo { tool: string; installer?: string; blocked?: boolean; stale?: boolean; blockedReason?: string; officialCommand?: string; runnable?: boolean; footprint?: number; userBytes?: number; error?: string }
interface UninstallResidueItem { path: string; bytes: number; tier: string; kind: string }
interface UninstallStatus { running: boolean; done: boolean; output: string; error?: string }

let uninstallPoll: number | null = null;
function stopUninstallPoll() {
    if (uninstallPoll !== null) { window.clearInterval(uninstallPoll); uninstallPoll = null; }
}

// 详情页「卸载」→ 起始信息（官方命令/黑名单/占用）→ 确认弹窗
function startUninstall(toolName: string) {
    UninstallStart(toolName).then(raw => {
        const info = JSON.parse(raw) as UninstallStartInfo;
        if (info.stale) { showToast(info.error || '', true); rescan(); return; }
        if (info.error) { showToast(info.error, true); return; }
        if (info.blocked) { showToast(info.blockedReason || '', true); return; }
        showUninstallConfirm(info);
    }).catch(e => showToast(String(e), true));
}

function showUninstallConfirm(info: UninstallStartInfo) {
    const body = el('uninstallBody');
    // 标题承载「卸载 <tool>」，body 不再重复工具名
    el('uninstallTitle').textContent = t('un.guiUninstall') + ' ' + info.tool;
    // 复制按钮内联在命令后面（极简 SVG 图标），不放操作行
    const cmdHtml = info.officialCommand
        ? `<p class="muted un-cmdline">${esc(t('un.guiOfficialCmd'))}<br><code>${esc(info.officialCommand)}</code> <button id="unCopy" class="btn icon" title="${esc(t('un.guiCopyCmd'))}"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button></p>`
        : `<p class="muted">${esc(t('un.noOfficialCmd'))}</p>`;
    // 两行按钮：主文案 + 小字说明（语义：主路径 = 卸载并自动检测；跳过 = 直达残留检测）
    const runnableActions = info.runnable ? `
            <button class="btn" id="unSkip"><span class="btn-main">${esc(t('un.guiSkipMain'))}</span><span class="btn-sub">${esc(t('un.guiSkipSub'))}</span></button>
            <button class="btn primary" id="unRun"><span class="btn-main">${esc(t('un.guiRunMain'))}</span><span class="btn-sub">${esc(t('un.guiRunSub'))}</span></button>` : `
            <button class="btn primary" id="unResidue">${esc(t('un.guiResidueTitle'))}</button>`;
    body.innerHTML = `
        <p class="muted">占用 ${hb(info.footprint || 0)} · 用户数据 ${hb(info.userBytes || 0)}</p>
        ${cmdHtml}
        <p id="unOut" class="muted un-output"></p>
        <div class="update-actions">
            <button class="btn" id="unCancel">${esc(t('ui.cancel'))}</button>
            ${runnableActions}
        </div>`;
    el('uninstallModal').classList.remove('hidden');
    el('unCancel').onclick = () => { stopUninstallPoll(); el('uninstallModal').classList.add('hidden'); };
    // 条件渲染的按钮用 getElementById 可空查找（el() 对缺失元素会 throw，
    // 导致后续按钮事件全部丢失——曾报 "missing element #unRun"）
    const copyBtn = document.getElementById('unCopy');
    if (copyBtn) copyBtn.onclick = () => {
        navigator.clipboard?.writeText(info.officialCommand!).catch(() => {});
        showToast(t('un.copied'));
    };
    const runBtn = document.getElementById('unRun') as HTMLButtonElement | null;
    if (runBtn) runBtn.onclick = async () => {
        runBtn.disabled = true;
        const err = await UninstallRunOfficial();
        if (err) { showToast(err, true); runBtn.disabled = false; return; }
        startUninstallPoll();
    };
    const skipBtn = document.getElementById('unSkip');
    if (skipBtn) skipBtn.onclick = () => showUninstallResidue();
    const residueBtn = document.getElementById('unResidue');
    if (residueBtn) residueBtn.onclick = () => showUninstallResidue();
}

function startUninstallPoll() {
    stopUninstallPoll();
    const t0 = Date.now();
    uninstallPoll = window.setInterval(async () => {
        try {
            const st = JSON.parse(await GetUninstallStatus()) as UninstallStatus;
            const out = el('unOut');
            if (out) {
                // 有输出展示输出；无输出时显示进行中 + 已耗时（标准卸载可能需数十秒）
                out.textContent = st.output
                    ? st.output
                    : (st.running ? `${t('un.guiRunning')}（${Math.round((Date.now() - t0) / 1000)}s）` : '');
            }
            if (st.done) {
                stopUninstallPoll();
                if (st.error) showToast(t('un.runFailed') + ': ' + st.error, true);
                // 进入残留检测
                showUninstallResidue();
            }
        } catch { /* 单次轮询失败忽略 */ }
    }, 300);
}

// 残留检测 → 列表（全选默认、凭证标红）→ 移入回收站
async function showUninstallResidue() {
    let rr: UninstallResidueItem[] = [];
    try {
        const parsed = JSON.parse(await UninstallResidue());
        rr = Array.isArray(parsed) ? parsed : [];
    } catch { showToast(t('un.residueNone'), true); return; }
    if (!rr.length) { showToast(t('un.guiResidueNone')); return; }
    const body = el('uninstallBody');
    el('uninstallTitle').textContent = t('un.guiResidueTitle');
    body.innerHTML = `
        ${rr.map((r, i) => `
            <label class="pref-row check">
                <input type="checkbox" data-idx="${i}" checked/>
                <span class="${r.tier === 'user' ? 'un-credential' : ''}">${esc(r.path)} · ${hb(r.bytes)}${r.tier === 'user' ? ' · ' + esc(t('un.guiResidueCredential')) : ''}</span>
            </label>`).join('')}
        <div class="update-actions">
            <button class="btn" id="unCancel2">${esc(t('ui.cancel'))}</button>
            <button class="btn danger" id="unTrash">${esc(t('un.guiTrashConfirm'))}</button>
        </div>`;
    el('uninstallModal').classList.remove('hidden');
    el('unCancel2').onclick = () => el('uninstallModal').classList.add('hidden');
    el('unTrash').onclick = async () => {
        const boxes = body.querySelectorAll<HTMLInputElement>('input[type=checkbox]');
        const paths = rr.filter((_, i) => boxes[i].checked).map(r => r.path);
        if (!paths.length) { showToast(t('un.residueNone')); return; }
        const rep = JSON.parse(await UninstallTrashResidues(paths));
        el('uninstallModal').classList.add('hidden');
        const hasErr = (rep.errors ?? []).length > 0;
        showToast(t('un.guiResidueDone') + (hasErr ? '（' + t('un.runFailed') + '）' : ''), hasErr);
    };
}

// ---- preferences ----
async function openPrefs() {
    let cfg: TrashConfig;
    let rem: ReminderConfig;
    let upd: UpdateConfig;
    try { cfg = JSON.parse(await GetTrashConfig()); } catch { cfg = {retentionDays: 7, expireAction: 'system-trash', useTrash: true}; }
    try { rem = JSON.parse(await GetReminderConfig()); } catch { rem = {thresholdBytes: 5 * 1024 * 1024 * 1024}; }
    try { upd = JSON.parse(await GetUpdateConfig()); } catch { upd = {}; }
    let cfgLang = '';
    try { cfgLang = await GetLanguage(); } catch { /* auto */ }
    const threshGB = (rem.thresholdBytes || 5 * 1024 * 1024 * 1024) / (1024 * 1024 * 1024);
    const langOpts: Array<[string, string]> = [
        ['auto', t('ui.langAuto')],
        ['zh-CN', t('ui.langZhCN')],
        ['zh-TW', t('ui.langZhTW')],
        ['en', t('ui.langEn')],
    ];
    el('prefsBody').innerHTML = `
        <div class="pref-head">${esc(t('ui.prefsLanguage'))}</div>
        <label class="pref-row">${esc(t('ui.prefsLanguage'))}
            <select id="prefLanguage">
                ${langOpts.map(([v, label]) => `<option value="${v}" ${cfgLang === v ? 'selected' : ''}>${esc(label)}</option>`).join('')}
            </select>
        </label>
        <div class="pref-head">${esc(t('ui.prefsTrashHead'))}</div>
        <label class="pref-row">${esc(t('ui.prefsRetention'))}
            <input id="prefRetention" type="number" min="1" max="365" value="${cfg.retentionDays}"/>
        </label>
        <label class="pref-row">${esc(t('ui.prefsExpire'))}
            <select id="prefExpire">
                <option value="system-trash" ${cfg.expireAction === 'system-trash' ? 'selected' : ''}>${esc(t('ui.prefsExpireSystem'))}</option>
                <option value="permanent" ${cfg.expireAction === 'permanent' ? 'selected' : ''}>${esc(t('ui.prefsExpirePermanent'))}</option>
            </select>
        </label>
        <label class="pref-row check">
            <input id="prefUseTrash" type="checkbox" ${cfg.useTrash ? 'checked' : ''}/>
            <span>${esc(t('ui.prefsUseTrash'))}</span>
        </label>
        <div class="pref-head">${esc(t('ui.prefsTrendsHead'))}</div>
        <label class="pref-row">${esc(t('ui.prefsThreshold'))}
            <input id="prefThreshold" type="number" min="0" step="0.5" value="${threshGB.toFixed(1)}"/>
            <span>GB</span>
        </label>
        <div class="pref-head">${esc(t('ui.prefsUpdateHead'))}</div>
        <label class="pref-row check">
            <input id="prefCheckUpdates" type="checkbox" ${upd.checkUpdates !== false ? 'checked' : ''}/>
            <span>${esc(t('ui.prefsAutoCheck'))}</span>
        </label>`;
    el('prefsModal').classList.remove('hidden');
    el('prefsSave').onclick = async () => {
        const lang = (el('prefLanguage') as HTMLSelectElement).value;
        const next: TrashConfig = {
            retentionDays: Math.max(1, parseInt((el('prefRetention') as HTMLInputElement).value) || 7),
            expireAction: (el('prefExpire') as HTMLSelectElement).value,
            useTrash: (el('prefUseTrash') as HTMLInputElement).checked,
        };
        const rem2: ReminderConfig = {
            thresholdBytes: Math.round(parseFloat((el('prefThreshold') as HTMLInputElement).value || '5') * 1024 * 1024 * 1024),
        };
        // 合并回读的缓存字段，避免整体覆盖丢 lastCheckAt / ignoredVersion
        const upd2: UpdateConfig = {
            checkUpdates: (el('prefCheckUpdates') as HTMLInputElement).checked,
            lastCheckAt: upd.lastCheckAt,
            ignoredVersion: upd.ignoredVersion,
        };
        const err = await SetTrashConfig(JSON.stringify(next));
        const rerr = await SetReminderConfig(JSON.stringify(rem2));
        const uerr = await SetUpdateConfig(JSON.stringify(upd2));
        const lerr = await SetLanguagePreference(lang);
        // 语言切换：拉取新字典 → 即时重渲染 + 同步 Go 侧
        if (!lerr && lang !== activeLocale()) {
            const resolved = lang === 'auto' ? normalizeNavigator(navigator.language || '') : lang;
            try {
                const raw = JSON.parse(await GetTranslations(resolved));
                setDict(raw.locale, raw.dict);
                await SetLanguage(raw.locale);
            } catch { /* 保持当前字典 */ }
        }
        el('prefsModal').classList.add('hidden');
        if (err || rerr || uerr || lerr) showToast(t('ui.saveFailed', {err: (err || rerr || uerr || lerr)}), true);
        else {
            showToast(t('ui.saved'));
            refreshTrashInfo(); refreshReminder();
            applyI18n();
            // 原生菜单下次启动生效（仅 macOS 原生菜单；Windows/Linux 为 HTML 菜单即时切换）
            if (lang !== cfgLang && isMac) showToast(t('ui.menuRestartNote'));
        }
    };
    el('prefsCancel').onclick = () => el('prefsModal').classList.add('hidden');
}

// ---- usage trends ----
let reminderTools: Tool[] = [];

// 刷新待清理提醒：按阈值筛选超限工具，更新铃铛徽标与下拉面板
async function refreshReminder() {
    const bell = el('bellBtn');
    if (!result) { bell.classList.add('hidden'); reminderTools = []; return; }
    let thresh = 5 * 1024 * 1024 * 1024;
    try { thresh = (JSON.parse(await GetReminderConfig()) as ReminderConfig).thresholdBytes || thresh; } catch { /* 默认 */ }
    reminderTools = result.tools.filter(t => t.cleanableBytes > thresh);
    if (!reminderTools.length) {
        bell.classList.add('hidden');
        el('bellPanel').classList.remove('open');
        return;
    }
    bell.classList.remove('hidden');
    el('bellCount').textContent = String(reminderTools.length);
    if (el('bellPanel').classList.contains('open')) renderBellPanel();
}

// 渲染铃铛下拉面板：待清理工具列表，点击可快速跳转到对应工具
function renderBellPanel() {
    const panel = el('bellPanel');
    if (!reminderTools.length) {
        panel.innerHTML = '<div class="empty">' + esc(t('ui.noReminders')) + '</div>';
        return;
    }
    const shown = reminderTools.slice(0, 8);
    panel.innerHTML = shown.map(t =>
        `<button class="bell-item" data-tool="${esc(t.name)}">
            <span class="bell-name">${esc(t.name)}</span>
            <span class="size clean">${hb(t.cleanableBytes)}</span>
        </button>`).join('')
        + (reminderTools.length > 8 ? `<div class="bell-more">${esc(t('ui.bellMore', {n: reminderTools.length - 8}))}</div>` : '');
    panel.querySelectorAll<HTMLButtonElement>('.bell-item').forEach(btn => {
        btn.onclick = () => {
            const name = btn.dataset.tool!;
            selected = name; selectedCleanIds.clear(); expandedCleanIds.clear();
            renderToolList(); renderDetail();
            el('bellPanel').classList.remove('open');
        };
    });
}

function openTrends() {
    el('trendsModal').classList.remove('hidden');
    refreshTrends();
}

async function refreshTrends() {
    let tr: TrendsResult;
    try {
        const parsed = JSON.parse(await GetTrends(30));
        if (parsed.error) { el('trendChart').innerHTML = `<div class="warn">${esc(parsed.error)}</div>`; return; }
        tr = parsed;
    } catch (e) {
        el('trendChart').innerHTML = `<div class="warn">${esc(t('ui.trendsReadFailed', {err: String(e)}))}</div>`;
        return;
    }
    renderTrendChart(el('trendChart'), tr.points);
    renderGrowers(el('trendGrowers'), tr.topGrowers);
}

// 手写 SVG 折线，延续零依赖风格；历史不足两个点提示"数据积累中"
const CHART_W = 760, CHART_H = 240, CHART_PAD = 34;
function renderTrendChart(container: HTMLElement, points: Point[]) {
    const { footprint, cleanable, labels, max } = computeTrendPaths(points, CHART_W, CHART_H, CHART_PAD);
    if (!footprint) {
        container.innerHTML = '<div class="empty">' + esc(t('ui.trendsPending')) + '</div>';
        return;
    }
    container.innerHTML = `
        <div class="trend-legend">
            <span><i class="dot user"></i>${esc(t('ui.legendFootprint'))}</span>
            <span><i class="dot clean"></i>${esc(t('ui.legendCleanable'))}</span>
        </div>
        <svg viewBox="0 0 ${CHART_W} ${CHART_H}" preserveAspectRatio="xMidYMid meet" width="100%">
            <path d="${footprint}" fill="none" stroke="var(--user)" stroke-width="2" opacity="0.9"/>
            <path d="${cleanable}" fill="none" stroke="var(--clean)" stroke-width="2" opacity="0.9"/>
            ${labels}
            <text x="${CHART_PAD}" y="${CHART_PAD - 8}" font-size="11" fill="var(--muted)">${hb(max)}</text>
        </svg>`;
}

function renderGrowers(container: HTMLElement, growers: Grower[]) {
    if (!growers.length) {
        container.innerHTML = '<div class="muted">' + esc(t('ui.growersEmpty')) + '</div>';
        return;
    }
    container.innerHTML = growers.map((g, i) =>
        `<div class="grower-row"><span class="grower-rank">${i + 1}</span><span class="grower-name">${esc(g.tool)}</span><span class="size clean">+${hb(g.deltaBytes)}</span></div>`).join('');
}

// ---- about dialog ----
function openAbout() {
    el('aboutVersion').textContent = appVersion ? `v${appVersion}` : '';
    el('aboutModal').classList.remove('hidden');
}

// ---- in-app menu bar (Windows) ----
// 关闭所有打开的下拉菜单
function closeMenus() {
    document.querySelectorAll('.menu-btn.open').forEach(b => b.classList.remove('open'));
    document.querySelectorAll('.menu-pop.open').forEach(p => p.classList.remove('open'));
}

// 初始化 Windows 自绘菜单条：文件/帮助下拉 + 动作分发
function initMenuBar() {
    const bar = el('menuBar');
    if (!bar) return;
    bar.querySelectorAll<HTMLButtonElement>('.menu-btn').forEach(btn => {
        btn.onclick = (e) => {
            e.stopPropagation();
            const wasOpen = btn.classList.contains('open');
            closeMenus();
            if (!wasOpen) {
                btn.classList.add('open');
                const pop = bar.querySelector<HTMLElement>(`.menu-pop[data-pop="${btn.dataset.pop}"]`);
                if (pop) pop.classList.add('open');
            }
        };
    });
    bar.querySelectorAll<HTMLButtonElement>('.menu-opt').forEach(opt => {
        opt.onclick = () => {
            closeMenus();
            switch (opt.dataset.act) {
                case 'prefs': openPrefs(); break;
                case 'quit': Quit(); break;
                case 'about': openAbout(); break;
                case 'check-updates': manualCheck(); break;
                case 'github': OpenURL('https://github.com/kevinjoy89/cli-analyzer'); break;
                case 'issue': OpenURL('https://github.com/kevinjoy89/cli-analyzer/issues/new'); break;
            }
        };
    });
    document.addEventListener('click', closeMenus);
}

// ---- scan flow ----
async function rescan() {
    setScanning(true, t('ui.scanning'));
    try { await Scan(); } catch (e) { setScanning(false); showToast(t('ui.scanStartFailed', {err: String(e)}), true); }
}

async function init() {
    // Windows/Linux have a native titlebar above the content; mark the platform
    // so CSS can drop the top padding that only exists to clear the macOS
    // traffic lights / drag area.
    try {
        const env = await Environment();
        document.body.classList.add('platform-' + env.platform);
        if (env.platform === 'darwin') isMac = true;
    } catch { /* non-Wails context (plain browser preview) */ }

    // 尽早注册更新相关事件：启动自动检查可能命中缓存而瞬时完成（见 GetUpdateStatus
    // 注释），事件必须赶在检查结果之前就位，否则更新提示会丢失。
    EventsOn('update:available', (payload: unknown) => {
        try { showUpdateAvailable(JSON.parse(String(payload)) as UpdateResult); } catch { /* 忽略异常负载 */ }
    });
    // 进度不走事件（macOS WKWebView 不可靠），由 startDownloadPoll 轮询 GetDownloadProgress
    EventsOn('update:downloaded', (p: UpdateDownloaded) => showUpdateDownloaded(p));
    EventsOn('update:verify-failed', (p: {error: string; releaseURL: string}) => showUpdateVerifyFailed(p.error, p.releaseURL));
    EventsOn('update:cancelled', () => {
        stopDownloadPoll();
        updateState = 'idle';
        el('updateModal').classList.add('hidden');
        showToast(t('ui.downloadCancelled'));
    });
    EventsOn('update:error', (p: {error: string}) => {
        stopDownloadPoll();
        updateState = 'idle';
        el('updateModal').classList.add('hidden');
        showToast(t('ui.updateFailed', {err: p.error}), true);
    });

    // 初始化 i18n（拉取字典 + SetLanguage 握手），再渲染界面
    await initI18n();
    applyI18n();

    initMenuBar();

    // Ctrl+, opens preferences (Windows in-app menu bar; macOS uses the native menu)
    document.addEventListener('keydown', (e) => {
        if (document.body.classList.contains('platform-windows') && (e.ctrlKey || e.metaKey) && e.key === ',') {
            e.preventDefault();
            openPrefs();
        }
    });

    el('rescanBtn').onclick = rescan;
    el('filter').addEventListener('input', (e) => { filterText = (e.target as HTMLInputElement).value; renderToolList(); });

    // built-in trash panel
    el('trashBtn').onclick = openTrashPanel;
    el('trashClose').onclick = () => el('trashModal').classList.add('hidden');
    el('trashPurgeAll').onclick = async () => {
        const ids = trashItems.map(it => it.id);
        if (!ids.length) return;
        const r = JSON.parse(await PurgeNow(ids));
        showToast((r.errors ?? []).length ? t('ui.emptyTrashFailed', {err: r.errors.join('; ')}) : t('ui.emptyTrashDone', {n: r.deleted.length}), (r.errors ?? []).length > 0);
        if (r.deleted?.length) rescan();
        refreshTrashList(); refreshTrashInfo();
    };

    // preferences panel (opened from the native menu)
    EventsOn('open-prefs', () => openPrefs());

    // usage trends panel
    el('trendsBtn').onclick = openTrends;
    el('trendsClose').onclick = () => el('trendsModal').classList.add('hidden');

    // notification bell: toggle dropdown panel, position it at the click site
    el('bellBtn').onclick = () => {
        const panel = el('bellPanel');
        if (panel.classList.contains('open')) { panel.classList.remove('open'); return; }
        renderBellPanel();
        panel.style.cssText = ''; // 用 CSS 定位（header 右下角），清除此前可能残留的内联定位
        panel.classList.add('open');
    };
    document.addEventListener('click', (e) => {
        const panel = el('bellPanel');
        if (panel.classList.contains('open') && !(e.target as HTMLElement).closest('#bellPanel, #bellBtn')) {
            panel.classList.remove('open');
        }
    });

    // about dialog (opened from the native Help menu)
    EventsOn('open-about', openAbout);
    el('aboutClose').onclick = () => el('aboutModal').classList.add('hidden');
    el('aboutLink').onclick = (e) => { e.preventDefault(); OpenURL('https://github.com/kevinjoy89/cli-analyzer'); };

    // update flow: native Help 菜单「检查更新…」触发手动检查
    EventsOn('check-updates', manualCheck);

    // 启动自动检查可能已完成（含缓存命中，瞬时返回）：事件已提前注册（见 init 顶部），
    // 这里再主动拉取一次结果，兜住事件与监听注册之间的竞态（问题：打开软件不弹更新提示）。
    try {
        const raw = await GetUpdateStatus();
        if (raw) {
            const res = JSON.parse(raw) as UpdateResult;
            if (res.updateAvailable && !res.error) showUpdateAvailable(res);
        }
    } catch { /* 检查未完成或解析失败，稍后事件兜底 */ }

    // theme toggle: system -> light -> dark -> system
    applyTheme('system');
    el('themeBtn').onclick = () => {
        const next: ThemeMode = themeMode === 'system' ? 'light' : themeMode === 'light' ? 'dark' : 'system';
        applyTheme(next);
    };
    if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
            if (themeMode === 'system') applyTheme('system');
        });
    };

    // app version in footer
    try {
        appVersion = await GetVersion();
        el('appVersion').textContent = `v${appVersion}`;
    } catch (e) { /* ignore */ }

    EventsOn('scan:done', (payload: unknown) => {
        setScanning(false);
        try {
            if (payload && typeof payload === 'object' && 'error' in (payload as object)) {
                showToast(t('ui.scanFailed', {err: JSON.stringify(payload)}), true);
                return;
            }
            const parsed = typeof payload === 'string' ? JSON.parse(payload) as ScanResult : null;
            result = parsed;
            if (parsed && (!selected || !parsed.tools.some(t => t.name === selected))) {
                selected = parsed.tools[0]?.name ?? null;
            }
            renderSummary(); renderToolList(); renderDetail(); refreshTrashInfo(); refreshReminder();
        } catch (err) {
            showToast(t('ui.scanParseFailed'), true);
        }
    });

    try {
        const raw = await GetLastResult();
        if (raw) { result = JSON.parse(raw); renderSummary(); renderToolList(); renderDetail(); }
    } catch (e) {
        console.error('load cache failed', e);
    }

    refreshTrashInfo();
    refreshReminder();
}

init();

// 渲染模块：全部 DOM 渲染函数（列表/详情/未认领/铃铛/趋势）。
// 依赖 dom + state + lib；对 flows 的反向调用（卸载/确认弹窗）走回调注册，
// 由 main.ts 在 init 时注入，避免循环依赖。
import {el, esc} from './dom';
import {expandedCleanIds, filterText, orphanSel, panelView, reminderTools, result, selected, selectedCleanIds, sortDir, sortKey} from './state';
import {setPanelView, setSelected, setSortDir, setSortKey} from './state';
import type {SortKey} from './state';
import {hb} from './lib/format';
import {fmtTime, t} from './lib/i18n';
import {kindLabel, orphanRootLabel} from './lib/labels';
import {computeTrendPaths} from './lib/trends';
import type {Cleanable, DataDir, Grower, Point, Tool} from './lib/types';
import {OpenURL, UninstallBlocked} from '../wailsjs/go/gui/ScannerService';

// ---- 回调注册（main.ts init 注入，避免 render → flows 循环依赖）----
let uninstallHandler: ((name: string) => void) | null = null;
let upgradeHandler: ((name: string) => void) | null = null;
let orphanConfirmHandler: ((paths: string[]) => void) | null = null;
let confirmHandler: ((items: PickItem[]) => void) | null = null;
export function setUninstallHandler(fn: (name: string) => void) { uninstallHandler = fn; }
export function setUpgradeHandler(fn: (name: string) => void) { upgradeHandler = fn; }
export function setOrphanConfirmHandler(fn: (paths: string[]) => void) { orphanConfirmHandler = fn; }
export function setConfirmHandler(fn: (items: PickItem[]) => void) { confirmHandler = fn; }

// ---- summary ----
export function renderSummary() {
    if (!result) return;
    el('sumFootprint').textContent = hb(result.totals.footprintBytes);
    el('sumCleanable').textContent = hb(result.totals.cleanableBytes);
    el('sumUser').textContent = hb(result.totals.userBytes);
    el('sumTools').textContent = String(result.tools.length);
    // 未认领数据统计卡：常驻顶部，静态展示占用大小（琥珀色）
    const orphans = result.unattributed ?? [];
    const oTotal = orphans.reduce((a, o) => a + (o.bytes || 0), 0);
    const ost = el('orphanStat');
    if (orphans.length) {
        ost.classList.remove('hidden');
        el('sumOrphan').textContent = hb(oTotal);
    } else {
        ost.classList.add('hidden');
    }
    el('lastScan').textContent = result.scannedAt ? t('ui.lastScan', {time: fmtTime(result.scannedAt)}) : '';
    const status = el('statusInfo');
    status.innerHTML = '';
    // data-role 标记各字段：扫描进行中 setScanning 会移除 scanTime/walkErrors
    // （旧结果数据），保留 platform（静态系统信息）
    const parts: Array<{role: string; labelKey: string; value: string}> = [
        {role: 'scanTime', labelKey: 'ui.scanTime', value: result.scanTimeMs > 0 ? `${(result.scanTimeMs / 1000).toFixed(1)} s` : t('ui.scanCache')},
        {role: 'walkErrors', labelKey: 'ui.walkErrors', value: String(result.walkErrors)},
        {role: 'platform', labelKey: 'ui.platform', value: result.platform},
    ];
    for (const {role, labelKey, value} of parts) {
        const span = document.createElement('span');
        span.dataset.role = role;
        span.textContent = `${t(labelKey)}: ${value}`;
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

export function renderToolList() {
    renderPanelTabs();
    const list = el('toolList');
    list.innerHTML = '';
    if (!result) {
        list.innerHTML = '<div class="empty">' + esc(t('ui.noData')) + '</div>';
        return;
    }
    // 未认领数据视图：数据清空后自动回到工具视图
    if (panelView === 'orphans' && !(result.unattributed ?? []).length) setPanelView('tools');
    if (panelView === 'orphans') {
        renderOrphanView(list);
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
                setSortDir(sortDir === -1 ? 1 : -1);
            } else {
                setSortKey(k);
                setSortDir((k === 'name') ? 1 : -1);
            }
            renderToolList();
        };
    });
    // 未认领数据有独立标签页，不再混排在工具列表里。
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
        row.onclick = () => { setSelected(tool.name); selectedCleanIds.clear(); expandedCleanIds.clear(); renderToolList(); renderDetail(); };
        list.appendChild(row);
    }
}

// ---- 未认领数据（标签页视图）----
// 面板顶部 tab：工具 | 未认领数据。未认领数据 = 数据根下未被任何工具认领
// 的目录（USER 级），唯一处置是移入内置回收站（可恢复），绝不永久删除。

// 极简垃圾桶图标（线性描边，与卸载复制按钮风格一致）；状态栏回收站按钮与
// 未认领数据行共用同一标识，避免 emoji 风格不统一且视觉过重。
const TRASH_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>';
// 恢复（撤销回退箭头）图标，回收站行内操作按钮用
const RESTORE_ICON = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>';

// 类型 → 标签色（方形色块）：未认领数据=蓝、缓存=绿、日志=琥珀，其余灰
const KIND_TONE: Record<string, string> = {
    'cache': 'green',
    'logs': 'amber',
    'data': 'blue',
    'download': 'blue',
    'old-version': 'gray',
    'backup': 'gray',
    'toolchain': 'gray',
    'config': 'gray',
    'state': 'gray',
    'install': 'gray',
};

// 数据根 / 类型标签的本地化映射见 lib/labels.ts（labels.test.ts 守护两端不漂移）

// 渲染面板顶部的标签控件；无未认领数据时不显示（工具视图即全部内容）
function renderPanelTabs() {
    const wrap = el('panelTabs');
    const orphans = result?.unattributed ?? [];
    if (!orphans.length) {
        wrap.innerHTML = '';
        return;
    }
    const tab = (v: 'tools' | 'orphans', label: string) =>
        `<button class="tab${panelView === v ? ' active' : ''}" data-v="${v}">${esc(label)}</button>`;
    wrap.innerHTML = tab('tools', t('ui.tabTools')) + tab('orphans', t('ui.tabOrphans'));
    wrap.querySelectorAll<HTMLButtonElement>('button.tab').forEach(btn => {
        btn.onclick = () => { setPanelView(btn.dataset.v as 'tools' | 'orphans'); renderToolList(); };
    });
}

function filteredOrphans(): DataDir[] {
    return (result?.unattributed ?? []).filter(o => !filterText || o.path.toLowerCase().includes(filterText.toLowerCase()));
}

// 未认领数据视图：工具栏（计数/全选/批量移入）+ 按数据根分组列表
export function renderOrphanView(list: HTMLElement) {
    const orphans = filteredOrphans();
    if (!orphans.length) {
        list.innerHTML = '<div class="empty">' + esc(t(filterText ? 'ui.orphanNoneMatch' : 'ui.orphanEmpty')) + '</div>';
        return;
    }
    const total = orphans.reduce((a, o) => a + (o.bytes || 0), 0);
    // 按数据根分组，组内按大小降序；组按总大小降序
    const byRoot = new Map<string, DataDir[]>();
    for (const o of orphans) {
        const k = o.root || '';
        if (!byRoot.has(k)) byRoot.set(k, []);
        byRoot.get(k)!.push(o);
    }
    const groups = [...byRoot.entries()].map(([root, items]) => {
        items.sort((a, b) => (b.bytes || 0) - (a.bytes || 0));
        const gTotal = items.reduce((s, o) => s + (o.bytes || 0), 0);
        // 组标题展示数据根的真实路径（~/.config 等），而非 xdg-config 这类
        // 技术规范名；roots 缺失时回退为空（仅显示分类标签）
        const base = result?.roots?.[root]?.[0] ?? '';
        return {root, items, total: gTotal, base};
    }).sort((a, b) => b.total - a.total);

    const selCount = orphans.filter(o => orphanSel.has(o.path)).length;
    list.innerHTML = `
        <div class="otool">
            <span class="ot">${esc(t('ui.orphanCount', {n: orphans.length, size: hb(total)}))}</span>
            <label class="selall"><input type="checkbox" id="orphanSelAll" ${selCount === orphans.length ? 'checked' : ''}/> ${esc(t('ui.orphanSelectAll'))}</label>
            <button class="btn mini danger" id="orphanTrashSel" ${selCount ? '' : 'disabled'}>${esc(t('ui.orphanMoveToTrash', {n: selCount}))}</button>
        </div>
        ${groups.map(g => `
            <div class="ogroup">
                <span class="og-name">${esc(orphanRootLabel(g.root))}</span>
                ${g.base ? `<span class="og-root" title="${esc(g.base)}">${esc(g.base)}</span>` : ''}
                <span class="og-size">${hb(g.total)} · ${g.items.length}</span>
            </div>
            ${g.items.map(o => `
                <div class="oitem">
                    <input type="checkbox" data-orphan="${esc(o.path)}" ${orphanSel.has(o.path) ? 'checked' : ''}/>
                    <span class="op" title="${esc(o.path)}">${esc(o.path)}</span>
                    <span class="os">${hb(o.bytes || 0)}</span>
                    <button class="otrash" data-trash="${esc(o.path)}" title="${esc(t('ui.orphanTrash'))}">${TRASH_ICON}</button>
                </div>`).join('')}
        `).join('')}`;

    // 同步批量按钮与全选状态（不整体重渲染，避免勾选闪烁）
    const syncBar = () => {
        const n = filteredOrphans().filter(o => orphanSel.has(o.path)).length;
        const btn = el('orphanTrashSel') as HTMLButtonElement;
        btn.textContent = t('ui.orphanMoveToTrash', {n});
        btn.disabled = n === 0;
        (el('orphanSelAll') as HTMLInputElement).checked = n === orphans.length;
    };
    el('orphanSelAll').onchange = (e) => {
        const on = (e.target as HTMLInputElement).checked;
        if (on) for (const o of orphans) orphanSel.add(o.path);
        else for (const o of orphans) orphanSel.delete(o.path);
        syncBar();
    };
    list.querySelectorAll<HTMLInputElement>('input[data-orphan]').forEach(cb => {
        cb.onchange = () => {
            const p = cb.dataset.orphan!;
            if (cb.checked) orphanSel.add(p); else orphanSel.delete(p);
            syncBar();
        };
    });
    // 批量按钮与单条按钮都先走二次确认（showOrphanConfirm）
    el('orphanTrashSel').onclick = () => orphanConfirmHandler?.(orphans.filter(o => orphanSel.has(o.path)).map(o => o.path));
    list.querySelectorAll<HTMLButtonElement>('button[data-trash]').forEach(btn => {
        btn.onclick = () => orphanConfirmHandler?.([btn.dataset.trash!]);
    });
}

// ---- detail ----
export function renderDetail() {
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

    // 安装根（Kind install）仅展示，不列为可处置项：删除安装根 = 卸载工具，
    // 走独立的卸载流程（详情页「卸载」按钮）。
    const installDirs = tool.dataDirs.filter(d => d.kind === 'install');
    const installHtml = installDirs.length
        ? `<div class="detail-list">${installDirs.map(d =>
            `<div class="detail-item"><span class="badge ${d.tier}">${d.tier}</span><span class="path" title="${esc(d.path)}">${esc(d.path)}</span><span class="size ${d.tier}">${hb(d.bytes)}</span><span class="keep">${esc(kindLabel(d.kind))}</span></div>`).join('')}</div>`
        : '';

    // 全部归因目录（安装根除外）都是可处置项：kind/tier 是信息标签，
    // 勾选即代表用户决策，应用不再替用户裁决"能不能删"。
    const cleanables = tool.cleanables;
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
                    <span class="badge safe">${esc(kindLabel(c.kind))}</span>
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
        ${installHtml ? `<div class="detail-section"><h4>${esc(t('ui.sectionInstallRoot'))}</h4>${installHtml}</div>` : ''}
        <div class="detail-section"><h4>${esc(t('ui.sectionSafe'))}</h4>${cleanHtml}</div>
        <div class="detail-actions">
            <button id="cleanBtn" class="btn danger">${esc(t('ui.cleanSelected'))}</button>
            <button id="uninstallBtn" class="btn" title="${esc(t('un.guiUninstall'))}">${esc(t('un.guiUninstall'))}</button>
            <button id="upgradeBtn" class="btn" title="${esc(t('up.guiCheckUpdate'))}">${esc(t('up.guiCheckUpdate'))}</button>
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
        confirmHandler?.(selectedItems(tool));
    };
    el<HTMLButtonElement>('uninstallBtn').onclick = () => uninstallHandler?.(tool.name);
    el<HTMLButtonElement>('upgradeBtn').onclick = () => upgradeHandler?.(tool.name);
    // 黑名单工具禁用卸载按钮（UninstallBlocked 来自服务层）
    UninstallBlocked(tool.name).then(raw => {
        try {
            const b = JSON.parse(raw) as {blocked: boolean};
            el<HTMLButtonElement>('uninstallBtn').disabled = !!b.blocked;
        } catch { /* 忽略 */ }
    });
}

// One level of size breakdown for a cleanable (its direct children), rendered
// as indented rows below the checkbox row. Children with an id are individually
// selectable (0 字节子项也可勾选，清理空文件/空目录）；当父项（整目录）被
// 勾选时子项禁用。Capped to keep wide dirs readable.
const SUB_CAP = 20;
function subRows(c: Cleanable, parentSelected: boolean): string {
    const sub = c.sub ?? [];
    if (!sub.length) return '';
    const shown = sub.slice(0, SUB_CAP);
    const more = sub.length - shown.length;
    // 分隔符随平台：Windows 路径为反斜杠（Go 端 filepath.Join），unix 为正斜杠；
    // 固定用 '/' 拼接会让反斜杠路径的子项退化为显示完整路径。
    const sep = c.path.includes('\\') ? '\\' : '/';
    const base = c.path.endsWith(sep) ? c.path : c.path + sep;
    const rows = shown.map(s => {
        const rel = s.path.startsWith(base) ? s.path.slice(base.length) : s.path;
        const selectable = !!s.id;
        if (!selectable) {
            return `<div class="detail-item sub-row">
                <span class="sub-path" title="${esc(s.path)}">${esc(rel)}</span>
                <span class="size muted">${hb(s.bytes)}</span>
            </div>`;
        }
        const checked = selectedCleanIds.has(s.id) ? 'checked' : '';
        const disabled = parentSelected ? 'disabled' : '';
        // 子项类型与父项不同时（如 ~/.npm/_logs → 日志）标注精确类型；
        // logs 类（_logs/*.log 是缓存目录的组成部分）不额外贴「日志」标签，
        // 避免在安全清理项里看起来像独立类别
        const kindTag = s.kind && s.kind !== c.kind && s.kind !== 'logs'
            ? `<span class="sub-kind">${esc(kindLabel(s.kind))}</span>` : '';
        return `<div class="detail-item sub-row">
            <input type="checkbox" class="sub-cb" data-id="${esc(s.id)}" data-parent="${esc(c.id)}" ${checked} ${disabled}/>
            <span class="sub-path" title="${esc(s.path)}">${esc(rel)}</span>
            ${kindTag}
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
export interface PickItem { id: string; path: string; bytes: number; kind: string }
export function selectedItems(t: Tool): PickItem[] {
    const out: PickItem[] = [];
    for (const c of t.cleanables) {
        if (selectedCleanIds.has(c.id)) {
            out.push({id: c.id, path: c.path, bytes: c.bytes, kind: c.kind});
            continue;
        }
        for (const s of c.sub ?? []) {
            // 子项用精确类型（_logs → 日志），未携带时继承父项
            if (s.id && selectedCleanIds.has(s.id)) out.push({id: s.id, path: s.path, bytes: s.bytes, kind: s.kind || c.kind});
        }
    }
    return out;
}

// ---- bell panel ----
// 渲染铃铛下拉面板：待清理工具列表，点击可快速跳转到对应工具
export function renderBellPanel() {
    const panel = el('bellPanel');
    if (!reminderTools.length) {
        panel.innerHTML = '<div class="empty">' + esc(t('ui.noReminders')) + '</div>';
        return;
    }
    const shown = reminderTools.slice(0, 8);
    panel.innerHTML = shown.map(tool =>
        `<button class="bell-item" data-tool="${esc(tool.name)}">
            <span class="bell-name">${esc(tool.name)}</span>
            <span class="size clean">${hb(tool.cleanableBytes)}</span>
        </button>`).join('')
        + (reminderTools.length > 8 ? `<div class="bell-more">${esc(t('ui.bellMore', {n: reminderTools.length - 8}))}</div>` : '');
    panel.querySelectorAll<HTMLButtonElement>('.bell-item').forEach(btn => {
        btn.onclick = () => {
            const name = btn.dataset.tool!;
            setSelected(name); selectedCleanIds.clear(); expandedCleanIds.clear();
            renderToolList(); renderDetail();
            panel.classList.remove('open');
        };
    });
}

// ---- usage trends ----
// 手写 SVG 折线，延续零依赖风格；历史不足两个点提示"数据积累中"
const CHART_W = 760, CHART_H = 240, CHART_PAD = 34;
export function renderTrendChart(container: HTMLElement, points: Point[]) {
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

export function renderGrowers(container: HTMLElement, growers: Grower[]) {
    if (!growers.length) {
        container.innerHTML = '<div class="muted">' + esc(t('ui.growersEmpty')) + '</div>';
        return;
    }
    container.innerHTML = growers.map((g, i) =>
        `<div class="grower-row"><span class="grower-rank">${i + 1}</span><span class="grower-name">${esc(g.tool)}</span><span class="size clean">+${hb(g.deltaBytes)}</span></div>`).join('');
}

// 供 flows/main 使用：未认领数据行内确认入口的别名（避免重名冲突）
export {RESTORE_ICON, TRASH_ICON, KIND_TONE};

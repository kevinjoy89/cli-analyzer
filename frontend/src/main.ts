import './style.css';

import {Environment, EventsOn} from '../wailsjs/runtime/runtime';
import {GetLastResult, GetLanguage, GetTranslations, GetUpdateStatus, GetVersion, OpenURL, PurgeNow, ScanIfChanged, SetLanguage, SetTheme} from '../wailsjs/go/gui/ScannerService';
import {el, showToast} from './dom';
import * as state from './state';
import {activeLocale, normalizeNavigator, setDict, t} from './lib/i18n';
import {initMenuBar} from './menu';
import * as flows from './flows';
import * as render from './render';
import {rescan} from './flows';

// ---- theme ----
function systemDark(): boolean {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme(mode: state.ThemeMode) {
    state.setThemeMode(mode);
    const resolved = mode === 'system' ? (systemDark() ? 'dark' : 'light') : mode;
    document.documentElement.setAttribute('data-theme', resolved);
    const meta = state.THEME_META[mode];
    el('themeIcon').textContent = meta.icon;
    el('themeBtn').title = t('ui.themeBtnTitle', {label: t(meta.labelKey)});
    // 同步 Windows 原生标题栏/菜单栏主题（macOS/Linux 由系统与 CSS 处理）
    SetTheme(mode);
}

// ---- i18n ----
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
    applyTheme(state.themeMode);
    render.renderSummary(); render.renderToolList();
    render.renderDetail(); // 无条件重渲染：无选中时的空态也是 JS 生成的，必须覆盖
    flows.refreshTrashInfo(); flows.refreshReminder();
    if (!el('prefsModal').classList.contains('hidden')) flows.openPrefs();
    if (!el('updateModal').classList.contains('hidden') && state.updateState === 'idle' && state.lastUpdateResult?.updateAvailable) {
        flows.showUpdateAvailable(state.lastUpdateResult);
    }
}

// ---- init ----
async function init() {
    // Windows/Linux have a native titlebar above the content; mark the platform
    // so CSS can drop the top padding that only exists to clear the macOS
    // traffic lights / drag area.
    try {
        const env = await Environment();
        document.body.classList.add('platform-' + env.platform);
        if (env.platform === 'darwin') state.setIsMac(true);
    } catch { /* non-Wails context (plain browser preview) */ }

    // 尽早注册更新相关事件：启动自动检查可能命中缓存而瞬时完成（见 GetUpdateStatus
    // 注释），事件必须赶在检查结果之前就位，否则更新提示会丢失。
    EventsOn('update:available', (payload: unknown) => {
        try { flows.showUpdateAvailable(JSON.parse(String(payload)) as state.UpdateResult); } catch { /* 忽略异常负载 */ }
    });
    // 进度不走事件（macOS WKWebView 不可靠），由 startDownloadPoll 轮询 GetDownloadProgress
    EventsOn('update:downloaded', (p: state.UpdateDownloaded) => flows.showUpdateDownloaded(p));
    EventsOn('update:verify-failed', (p: {error: string; releaseURL: string}) => flows.showUpdateVerifyFailed(p.error, p.releaseURL));
    EventsOn('update:cancelled', () => {
        flows.stopDownloadPoll();
        state.setUpdateState('idle');
        el('updateModal').classList.add('hidden');
        showToast(t('ui.downloadCancelled'));
    });
    EventsOn('update:error', (p: {error: string; releaseURL?: string}) => {
        flows.stopDownloadPoll();
        state.setUpdateState('idle');
        // 失败面板保留弹窗：展示错误 + 重试 + Release 页跳转（不自动关闭）
        flows.showUpdateFailed(p.error, p.releaseURL ?? '');
    });

    // 初始化 i18n（拉取字典 + SetLanguage 握手），再渲染界面
    await initI18n();
    applyI18n();

    // 回调注入：render → flows 的反向调用（避免循环依赖）
    render.setUninstallHandler((n) => flows.startUninstall(n));
    render.setOrphanConfirmHandler((paths) => flows.showOrphanConfirm(paths));
    render.setConfirmHandler((items: render.PickItem[]) => flows.showConfirmModal(items));
    flows.setApplyI18n(applyI18n);

    initMenuBar();

    // Ctrl+, opens preferences (Windows in-app menu bar; macOS uses the native menu)
    document.addEventListener('keydown', (e) => {
        if (document.body.classList.contains('platform-windows') && (e.ctrlKey || e.metaKey) && e.key === ',') {
            e.preventDefault();
            flows.openPrefs();
        }
    });

    el('rescanBtn').onclick = rescan;
    el('filter').addEventListener('input', (e) => { state.setFilterText((e.target as HTMLInputElement).value); render.renderToolList(); });

    // built-in trash panel
    el('trashBtn').onclick = flows.openTrashPanel;
    el('trashClose').onclick = () => el('trashModal').classList.add('hidden');
    el('trashPurgeAll').onclick = async () => {
        const ids = state.trashItems.map(it => it.id);
        if (!ids.length) return;
        // 永久删除前必须确认（自 0.3.8 起 PurgeNow 为不可恢复的永久删除）
        const ok = await flows.confirmDialog({
            title: t('ui.emptyTrash'),
            message: t('ui.purgeConfirm', {n: ids.length}),
            confirmText: t('ui.emptyTrash'),
        });
        if (!ok) return;
        const r = JSON.parse(await PurgeNow(ids));
        showToast((r.errors ?? []).length ? t('ui.emptyTrashFailed', {err: r.errors.join('; ')}) : t('ui.emptyTrashDone', {n: r.deleted.length}), (r.errors ?? []).length > 0);
        if (r.deleted?.length) rescan();
        flows.refreshTrashList(); flows.refreshTrashInfo();
    };

    // preferences panel (opened from the native menu)
    EventsOn('open-prefs', () => flows.openPrefs());

    // usage trends panel
    el('trendsBtn').onclick = flows.openTrends;
    el('trendsClose').onclick = () => el('trendsModal').classList.add('hidden');

    // notification bell: toggle dropdown panel, position it at the click site
    el('bellBtn').onclick = () => {
        const panel = el('bellPanel');
        if (panel.classList.contains('open')) { panel.classList.remove('open'); return; }
        render.renderBellPanel();
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
    EventsOn('open-about', () => flows.openAbout());
    el('aboutClose').onclick = () => el('aboutModal').classList.add('hidden');
    el('aboutLink').onclick = (e) => { e.preventDefault(); OpenURL('https://github.com/kevinjoy89/cli-analyzer'); };

    // update flow: native Help 菜单「检查更新…」触发手动检查
    EventsOn('check-updates', () => flows.manualCheck());

    // 启动自动检查可能已完成（含缓存命中，瞬时返回）：事件已提前注册（见 init 顶部），
    // 这里再主动拉取一次结果，兜住事件与监听注册之间的竞态（问题：打开软件不弹更新提示）。
    try {
        const raw = await GetUpdateStatus();
        if (raw) {
            const res = JSON.parse(raw) as state.UpdateResult;
            if (res.updateAvailable && !res.error) flows.showUpdateAvailable(res);
        }
    } catch { /* 检查未完成或解析失败，稍后事件兜底 */ }

    // theme toggle: system -> light -> dark -> system
    applyTheme('system');
    el('themeBtn').onclick = () => {
        const next: state.ThemeMode = state.themeMode === 'system' ? 'light' : state.themeMode === 'light' ? 'dark' : 'system';
        applyTheme(next);
    };
    if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
            if (state.themeMode === 'system') applyTheme('system');
        });
    };

    // app version in footer
    try {
        state.setAppVersion(await GetVersion());
        el('appVersion').textContent = `v${state.appVersion}`;
    } catch (e) { /* ignore */ }

    EventsOn('scan:done', (payload: unknown) => {
        flows.setScanning(false);
        try {
            if (payload && typeof payload === 'object' && 'error' in (payload as object)) {
                showToast(t('ui.scanFailed', {err: JSON.stringify(payload)}), true);
                return;
            }
            const parsed = typeof payload === 'string' ? JSON.parse(payload) as state.ScanResult : null;
            state.setResult(parsed);
            state.orphanSel.clear(); // 新扫描结果：旧勾选失效
            if (parsed && (!state.selected || !parsed.tools.some(tool => tool.name === state.selected))) {
                state.setSelected(parsed.tools[0]?.name ?? null);
            }
            // 存在版本未知的工具 → 后台探测进行中（版本列显示 …）
            state.setProbing(!!parsed && parsed.tools.some(tool => !tool.version && tool.binaries?.length));
            render.renderSummary(); render.renderToolList(); render.renderDetail(); flows.refreshTrashInfo(); flows.refreshReminder();
        } catch (err) {
            showToast(t('ui.scanParseFailed'), true);
        }
    });

    // 健康探测完成：用探测后的结果重渲染版本列
    EventsOn('probe:done', (payload: unknown) => {
        state.setProbing(false);
        try {
            if (typeof payload === 'string') {
                state.setResult(JSON.parse(payload) as state.ScanResult);
                render.renderSummary(); render.renderToolList(); render.renderDetail();
            }
        } catch { /* 忽略 */ }
    });

    try {
        const raw = await GetLastResult();
        if (raw) { state.setResult(JSON.parse(raw)); render.renderSummary(); render.renderToolList(); render.renderDetail(); }
    } catch (e) {
        console.error('load cache failed', e);
    }

    flows.refreshTrashInfo();
    flows.refreshReminder();

    // 每次打开软件都触发一次异步扫描：指纹未变化时直接复用缓存（秒开、
    // 无全量 IO），数据变化时自动全量。手动"重新扫描"按钮仍走 rescan()
    // （强制全量，main.ts 中 rescanBtn 的 onclick 不变）。
    // 先进入 busy 状态：与手动重扫一致的按钮禁用/转圈（数据变化触发全量
    // 扫描时用户有反馈，也防止扫描期间触发并发扫描）；scan:done 处理器首行
    // setScanning(false) 复位，错误路径 catch 同样复位。
    flows.setScanning(true, t('ui.scanning'));
    try {
        await ScanIfChanged();
    } catch (e) {
        flows.setScanning(false);
        showToast(t('ui.scanStartFailed', {err: String(e)}), true);
    }
}

init();

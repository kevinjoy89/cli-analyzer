// 业务流程模块：更新下载、卸载、回收站、首选项、趋势、确认弹窗、扫描触发。
// 依赖 dom + state + render + lib + wailsjs；render 对本文的回调在 main.ts 注入。
import { el, esc, showToast } from "./dom";
import {
    appVersion,
    downloadPoll,
    isMac,
    lastShownPct,
    lastUpdateResult,
    orphanSel,
    reminderTools,
    result,
    selected,
    selectedCleanIds,
    trashItems,
    updateState,
    uninstallPoll,
    upgradePoll,
} from "./state";
import {
    setDownloadPoll,
    setLastShownPct,
    setLastUpdateResult,
    setReminderTools,
    setResult,
    setTrashItems,
    setUpdateState,
    setUninstallPoll,
    setUpgradePoll,
} from "./state";
import * as render from "./render";
import { KIND_TONE, RESTORE_ICON, TRASH_ICON } from "./render";
import type { PickItem } from "./render";
import { applyCleanLocally } from "./lib/clean";
import { hb } from "./lib/format";
import {
    activeLocale,
    fmtTime,
    normalizeNavigator,
    setDict,
    t,
} from "./lib/i18n";
import { kindLabel as renderKindLabel } from "./lib/labels";
import type { ReminderConfig, TrendsResult } from "./lib/types";
import {
    CancelDownload,
    CheckForUpdates,
    CheckToolUpdate,
    Clean,
    DownloadUpdate,
    GetDownloadProgress,
    GetLanguage,
    GetLastResult,
    GetReminderConfig,
    GetTranslations,
    GetTrashConfig,
    GetTrends,
    GetUpdateConfig,
    GetUninstallStatus,
    GetUpgradeStatus,
    IgnoreVersion,
    InstallUpdate,
    OpenURL,
    OrphanTrash,
    PurgeNow,
    Restore,
    RunToolUpgrade,
    Scan,
    SetLanguage,
    SetLanguagePreference,
    SetReminderConfig,
    SetTrashConfig,
    SetUpdateConfig,
    TrashInfo,
    TrashList,
    UninstallDeleteResidues,
    UninstallResidue,
    UninstallRunOfficial,
    UninstallStart,
    UninstallTrashResidues,
} from "../wailsjs/go/gui/ScannerService";

// applyI18n 由 main.ts 提供（涉及 theme + render + flows 的全局刷新），flows 侧仅转发
let applyI18nFromFlows: () => void = () => {};
export function setApplyI18n(fn: () => void) {
    applyI18nFromFlows = fn;
}

// 扫描状态（方案 B+C 结合）：
// - 「重新扫描」按钮自身变为加载态：灰色中性 + 小转圈 + 「扫描中…」，禁用
// - 底部状态栏同步显示 转圈 + 「扫描中…」（与扫描用时/遍历错误同区）
export function setScanning(busy: boolean, label = "") {
    const s = el("scanState");
    s.className = "scan-state" + (busy ? " busy" : " hidden");
    s.innerHTML = busy ? `<span class="btn-spin"></span>${esc(label)}` : "";
    const btn = el("rescanBtn") as HTMLButtonElement;
    btn.disabled = busy;
    btn.classList.toggle("scanning", busy);
    btn.innerHTML = busy
        ? `<span class="btn-spin"></span>${esc(label)}`
        : esc(t("menu.rescan"));
    // 扫描进行中隐藏旧结果的「扫描用时/遍历错误」（这些来自上一次扫描，
    // 扫描期间展示是误导）；平台信息是静态系统信息，保持可见。扫描完成
    // 由 renderSummary 填充新值；失败/取消路径恢复旧摘要，避免状态栏空白。
    const info = el("statusInfo");
    if (busy) {
        info.querySelectorAll(
            'span[data-role="scanTime"], span[data-role="walkErrors"]',
        ).forEach((s) => s.remove());
    } else if (!info.querySelector('span[data-role="scanTime"]')) {
        render.renderSummary();
    }
}

// ---- scan flow ----
export async function rescan() {
    setScanning(true, t("ui.scanning"));
    try {
        await Scan();
    } catch (e) {
        setScanning(false);
        showToast(t("ui.scanStartFailed", { err: String(e) }), true);
    }
}

// 移入内置回收站（可恢复）；成功后本地移除已移入项，后台重扫兜底校准。
export async function trashPaths(paths: string[]) {
    if (!paths.length) return;
    try {
        const rep = JSON.parse(await OrphanTrash(paths)) as {
            trashed?: string[];
            errors?: string[];
        };
        const hasErr = (rep.errors ?? []).length > 0;
        showToast(
            (rep.trashed?.length ? t("ui.orphanTrash") + " ✓" : "") +
                (hasErr ? t("un.runFailed") : ""),
            hasErr,
        );
        for (const p of rep.trashed ?? []) orphanSel.delete(p);
        if (result && (rep.trashed ?? []).length) {
            const gone = new Set(rep.trashed);
            result.unattributed = (result.unattributed ?? []).filter(
                (o) => !gone.has(o.path),
            );
            render.renderSummary();
            render.renderToolList();
        }
        refreshTrashInfo(); // 立即刷新右下角回收站占用
    } catch (e) {
        showToast(String(e), true);
    }
}

// 应用内确认弹窗（复用工具风格 #modal，替代原生 confirm()）：
// macOS WKWebView 不支持 window.confirm（点击永远无反应），Windows WebView2
// 的 confirm 是网页风格弹窗，与工具 UI 不一致。返回 Promise<boolean>。
export function confirmDialog(opts: {
    title: string;
    message: string;
    confirmText: string;
}): Promise<boolean> {
    return new Promise((resolve) => {
        el("modalTitle").textContent = opts.title;
        el("modalBody").innerHTML = `<p class="warn">${esc(opts.message)}</p>`;
        el("modalConfirm").textContent = opts.confirmText;
        const done = (ok: boolean) => {
            el("modal").classList.add("hidden");
            resolve(ok);
        };
        el("modalConfirm").onclick = () => done(true);
        el("modalCancel").onclick = () => done(false);
        el("modal").classList.remove("hidden");
    });
}

// 二次确认：展示待移入项（路径 + 大小）与可恢复说明，确认后才真正移入回收站
export function showOrphanConfirm(paths: string[]) {
    if (!paths.length) return;
    const items = paths.map((p) => {
        const o = (result?.unattributed ?? []).find((x) => x.path === p);
        return { path: p, bytes: o?.bytes ?? 0 };
    });
    const total = items.reduce((a, i) => a + i.bytes, 0);
    el("modalTitle").textContent = t("ui.orphanConfirmTitle", {
        n: items.length,
        size: hb(total),
    });
    const MAX = 15;
    const shown = items.slice(0, MAX);
    const more = items.length - shown.length;
    el("modalBody").innerHTML =
        shown
            .map(
                (i) =>
                    `<div class="clean-row">
            <span class="path" style="flex:1;font-family:var(--mono);font-size:11px;white-space:normal;word-break:break-all">${esc(i.path)}</span>
            <span class="size" style="font-family:var(--mono);color:var(--muted);white-space:nowrap">${hb(i.bytes)}</span>
        </div>`,
            )
            .join("") +
        (more > 0
            ? `<div class="sub-more">${esc(t("ui.moreSubitems", { n: more }))}</div>`
            : "") +
        `<div class="warn">${esc(t("ui.orphanConfirmNote"))}</div>`;
    el("modal").classList.remove("hidden");
    el("modalConfirm").textContent = t("ui.orphanConfirmBtn");
    el("modalConfirm").onclick = async () => {
        el("modal").classList.add("hidden");
        await trashPaths(paths);
    };
    el("modalCancel").onclick = () => el("modal").classList.add("hidden");
}

// ---- confirm modal ----
export function showConfirmModal(items: PickItem[]) {
    const total = items.reduce((a, c) => a + c.bytes, 0);
    el("modalTitle").textContent = t("ui.confirmClean", {
        n: items.length,
        size: hb(total),
    });
    el("modalConfirm").textContent = t("ui.confirmCleanBtn"); // 未认领确认弹窗可能改过按钮文案，这里复位
    el("modalBody").innerHTML =
        items
            .map(
                (c) =>
                    `<div class="clean-row">
            <span class="badge safe">${esc(renderKindLabel(c.kind))}</span>
            <span class="path" style="flex:1;font-family:var(--mono);font-size:11px;white-space:normal;word-break:break-all">${esc(c.path)}</span>
            <span class="size" style="font-family:var(--mono);color:var(--clean);white-space:nowrap">${hb(c.bytes)}</span>
        </div>`,
            )
            .join("") +
        '<div class="warn">' +
        esc(t("ui.safeOnly")) +
        "</div>";
    el("modal").classList.remove("hidden");
    el("modalConfirm").onclick = async () => {
        el("modal").classList.add("hidden");
        const ids = items.map((c) => c.id);
        try {
            const rep: import("./state").CleanReport = JSON.parse(
                await Clean(ids, false),
            );
            const del = (rep.deleted ?? []).length;
            const skipped = (rep.skipped ?? []).length;
            const hasErr = (rep.errors ?? []).length > 0;
            const extra = skipped ? t("ui.cleanSkipped", { n: skipped }) : "";
            showToast(
                t("ui.cleanDone", { del, size: hb(rep.freedBytes), extra }),
                hasErr,
            );
            selectedCleanIds.clear();
            if (del > 0 && result) {
                // 本地立即刷新详情，无需等整体重扫
                applyCleanLocally(result, ids);
                render.renderSummary();
                render.renderToolList();
                render.renderDetail();
                refreshReminder();
                refreshTrashInfo();
            }
            rescan();
        } catch (e) {
            showToast(t("ui.cleanFailed", { err: String(e) }), true);
        }
    };
    el("modalCancel").onclick = () => el("modal").classList.add("hidden");
}

// ---- built-in trash ----
// 刷新状态栏常驻的回收站占用信息（空回收站显示 0 B）
export async function refreshTrashInfo() {
    try {
        const info: import("./state").TrashInfoData = JSON.parse(
            await TrashInfo(),
        );
        const total = info.totalBytes || 0;
        el("trashInfo").textContent =
            total > 0
                ? t("ui.trashInfo", { size: hb(total), n: info.items })
                : t("ui.trashInfoEmpty");
        el("trashBtn").title = info.earliestExpiresAt
            ? t("ui.trashBtnFull", {
                  size: hb(total),
                  n: info.items,
                  time: fmtTime(info.earliestExpiresAt),
              })
            : t("ui.trashBtnTitle");
    } catch {
        el("trashInfo").textContent = t("ui.trashInfoEmpty");
    }
}

export function openTrashPanel() {
    el("trashModal").classList.remove("hidden");
    refreshTrashList();
}

export async function refreshTrashList() {
    const body = el("trashBody");
    try {
        const parsed = JSON.parse(await TrashList());
        if (parsed.error) {
            body.innerHTML = `<div class="warn">${esc(parsed.error)}</div>`;
            return;
        }
        setTrashItems(parsed);
    } catch (e) {
        body.innerHTML = `<div class="warn">${esc(t("ui.trashReadFailed", { err: String(e) }))}</div>`;
        return;
    }
    if (!trashItems.length) {
        body.innerHTML =
            '<div class="empty">' + esc(t("ui.trashEmpty")) + "</div>";
        el("trashSummary").textContent = "";
        return;
    }
    // 头部汇总：项数 · 总大小 · 最早到期时间
    const total = trashItems.reduce((a, it) => a + (it.bytes || 0), 0);
    const earliest = trashItems.reduce(
        (m, it) => (it.expiresAt < m ? it.expiresAt : m),
        trashItems[0].expiresAt,
    );
    el("trashSummary").textContent = t("ui.trashSummary", {
        n: trashItems.length,
        size: hb(total),
        time: fmtTime(earliest),
    });
    // 表格行：类型标签（方形色块）｜路径｜大小｜来源·时间｜恢复｜删除
    body.innerHTML = trashItems
        .map((it) => {
            // 未认领数据移入回收站时后端以内部标识 "orphan" 记录（稳定 ID，供恢复/到期
            // 处理），展示层翻译为「未认领数据」，不把内部名暴露给用户。
            const isOrphan = it.tool === "orphan";
            const tone = isOrphan ? "blue" : (KIND_TONE[it.kind] ?? "gray");
            const label = isOrphan
                ? t("ui.orphanSection")
                : renderKindLabel(it.kind);
            const exp = t("ui.trashExp", {
                time: fmtTime(it.trashedAt),
                time2: fmtTime(it.expiresAt),
            });
            const meta = isOrphan ? exp : `${esc(it.tool)} · ${exp}`;
            return `
        <div class="trash-row">
            <span class="ktag ${tone}">${esc(label)}</span>
            <span class="tpath" title="${esc(it.original)}">${esc(it.original)}</span>
            <span class="tsize">${hb(it.bytes)}</span>
            <span class="tmeta" title="${esc(meta)}">${meta}</span>
            <button class="ibtn" data-restore="${esc(it.id)}" title="${esc(t("ui.restore"))}">${RESTORE_ICON}</button>
            <button class="ibtn danger" data-purge="${esc(it.id)}" title="${esc(t("ui.purge"))}">${TRASH_ICON}</button>
        </div>`;
        })
        .join("");
    body.querySelectorAll<HTMLButtonElement>("[data-restore]").forEach(
        (btn) => {
            btn.onclick = async () => {
                const r = JSON.parse(await Restore(btn.dataset.restore!));
                if (r.error)
                    showToast(t("ui.restoreFailed", { err: r.error }), true);
                else {
                    showToast(t("ui.restored", { path: r.restored }));
                    rescan(); // 后台重扫，让主界面详情反映恢复的文件
                }
                refreshTrashList();
                refreshTrashInfo();
            };
        },
    );
    body.querySelectorAll<HTMLButtonElement>("[data-purge]").forEach((btn) => {
        btn.onclick = async () => {
            // 永久删除前必须确认（自 0.3.8 起 PurgeNow 为不可恢复的永久删除）
            const ok = await confirmDialog({
                title: t("ui.purge"),
                message: t("ui.purgeConfirm", { n: 1 }),
                confirmText: t("ui.purge"),
            });
            if (!ok) return;
            const r = JSON.parse(await PurgeNow([btn.dataset.purge!]));
            if ((r.errors ?? []).length)
                showToast(
                    t("ui.purgeFailed", { err: r.errors.join("; ") }),
                    true,
                );
            else {
                showToast(t("ui.purged"));
                rescan();
            }
            refreshTrashList();
            refreshTrashInfo();
        };
    });
}

// 下载进度改用轮询（GetDownloadProgress）：macOS WKWebView 对跨桥事件不可靠，
// 事件推进度在 Mac 上完全不显示，轮询与 GetUpdateStatus 同一套路（已验证有效）。
export function stopDownloadPoll() {
    if (downloadPoll !== null) {
        window.clearInterval(downloadPoll);
        setDownloadPoll(null);
    }
}
export function startDownloadPoll() {
    stopDownloadPoll();
    setLastShownPct(0);
    setDownloadPoll(
        window.setInterval(async () => {
            try {
                const raw = await GetDownloadProgress();
                if (!raw) return; // 已结束或未开始，由完成事件兜底
                const p = JSON.parse(raw) as {
                    downloaded: number;
                    total: number;
                };
                if (updateState !== "downloading") return;
                const bar = el("updBar") as HTMLElement;
                const pct = el("updPct");
                // 未开始或仍为 0%：保持初始 "0%"，不显示 0 字节、不切准备中文案（避免闪烁）
                if (!p.total || p.downloaded <= 0) return;
                const n = Math.min(
                    100,
                    Math.round((p.downloaded / p.total) * 100),
                );
                if (n < 1) return; // 不足 1% 仍保持 0%
                // 单调防线：进度只进不退（双保险，Go 侧已有同样守卫）
                if (n < lastShownPct) return;
                setLastShownPct(n);
                bar.style.width = n + "%";
                pct.textContent = `${n}%（${hb(p.downloaded)} / ${hb(p.total)}）`;
            } catch {
                /* 单次轮询失败忽略 */
            }
        }, 200),
    );
}

// 发现新版本：展示 当前/最新 版本与 [下载] [忽略该版本] [稍后]
export function showUpdateAvailable(res: import("./state").UpdateResult) {
    if (!res.updateAvailable) return;
    setLastUpdateResult(res);
    setUpdateState("idle");
    const body = el("updateBody");
    // 无下载入口（安装来源 unknown 且资产匹配失败）时只提供 Release 页链接，
    // 不显示无效的下载按钮（DownloadUpdate 对空 URL 只返回提示）
    const dlAction = res.downloadURL
        ? `<button class="btn primary" id="updDownload">${esc(t("upd.download"))}</button>`
        : `<button class="btn primary" id="updRelease">${esc(t("upd.releasePage"))}</button>`;
    body.innerHTML = `
        <p class="update-versions">${t("upd.versions", { current: res.current, latest: res.latest })}</p>
        <p class="muted">${esc(t("upd.manualInstall"))}</p>
        <div class="update-actions">
            <button class="btn" id="updLater">${esc(t("upd.later"))}</button>
            <button class="btn" id="updIgnore">${esc(t("upd.ignore"))}</button>
            ${dlAction}
        </div>`;
    el("updateModal").classList.remove("hidden");
    el("updLater").onclick = () => el("updateModal").classList.add("hidden");
    el("updIgnore").onclick = async () => {
        await IgnoreVersion(res.latest);
        el("updateModal").classList.add("hidden");
        showToast(t("upd.ignored", { version: res.latest }));
    };
    const dl = document.getElementById("updDownload");
    if (dl) dl.onclick = () => startDownload();
    const rl = document.getElementById("updRelease");
    if (rl)
        rl.onclick = () =>
            OpenURL(
                res.releaseURL ||
                    "https://github.com/kevinjoy89/cli-analyzer/releases",
            );
}

// 开始下载：进度条 + 取消；进度由 update:progress 事件驱动
export function startDownload() {
    setUpdateState("downloading");
    const body = el("updateBody");
    body.innerHTML = `
        <p class="update-versions">${t("upd.downloading", { name: esc(lastUpdateResult?.assetName ?? "") })}</p>
        <div class="progress"><div id="updBar" class="progress-bar" style="width:0%"></div></div>
        <p id="updPct" class="muted">0%</p>
        <div class="update-actions"><button class="btn" id="updCancel">${esc(t("upd.cancel"))}</button></div>`;
    el("updCancel").onclick = () => {
        CancelDownload();
    };
    DownloadUpdate().then((err) => {
        if (err) {
            showUpdateFailed(
                t("upd.startFailed", { err }),
                lastUpdateResult?.releaseURL ?? "",
            );
            return;
        }
        startDownloadPoll(); // 进度轮询（事件在 macOS 不可靠）
    });
}

// 下载完成（已通过校验）：[立即安装] [稍后]；压缩包产物额外展示当前二进制路径
export function showUpdateDownloaded(d: import("./state").UpdateDownloaded) {
    setUpdateState("downloaded");
    stopDownloadPoll();
    const isArchive =
        d.installSource === "tarball" || d.installSource === "portable";
    const body = el("updateBody");
    body.innerHTML = `
        <p class="update-versions">${esc(t("upd.downloaded"))}</p>
        <p class="muted">${isArchive ? esc(t("upd.archiveSaved")) : esc(t("upd.installerSaved"))}<code>${esc(d.path)}</code></p>
        ${isArchive ? `<p class="muted">${esc(t("upd.binaryPath"))}<code>${esc(d.executablePath)}</code></p>` : ""}
        <div class="update-actions">
            <button class="btn" id="updLater">${esc(t("upd.later"))}</button>
            <button class="btn primary" id="updInstall">${esc(t("upd.installNow"))}</button>
        </div>`;
    el("updLater").onclick = () => el("updateModal").classList.add("hidden");
    // 点击后 Go 侧先打开安装包再退出应用（design D7）
    el("updInstall").onclick = () => {
        InstallUpdate();
    };
}

// 校验失败：安全优先，不给安装入口，仅提供 Release 页面链接
export function showUpdateVerifyFailed(err: string, releaseURL: string) {
    setUpdateState("idle");
    const body = el("updateBody");
    body.innerHTML = `
        <p class="warn">${esc(t("upd.verifyFailed"))}</p>
        <p class="muted">${esc(err)}</p>
        <div class="update-actions">
            <button class="btn" id="updClose">${esc(t("upd.close"))}</button>
            <button class="btn" id="updRelease">${esc(t("upd.releasePage"))}</button>
        </div>`;
    el("updClose").onclick = () => el("updateModal").classList.add("hidden");
    el("updRelease").onclick = () =>
        OpenURL(
            releaseURL || "https://github.com/kevinjoy89/cli-analyzer/releases",
        );
}

// 下载失败：保留面板展示错误信息，提供重试与 Release 页跳转（不自动关闭）
export function showUpdateFailed(err: string, releaseURL: string) {
    setUpdateState("idle");
    const body = el("updateBody");
    body.innerHTML = `
        <p class="warn">${esc(t("upd.downloadFailed"))}</p>
        <p class="muted">${esc(err)}</p>
        <div class="update-actions">
            <button class="btn" id="updClose">${esc(t("upd.close"))}</button>
            <button class="btn" id="updRetry">${esc(t("upd.download"))}</button>
            <button class="btn" id="updRelease">${esc(t("upd.releasePage"))}</button>
        </div>`;
    el("updClose").onclick = () => el("updateModal").classList.add("hidden");
    el("updRetry").onclick = () => startDownload();
    el("updRelease").onclick = () =>
        OpenURL(
            releaseURL || "https://github.com/kevinjoy89/cli-analyzer/releases",
        );
}

// 手动检查（Help 菜单「检查更新…」）：不受 4h 缓存限制
export async function manualCheck() {
    let res: import("./state").UpdateResult;
    try {
        res = JSON.parse(await CheckForUpdates());
    } catch (e) {
        showToast(t("upd.checkFailed", { err: String(e) }), true);
        return;
    }
    if (res.error) {
        showToast(res.error, true);
        return;
    }
    if (res.updateAvailable) showUpdateAvailable(res);
    else showToast(t("upd.upToDate", { version: res.latest }));
}

// ---- uninstall ----
export function stopUninstallPoll() {
    if (uninstallPoll !== null) {
        window.clearInterval(uninstallPoll);
        setUninstallPoll(null);
    }
}

// 详情页「卸载」→ 起始信息（官方命令/黑名单/占用）→ 确认弹窗
export function startUninstall(toolName: string) {
    UninstallStart(toolName)
        .then((raw) => {
            const info = JSON.parse(
                raw,
            ) as import("./state").UninstallStartInfo;
            if (info.stale) {
                showToast(info.error || "", true);
                rescan();
                return;
            }
            if (info.error) {
                showToast(info.error, true);
                return;
            }
            if (info.blocked) {
                showToast(info.blockedReason || "", true);
                return;
            }
            showUninstallConfirm(info);
        })
        .catch((e) => showToast(String(e), true));
}

export function showUninstallConfirm(
    info: import("./state").UninstallStartInfo,
) {
    const body = el("uninstallBody");
    // 标题承载「卸载 <tool>」，body 不再重复工具名
    el("uninstallTitle").textContent = t("un.guiUninstall") + " " + info.tool;
    // 复制按钮内联在命令后面（极简 SVG 图标），不放操作行
    const cmdHtml = info.officialCommand
        ? `<p class="muted un-cmdline">${esc(t("un.guiOfficialCmd"))}<br><code>${esc(info.officialCommand)}</code> <button id="unCopy" class="btn icon" title="${esc(t("un.guiCopyCmd"))}"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button></p>`
        : `<p class="muted">${esc(t("un.noOfficialCmd"))}</p>`;
    // 两行按钮：主文案 + 小字说明（语义：主路径 = 卸载并自动检测；跳过 = 直达残留检测）
    const runnableActions = info.runnable
        ? `
            <button class="btn" id="unSkip"><span class="btn-main">${esc(t("un.guiSkipMain"))}</span><span class="btn-sub">${esc(t("un.guiSkipSub"))}</span></button>
            <button class="btn primary" id="unRun"><span class="btn-main">${esc(t("un.guiRunMain"))}</span><span class="btn-sub">${esc(t("un.guiRunSub"))}</span></button>`
        : `
            <button class="btn primary" id="unResidue">${esc(t("un.guiResidueTitle"))}</button>`;
    body.innerHTML = `
        <p class="muted">占用 ${hb(info.footprint || 0)} · 用户数据 ${hb(info.userBytes || 0)}</p>
        ${cmdHtml}
        <p id="unOut" class="muted un-output"></p>
        <div class="update-actions">
            <button class="btn" id="unCancel">${esc(t("ui.cancel"))}</button>
            ${runnableActions}
        </div>`;
    el("uninstallModal").classList.remove("hidden");
    el("unCancel").onclick = () => {
        stopUninstallPoll();
        el("uninstallModal").classList.add("hidden");
    };
    // 条件渲染的按钮用 getElementById 可空查找（el() 对缺失元素会 throw，
    // 导致后续按钮事件全部丢失——曾报 "missing element #unRun"）
    const copyBtn = document.getElementById("unCopy");
    if (copyBtn)
        copyBtn.onclick = () => {
            navigator.clipboard
                ?.writeText(info.officialCommand!)
                .catch(() => {});
            showToast(t("un.copied"));
        };
    const runBtn = document.getElementById("unRun") as HTMLButtonElement | null;
    if (runBtn)
        runBtn.onclick = async () => {
            // 标准卸载会移除程序文件：执行前必须二次确认（此前直接执行）
            const ok = await confirmDialog({
                title: t("un.guiRunMain"),
                message: t("un.guiRunConfirm"),
                confirmText: t("un.guiRunMain"),
            });
            if (!ok) return;
            runBtn.disabled = true;
            const err = await UninstallRunOfficial();
            if (err) {
                showToast(err, true);
                runBtn.disabled = false;
                return;
            }
            startUninstallPoll();
        };
    const skipBtn = document.getElementById("unSkip");
    if (skipBtn) skipBtn.onclick = () => showUninstallResidue();
    const residueBtn = document.getElementById("unResidue");
    if (residueBtn) residueBtn.onclick = () => showUninstallResidue();
}

export function startUninstallPoll() {
    stopUninstallPoll();
    const t0 = Date.now();
    setUninstallPoll(
        window.setInterval(async () => {
            try {
                const st = JSON.parse(
                    await GetUninstallStatus(),
                ) as import("./state").UninstallStatus;
                const out = el("unOut");
                if (out) {
                    // 有输出展示输出；无输出时显示进行中 + 已耗时（标准卸载可能需数十秒）
                    out.textContent = st.output
                        ? st.output
                        : st.running
                          ? `${t("un.guiRunning")}（${Math.round((Date.now() - t0) / 1000)}s）`
                          : "";
                }
                if (st.done) {
                    stopUninstallPoll();
                    if (st.error)
                        showToast(t("un.runFailed") + ": " + st.error, true);
                    // 进入残留检测
                    showUninstallResidue();
                }
            } catch {
                /* 单次轮询失败忽略 */
            }
        }, 300),
    );
}

// 残留检测 → 列表（全选默认、凭证标红）→ 移入回收站
// 检测可能需数秒（扫描大数据目录，如 uv 缓存 7GB+）：先渲染进行中状态，
// 避免点击后无反馈像假死；完成后替换为残留列表。
export async function showUninstallResidue() {
    const body = el("uninstallBody");
    el("uninstallTitle").textContent = t("un.guiResidueTitle");
    body.innerHTML = `<p class="muted">${esc(t("un.guiResidueScanning"))}</p><p class="muted un-residue-hint">${esc(t("un.guiResidueScanningHint"))}</p>`;
    let rr: import("./state").UninstallResidueItem[] = [];
    try {
        const parsed = JSON.parse(await UninstallResidue());
        rr = Array.isArray(parsed) ? parsed : [];
    } catch {
        const msg = t("un.residueNone");
        body.innerHTML = `<p class="muted">${esc(msg)}</p>`;
        showToast(msg, true);
        return;
    }
    if (!rr.length) {
        body.innerHTML = `<p class="muted">${esc(t("un.guiResidueNone"))}</p>`;
        showToast(t("un.guiResidueNone"));
        return;
    }
    el("uninstallTitle").textContent = t("un.guiResidueTitle");
    body.innerHTML = `
        ${rr
            .map(
                (r, i) => `
            <label class="pref-row check">
                <input type="checkbox" data-idx="${i}" checked/>
                <span class="${r.tier === "user" ? "un-credential" : ""}">${esc(r.path)} · ${hb(r.bytes)}${r.tier === "user" ? " · " + esc(t("un.guiResidueCredential")) : ""}</span>
            </label>`,
            )
            .join("")}
        <div class="update-actions un-actions">
            <button class="btn" id="unCancel2">${esc(t("ui.cancel"))}</button>
            <button class="btn danger" id="unDeletePerm">${esc(t("un.guiDeletePermanent"))}</button>
            <button class="btn danger" id="unTrash"><span class="btn-main">${esc(t("un.guiTrashMain"))}</span><span class="btn-sub">${esc(t("un.guiTrashSub"))}</span></button>
        </div>`;
    el("uninstallModal").classList.remove("hidden");
    el("unCancel2").onclick = () =>
        el("uninstallModal").classList.add("hidden");
    const checkedPaths = () => {
        const boxes = body.querySelectorAll<HTMLInputElement>(
            "input[type=checkbox]",
        );
        return rr.filter((_, i) => boxes[i].checked).map((r) => r.path);
    };
    el("unTrash").onclick = async () => {
        const paths = checkedPaths();
        if (!paths.length) {
            showToast(t("un.residueNone"));
            return;
        }
        const rep = JSON.parse(await UninstallTrashResidues(paths));
        el("uninstallModal").classList.add("hidden");
        const hasErr = (rep.errors ?? []).length > 0;
        showToast(
            t("un.guiResidueDone") +
                (hasErr ? "（" + t("un.runFailed") + "）" : ""),
            hasErr,
        );
        // 立即刷新右下角回收站占用（不等后台重扫完成，与 clean 流程一致）
        refreshTrashInfo();
    };
    // 永久删除：不可恢复，必须先经强确认（confirmDialog），再调后端
    el("unDeletePerm").onclick = async () => {
        const paths = checkedPaths();
        if (!paths.length) {
            showToast(t("un.residueNone"));
            return;
        }
        const ok = await confirmDialog({
            title: t("un.guiDeletePermanent"),
            message: t("un.guiDeletePermanentConfirm"),
            confirmText: t("un.guiDeletePermanent"),
        });
        if (!ok) return;
        const rep = JSON.parse(await UninstallDeleteResidues(paths));
        el("uninstallModal").classList.add("hidden");
        const hasErr = (rep.errors ?? []).length > 0;
        showToast(
            t("un.guiResidueDonePermanent") +
                (hasErr ? "（" + t("un.runFailed") + "）" : ""),
            hasErr,
        );
        refreshTrashInfo();
        rescan(); // 永久删除真正释放空间，后台重扫校准占用
    };
}

// ---- tool upgrade（工具升级）----
// 与 uninstall 同为「详情页显式触发 + 页面守卫」：检测请求在途时用户离开
// 详情页 → 结果静默丢弃（不弹窗不提示）；按钮在渲染详情页时重建，无需复位。
export function stopUpgradePoll() {
    if (upgradePoll !== null) {
        window.clearInterval(upgradePoll);
        setUpgradePoll(null);
    }
}

// 详情页「检查更新」→ 服务端检测（无缓存，每次全新查询）→ 展示结果/代跑
// 页面守卫在 promise resolve 时校验当前详情页工具是否仍为触发者。
// upgradeCheckSeq 记录最近一次发起的检测序号：并发检测（查 A 后离开再查 B）
// 时，只有「最近发起」的检测的 finally 才复位按钮，避免陈旧检测把新检测的
// 「检查中…」按钮误复位成可用（导致重复点击）（code review #5 修复）。
let upgradeCheckSeq = 0;
export function startToolUpgrade(toolName: string) {
    const seq = ++upgradeCheckSeq;
    const btn = document.getElementById(
        "upgradeBtn",
    ) as HTMLButtonElement | null;
    if (btn) {
        btn.disabled = true;
        btn.textContent = t("up.guiChecking");
    }
    CheckToolUpdate(toolName)
        .then((raw) => {
            // 页面守卫：已离开该工具的详情页 → 丢弃检测结果
            if (selected !== toolName) return;
            let res: import("./state").UpgradeCheckResult;
            try {
                res = JSON.parse(raw);
            } catch {
                return;
            }
            // 检测失败（网络/命令缺失/超时）时 res.detected=false 且 res.error 携带
            // 原因；GUI 按 detected 降级展示「无法检测 + 官方升级命令」，不把 error
            // 当致命错误（error 仅供 CLI/调试，见 upgrade.CheckResult 契约注释）。
            if (!res.detected) {
                showUpgradeResult(toolName, res);
                return;
            }
            if (!res.hasUpdate) {
                showToast(t("up.guiUpToDate", { tool: res.name }));
                return;
            }
            showUpgradeResult(toolName, res);
        })
        .catch(() => {
            /* 查询失败静默：按钮经下方 finally 复位 */
        })
        .finally(() => {
            // 已有更新的检测发起（本检测已陈旧）→ 不复位按钮，交给新检测的 finally
            if (seq !== upgradeCheckSeq) return;
            // 已离开详情页时 renderDetail 已重建按钮（新工具页面），这里只恢复
            // 仍在当前页面上的按钮状态（文本 + 可用）。
            const b = document.getElementById(
                "upgradeBtn",
            ) as HTMLButtonElement | null;
            if (b) {
                b.disabled = false;
                b.textContent = t("up.guiCheckUpdate");
            }
        });
}

// 升级弹窗：版本信息 + 官方命令（可复制）→ [代跑升级]
function showUpgradeResult(
    toolName: string,
    res: import("./state").UpgradeCheckResult,
) {
    // 无官方升级命令：不弹面板（按钮已按 ToolUpgradeSupported 隐藏，此处
    // 仅防御：任何来源退化到无命令时直接静默，不展示无意义的提示面板）
    if (!res.command) return;
    const body = el("upgradeBody");
    el("upgradeTitle").textContent = t("up.guiCheckUpdate") + " " + toolName;
    const cmdHtml = res.command
        ? `<p class="muted un-cmdline">${esc(t("up.guiCmd"))}<br><code>${esc(res.command)}</code> <button id="ugCopy" class="btn icon" title="${esc(t("un.guiCopyCmd"))}"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button></p>`
        : "";
    // 无检测能力：只给命令提示，不给代跑入口
    if (!res.detected) {
        body.innerHTML = `
            <p class="muted">${esc(t("up.guiNoDetect"))}</p>
            ${cmdHtml}
            <div class="update-actions"><button class="btn" id="ugClose">${esc(t("ui.close"))}</button></div>`;
        el("upgradeModal").classList.remove("hidden");
        el("ugClose").onclick = () =>
            el("upgradeModal").classList.add("hidden");
        const copyBtn = document.getElementById("ugCopy");
        if (copyBtn)
            copyBtn.onclick = () => {
                navigator.clipboard?.writeText(res.command!).catch(() => {});
                showToast(t("un.copied"));
            };
        return;
    }
    body.innerHTML = `
        <p class="update-versions">${t("up.guiFound", { tool: res.name, current: res.current, latest: res.latest })}</p>
        ${cmdHtml}
        <p id="ugOut" class="muted un-output"></p>
        <div class="update-actions">
            <button class="btn" id="ugClose">${esc(t("ui.cancel"))}</button>
            ${res.runnable ? `<button class="btn primary" id="ugRun"><span class="btn-main">${esc(t("up.guiRun"))}</span><span class="btn-sub">${esc(t("up.guiRunSub"))}</span></button>` : ""}
        </div>`;
    el("upgradeModal").classList.remove("hidden");
    el("ugClose").onclick = () => {
        stopUpgradePoll();
        el("upgradeModal").classList.add("hidden");
    };
    const copyBtn = document.getElementById("ugCopy");
    if (copyBtn)
        copyBtn.onclick = () => {
            navigator.clipboard?.writeText(res.command!).catch(() => {});
            showToast(t("un.copied"));
        };
    const runBtn = document.getElementById("ugRun") as HTMLButtonElement | null;
    if (runBtn)
        runBtn.onclick = async () => {
            // 代跑会修改工具安装（升级/重装）：执行前先经应用内确认弹窗
            const ok = await confirmDialog({
                title: t("up.guiRun"),
                message: t("up.guiRunConfirm", { tool: toolName }),
                confirmText: t("up.guiRun"),
            });
            if (!ok) return;
            runBtn.disabled = true;
            const err = await RunToolUpgrade(toolName);
            if (err) {
                showToast(err, true);
                runBtn.disabled = false;
                return;
            }
            startUpgradePoll();
        };
}

// 代跑进度轮询（GetUpgradeStatus，与 uninstall 同套路）
export function startUpgradePoll() {
    stopUpgradePoll();
    const t0 = Date.now();
    setUpgradePoll(
        window.setInterval(async () => {
            try {
                const st = JSON.parse(
                    await GetUpgradeStatus(),
                ) as import("./state").UpgradeStatus;
                const out = el("ugOut");
                out.textContent = st.output
                    ? st.output
                    : st.running
                      ? `${t("up.guiRunning")}（${Math.round((Date.now() - t0) / 1000)}s）`
                      : "";
                if (st.done) {
                    stopUpgradePoll();
                    if (st.error) {
                        showToast(t("up.guiFailed") + ": " + st.error, true);
                        return;
                    }
                    showToast(t("up.guiDone"));
                    el("upgradeModal").classList.add("hidden");
                    // 升级成功：后端已重探测该工具版本；拉取最新结果刷新界面
                    //（复用 scan:done 的渲染路径，不触发全量扫描）
                    try {
                        const raw = await GetLastResult();
                        if (raw) {
                            setResult(JSON.parse(raw));
                            render.renderSummary();
                            render.renderToolList();
                            render.renderDetail();
                            refreshTrashInfo();
                            refreshReminder();
                        }
                    } catch {
                        /* 刷新失败不阻塞 */
                    }
                }
            } catch {
                /* 单次轮询失败忽略 */
            }
        }, 300),
    );
}

// ---- preferences ----
export async function openPrefs() {
    let cfg: import("./state").TrashConfig;
    let rem: ReminderConfig;
    let upd: import("./state").UpdateConfig;
    try {
        cfg = JSON.parse(await GetTrashConfig());
    } catch {
        cfg = {
            retentionDays: 7,
            expireAction: "system-trash",
            useTrash: true,
        };
    }
    try {
        rem = JSON.parse(await GetReminderConfig());
    } catch {
        rem = { thresholdBytes: 5 * 1024 * 1024 * 1024 };
    }
    try {
        upd = JSON.parse(await GetUpdateConfig());
    } catch {
        upd = {};
    }
    let cfgLang = "";
    try {
        cfgLang = await GetLanguage();
    } catch {
        /* auto */
    }
    const threshGB =
        (rem.thresholdBytes || 5 * 1024 * 1024 * 1024) / (1024 * 1024 * 1024);
    const langOpts: Array<[string, string]> = [
        ["auto", t("ui.langAuto")],
        ["zh-CN", t("ui.langZhCN")],
        ["zh-TW", t("ui.langZhTW")],
        ["en", t("ui.langEn")],
    ];
    el("prefsBody").innerHTML = `
        <div class="pref-head">${esc(t("ui.prefsLanguage"))}</div>
        <label class="pref-row">${esc(t("ui.prefsLanguage"))}
            <select id="prefLanguage">
                ${langOpts.map(([v, label]) => `<option value="${v}" ${cfgLang === v ? "selected" : ""}>${esc(label)}</option>`).join("")}
            </select>
        </label>
        <div class="pref-head">${esc(t("ui.prefsTrashHead"))}</div>
        <label class="pref-row">${esc(t("ui.prefsRetention"))}
            <input id="prefRetention" type="number" min="1" max="365" value="${cfg.retentionDays}"/>
        </label>
        <label class="pref-row">${esc(t("ui.prefsExpire"))}
            <select id="prefExpire">
                <option value="system-trash" ${cfg.expireAction === "system-trash" ? "selected" : ""}>${esc(t("ui.prefsExpireSystem"))}</option>
                <option value="permanent" ${cfg.expireAction === "permanent" ? "selected" : ""}>${esc(t("ui.prefsExpirePermanent"))}</option>
            </select>
        </label>
        <label class="pref-row check">
            <input id="prefUseTrash" type="checkbox" ${cfg.useTrash ? "checked" : ""}/>
            <span>${esc(t("ui.prefsUseTrash"))}</span>
        </label>
        <div class="pref-head">${esc(t("ui.prefsTrendsHead"))}</div>
        <label class="pref-row">${esc(t("ui.prefsThreshold"))}
            <input id="prefThreshold" type="number" min="0" step="0.5" value="${threshGB.toFixed(1)}"/>
            <span>GB</span>
        </label>
        <div class="pref-head">${esc(t("ui.prefsUpdateHead"))}</div>
        <label class="pref-row check">
            <input id="prefCheckUpdates" type="checkbox" ${upd.checkUpdates !== false ? "checked" : ""}/>
            <span>${esc(t("ui.prefsAutoCheck"))}</span>
        </label>`;
    el("prefsModal").classList.remove("hidden");
    el("prefsSave").onclick = async () => {
        const lang = (el("prefLanguage") as HTMLSelectElement).value;
        const next: import("./state").TrashConfig = {
            retentionDays: Math.max(
                1,
                parseInt((el("prefRetention") as HTMLInputElement).value) || 7,
            ),
            expireAction: (el("prefExpire") as HTMLSelectElement).value,
            useTrash: (el("prefUseTrash") as HTMLInputElement).checked,
        };
        const rem2: ReminderConfig = {
            thresholdBytes: Math.round(
                parseFloat(
                    (el("prefThreshold") as HTMLInputElement).value || "5",
                ) *
                    1024 *
                    1024 *
                    1024,
            ),
        };
        // 合并回读的缓存字段，避免整体覆盖丢 lastCheckAt / ignoredVersion
        const upd2: import("./state").UpdateConfig = {
            checkUpdates: (el("prefCheckUpdates") as HTMLInputElement).checked,
            lastCheckAt: upd.lastCheckAt,
            ignoredVersion: upd.ignoredVersion,
        };
        const err = await SetTrashConfig(JSON.stringify(next));
        const rerr = await SetReminderConfig(JSON.stringify(rem2));
        const uerr = await SetUpdateConfig(JSON.stringify(upd2));
        const lerr = await SetLanguagePreference(lang);
        // 语言切换：拉取新字典 → 即时重渲染 + 同步 Go 侧
        if (!lerr && lang !== activeLocale()) {
            const resolved =
                lang === "auto"
                    ? normalizeNavigator(navigator.language || "")
                    : lang;
            try {
                const raw = JSON.parse(await GetTranslations(resolved));
                setDict(raw.locale, raw.dict);
                await SetLanguage(raw.locale);
            } catch {
                /* 保持当前字典 */
            }
        }
        el("prefsModal").classList.add("hidden");
        if (err || rerr || uerr || lerr)
            showToast(
                t("ui.saveFailed", { err: err || rerr || uerr || lerr }),
                true,
            );
        else {
            showToast(t("ui.saved"));
            refreshTrashInfo();
            refreshReminder();
            applyI18nFromFlows();
            // 原生菜单下次启动生效（仅 macOS 原生菜单；Windows/Linux 为 HTML 菜单即时切换）
            if (lang !== cfgLang && isMac) showToast(t("ui.menuRestartNote"));
        }
    };
    el("prefsCancel").onclick = () => el("prefsModal").classList.add("hidden");
}

// ---- about dialog ----
export function openAbout() {
    el("aboutVersion").textContent = appVersion ? `v${appVersion}` : "";
    el("aboutModal").classList.remove("hidden");
}

// ---- usage trends / reminder ----
// 刷新待清理提醒：按阈值筛选超限工具，更新铃铛徽标与下拉面板
export async function refreshReminder() {
    const bell = el("bellBtn");
    if (!result) {
        bell.classList.add("hidden");
        setReminderTools([]);
        return;
    }
    let thresh = 5 * 1024 * 1024 * 1024;
    try {
        thresh =
            (JSON.parse(await GetReminderConfig()) as ReminderConfig)
                .thresholdBytes || thresh;
    } catch {
        /* 默认 */
    }
    setReminderTools(
        result.tools.filter((tool) => tool.cleanableBytes > thresh),
    );
    if (!reminderTools.length) {
        bell.classList.add("hidden");
        el("bellPanel").classList.remove("open");
        return;
    }
    bell.classList.remove("hidden");
    el("bellCount").textContent = String(reminderTools.length);
    if (el("bellPanel").classList.contains("open")) render.renderBellPanel();
}

export function openTrends() {
    el("trendsModal").classList.remove("hidden");
    refreshTrends();
}

export async function refreshTrends() {
    let tr: TrendsResult;
    try {
        const parsed = JSON.parse(await GetTrends(30));
        if (parsed.error) {
            el("trendChart").innerHTML =
                `<div class="warn">${esc(parsed.error)}</div>`;
            return;
        }
        tr = parsed;
    } catch (e) {
        el("trendChart").innerHTML =
            `<div class="warn">${esc(t("ui.trendsReadFailed", { err: String(e) }))}</div>`;
        return;
    }
    render.renderTrendChart(el("trendChart"), tr.points);
    render.renderGrowers(el("trendGrowers"), tr.topGrowers);
}

# 鍓嶇 main.ts 鎷嗘ā鍧?
> 鍙樻洿鐩綍锛?026-08-15-frontend-module-split锛涙棤鑳藉姏鍙樻洿锛堢函閲嶆瀯锛?
> **鎵ц鐜锛堝繀璇伙級**锛氭湰浠撳簱鍦?DSH 娌欑涓嬪伐浣滐紝鎵€鏈?`pwsh` 鍛戒护锛坣pm / npx / tsc / git 绛夛級褰撳墠琚矙绠辨嫤鎴紙`SetNamedSecurityInfoW failed: grantWrite`锛夈€傚疄鏂芥椂姣忎竴姝ラ獙璇?鎻愪氦鍛戒护缁?`pwsh` 鎵ц浼氬脊鍑烘巿鏉冭姹傗€斺€旀壒鍑?`workspace-write` 鍗冲彲鏀捐锛涙巿鏉冧负浼氳瘽绾э紝鎵瑰噯鍚庡悗缁懡浠ら€氬父涓嶅啀閲嶅寮圭獥銆傛湰鍙樻洿鏃犳枃浠跺垹闄ゆ搷浣溿€?
## 1. dom.ts + state.ts

- [x] 1.1 鏂板缓 `frontend/src/dom.ts`锛氬壀鍒?main.ts 鐨?`el` / `esc` / `showToast`锛堢害 106-118銆?62-564 琛岋級
- [x] 1.2 鏂板缓 `frontend/src/state.ts`锛氬壀鍒囧叏閮ㄩ《灞傜姸鎬佷笌鐘舵€佺被鍨嬶紙result/probing/selected/appVersion/filterText/selectedCleanIds/expandedCleanIds/sortKey/sortDir/themeMode/isMac/panelView/orphanSel/trashItems/updateState/lastUpdateResult/downloadPoll/lastShownPct/uninstallPoll/reminderTools + interface CleanReport/TrashItem/鈥︼級锛岄噰鐢ㄥ懡鍚嶇┖闂村啓娉曪紙鏂规 A锛夛紱main.ts 鍐呴噸澶?interface 杩佸叆 state.ts 鎴栨敼 import lib/types锛圖4锛?- [ ] 1.3 main.ts 鏀?`import * as dom from './dom'; import * as state from './state';`
- [x] 1.4 楠岃瘉锛歚npx tsc --noEmit`锛? error锛? `npm test` + `npm run build`
- [x] 1.5 鎻愪氦锛歚refactor(frontend): extract dom primitives and global state into modules`

## 2. render.ts

- [x] 2.1 鏂板缓 `frontend/src/render.ts`锛氬壀鍒囨覆鏌撳嚱鏁帮紙renderSummary/renderToolList/renderPanelTabs/filteredOrphans/renderOrphanView/renderDetail/subRows/subIdsOf/selectedItems/renderBellPanel/renderTrendChart/renderGrowers + COLUMNS/KIND_TONE/SUB_CAP/TRASH_ICON/RESTORE_ICON锛?- [ ] 2.2 `renderDetail` 鐨勫嵏杞藉洖璋冩敼涓?`uninstallHandler?.(tool.name)`锛圖2 鍥炶皟娉ㄥ唽锛夛紱`export let uninstallHandler: ((name: string) => void) | null = null`
- [x] 2.3 main.ts 鍒犻櫎宸茬Щ鍔ㄥ嚱鏁帮紝`import * as render from './render'`
- [x] 2.4 楠岃瘉锛歚npx tsc --noEmit` + `npm test` + `npm run build`
- [x] 2.5 鎻愪氦锛歚refactor(frontend): extract render functions into render module`

## 3. flows.ts + menu.ts + main.ts 鐦﹁韩

- [x] 3.1 鏂板缓 `frontend/src/flows.ts`锛氬壀鍒囦笟鍔℃祦绋嬶紙trashPaths/confirmDialog/showOrphanConfirm/showConfirmModal/refreshTrashInfo/openTrashPanel/refreshTrashList/鏇存柊娴佺▼鍏ㄩ儴/鍗歌浇娴佺▼鍏ㄩ儴/openPrefs/refreshReminder/openTrends/refreshTrends/openAbout/rescan/setScanning锛?- [ ] 3.2 鏂板缓 `frontend/src/menu.ts`锛氬壀鍒?closeMenus/initMenuBar锛堜緷璧?flows 鐨?openPrefs/manualCheck锛?- [ ] 3.3 main.ts 鐦﹁韩锛氫繚鐣?import 姹囨€?+ systemDark/applyTheme + resolveLocale/initI18n/applyI18n + init() 鎺ョ嚎锛沬nit() 琛?`render.uninstallHandler = (n) => flows.startUninstall(n);`
- [x] 3.4 楠岃瘉锛歚npx tsc --noEmit`锛堥噸鐐癸細鏃犲惊鐜緷璧栨姤閿欙級+ `npm test` + `npm run build`
- [x] 3.5 鐪熸満鍐掔儫锛坮elease-process.md 绗?7 姝ユ竻鍗曪級锛氭壂鎻?杩囨护/鎺掑簭/璇︽儏鍕鹃€夋竻鐞嗭紱鍥炴敹绔欐墦寮€-鎭㈠-娓呯┖锛堢‘璁ゅ脊绐楁牱寮忎笌灞傜骇锛夛紱鏇存柊闈㈡澘鎵嬪姩妫€鏌?涓嬭浇鍙栨秷-澶辫触闈㈡澘淇濈暀锛涘嵏杞借捣濮嬩俊鎭?浠ｈ窇-娈嬬暀鍒楄〃锛涢閫夐」璇█鍗虫椂鍒囨崲锛沇indows 鑿滃崟鏉′笅鎷夛紱**閲嶇偣楠岃瘉浜嬩欢涓嶉噸澶嶇粦瀹?*锛堝脊绐楀紑-鍏?鍐嶅紑锛屾搷浣滀笉鍙犲姞锛?- [ ] 3.6 鎻愪氦锛歚refactor(frontend): extract flows and menu modules; slim main.ts to wiring`

## 4. 鏀跺熬楠岃瘉

- [x] 4.1 鍏ㄩ噺锛歚gofmt -l .` + `go vet ./...` + `go test ./... -cover` + `(cd frontend && npm test && npm run build)` + 涓夊钩鍙?`GOOS=windows|linux|darwin go build ./...`
- [x] 4.2 GUI 鍐掔儫锛堢湡鏈猴紝蹇呴』锛夛細鎸?release-process.md 娓呭崟閫愰」锛涜嫢鏈変慨澶嶏紝杩藉姞鎻愪氦 `fix(frontend): smoke-test fixes after module split`

# 鎵弿澧為噺浼樺寲锛坢time 鎸囩汗鍙樻洿妫€娴嬶級

> 鍙樻洿鐩綍锛?026-08-15-scan-change-detection锛涜兘鍔涳細scan-change-detection锛堟柊锛?
> **鎵ц鐜锛堝繀璇伙級**锛氭湰浠撳簱鍦?DSH 娌欑涓嬪伐浣滐紝鎵€鏈?`pwsh` 鍛戒护锛坓o test / gofmt / go vet / git / npx / npm 绛夛級褰撳墠琚矙绠辨嫤鎴紙`SetNamedSecurityInfoW failed: grantWrite`锛夈€傚疄鏂芥椂姣忎竴姝ラ獙璇?鎻愪氦鍛戒护缁?`pwsh` 鎵ц浼氬脊鍑烘巿鏉冭姹傗€斺€旀壒鍑?`workspace-write`锛堜釜鍒懡浠ら渶 `danger-full-access`锛夊嵆鍙斁琛岋紱鎺堟潈涓轰細璇濈骇锛屾壒鍑嗗悗鍚庣画鍛戒护閫氬父涓嶅啀閲嶅寮圭獥銆傛湰鍙樻洿鏃犳枃浠跺垹闄ゆ搷浣溿€?
## 1. 鎸囩汗鏍稿績锛圱DD锛?
- [x] 1.1 鍐欏け璐ユ祴璇?`internal/scanner/fingerprint_test.go`锛歍estComputeFingerprintAndEqual锛堜复鏃剁洰褰曟瀯閫犵粨鏋滐紱绛夊€?椤哄簭鏃犲叧/鏉＄洰缂哄け/mtime 鍙樺寲/size 鍙樺寲/鐪熷疄 stat 涓€鑷存€э級
- [x] 1.2 杩愯纭澶辫触锛坲ndefined: ComputeFingerprint锛?- [ ] 1.3 瀹炵幇 `internal/scanner/fingerprint.go`锛欶ingerprintEntry / statEntry / measurePaths / ComputeFingerprint锛堟帓搴忥級/ FingerprintsEqual
- [x] 1.4 杩愯纭閫氳繃 + gofmt 骞插噣
- [x] 1.5 鎻愪氦锛歚feat(scanner): add mtime-based fingerprint for change detection`

## 2. 鎸囩汗鎸佷箙鍖栵紙TDD锛?
- [x] 2.1 杩藉姞澶辫触娴嬭瘯 `cache_test.go`锛歍estFingerprintRoundTrip锛堥殧绂?XDG_CACHE_HOME + LOCALAPPDATA锛汼ave/Load 寰€杩旓紱ClearCache 鑱斿姩娓呮寚绾癸級
- [x] 2.2 杩愯纭澶辫触锛坲ndefined: SaveFingerprint锛?- [ ] 2.3 瀹炵幇 `cache.go`锛氭娊 `writeJSONAtomic`锛圫aveCache 鏀圭敤瀹冿級锛沗SaveFingerprint`/`LoadFingerprint`锛坙ast-scan.fp.json锛夛紱`ClearCache` 杩炲甫娓呮寚绾?- [ ] 2.4 杩愯纭閫氳繃锛堝惈鏃㈡湁 cache 娴嬭瘯锛? gofmt 骞插噣
- [x] 2.5 鎻愪氦锛歚feat(scanner): persist fingerprint file, clear it with cache --clear`

## 3. ScanIfUnchanged锛圱DD锛?
- [x] 3.1 鍐欏け璐ユ祴璇?`internal/scanner/unchanged_test.go`锛歍estScanIfUnchangedCacheHit锛堥缃紦瀛?鎸囩汗锛岃繑鍥炵紦瀛樹笖 ScannedAt 涓嶅彉锛夛紱TestScanIfUnchangedNoFingerprintFallsBack锛堟棤鎸囩汗璧板叏閲忥紝ScannedAt 鍙樺寲锛?- [ ] 3.2 杩愯纭澶辫触锛坲ndefined: ScanIfUnchanged锛?- [ ] 3.3 瀹炵幇 `scanner.go`锛歚Scan` 涓讳綋鎶戒负鍐呴儴 `scan(opts, skipIfUnchanged)`锛涙柊澧?`ScanIfUnchanged`锛涚紦瀛樺啓鍏ュ杩藉姞 `SaveFingerprint(ComputeFingerprint(cached))`锛坈ached 涓烘湭杩囨护缁撴灉锛?- [ ] 3.4 杩愯纭閫氳繃锛堝惈鏃㈡湁 scan/classify/attribute 娴嬭瘯锛? gofmt 骞插噣
- [x] 3.5 鎻愪氦锛歚feat(scanner): ScanIfUnchanged skips full rescan when fingerprint is unchanged`

## 4. GUI 鎺ュ叆

- [x] 4.1 `gui/service.go` 鏂板 `ScanIfChanged()`锛氬唴閮?`scanner.ScanIfUnchanged`锛涗粎鐪熷疄鎵弿锛坧rev==nil 鎴?ScannedAt 鍙樺寲锛夋墠 history.Record + probeAll锛涗簨浠跺绾︿笌 Scan 涓€鑷?- [ ] 4.2 閲嶆柊鐢熸垚 wails 缁戝畾锛坄wails generate module`锛夛紱纭 `frontend/wailsjs/go/gui/ScannerService.js` 涓?`.d.ts` 鍚?ScanIfChanged锛涙棤娉曡窇 wails 鏃舵墜宸ヨˉ鍚屾瀯鏉＄洰
- [x] 4.3 `frontend/src/main.ts`锛歩mport 鍔?ScanIfChanged锛沗init()` 鏈熬 `rescan()` 鈫?`ScanIfChanged()`锛況escanBtn onclick 涓嶅彉锛堝己鍒跺叏閲忥級
- [x] 4.4 楠岃瘉锛歚npx tsc --noEmit` + `npm test` + `go vet ./gui/` + `go build ./...`
- [x] 4.5 鐪熸満鍐掔儫锛氶鍚叏閲忊啋鍐嶅惎绉掑紑锛堟棤鍏ㄩ噺 IO/鏃犳壂鎻忛棯鐑侊級锛涙墜鍔ㄩ噸鎵粛鍏ㄩ噺锛涘畨瑁?鍒犻櫎宸ュ叿鍚庡惎鍔ㄨ嚜鍔ㄥ叏閲?- [ ] 4.6 鎻愪氦锛歚feat(gui): startup scan uses change detection, manual rescan stays full`

## 5. CLI 鎺ュ叆

- [x] 5.1 `internal/cli/scan.go`锛歚--refresh` 璧?Scan锛涢潪 `--refresh` 璧?ScanIfUnchanged锛坄--no-cache` 鏃剁洿鎺?Scan锛夛紱鍘嗗彶浠呯湡瀹炴壂鎻忚拷鍔?- [ ] 5.2 楠岃瘉锛歚go build ./...` + `go test ./internal/cli/` + 鎵嬪姩涓夋锛堥鎵啋绉掑洖鈫抰ouch 瑙﹀彂閲嶆壂锛?- [ ] 5.3 鎻愪氦锛歚feat(cli): scan without --refresh auto-rescans when fingerprint changed`

## 6. 鏂囨。

- [x] 6.1 README.md / README.zh-CN.md锛欿nown limitations 杩藉姞鎸囩汗鐩插尯璇存槑锛沗scan` 琛屾敞閲婃洿鏂帮紙auto-rescans when files changed锛?- [ ] 6.2 鎻愪氦锛歚docs: document fingerprint-based skip of unchanged scans`

# 浠撳簱鍗敓锛歊EADME 婕傜Щ銆佹鍙傛暟銆佸彉鏇村綊妗?
> 鍙樻洿鐩綍锛?026-08-15-repo-hygiene锛涙棤鑳藉姏鍙樻洿

> **鎵ц鐜锛堝繀璇伙級**锛氭湰浠撳簱鍦?DSH 娌欑涓嬪伐浣滐紝鎵€鏈?`pwsh` 鍛戒护锛坓it mv / go test / gofmt 绛夛級褰撳墠琚矙绠辨嫤鎴紙`SetNamedSecurityInfoW failed: grantWrite`锛夈€傚疄鏂芥椂姣忎竴姝ラ獙璇?鎻愪氦鍛戒护缁?`pwsh` 鎵ц浼氬脊鍑烘巿鏉冭姹傗€斺€旀壒鍑?`workspace-write` 鍗冲彲鏀捐锛涙巿鏉冧负浼氳瘽绾с€傜 3 姝ュ綊妗ｇ敤 `git mv` 鎴?`openspec archive`锛堥渶 pwsh锛夈€?
## 1. README 淇

- [x] 1.1 `README.md`锛氭埅鍥捐〃鏍硷紙app-light-en.png/app-dark-en.png锛夋浛鎹负鍗犱綅娉ㄩ噴锛沀sage 琛?`uninstall <tool>` 涓ゆ潯锛涚増鏈ず渚?0.3.3 鈫?0.3.8
- [x] 1.2 `README.zh-CN.md`锛氬悓姝ワ紙鎴浘 app-light.png/app-dark.png 鍗犱綅銆乽ninstall 涓ゆ潯銆佺増鏈ず渚嬶級
- [x] 1.3 楠岃瘉锛歚grep -rn "docs/screenshots" README.md README.zh-CN.md` 鏃犺緭鍑猴紱`grep -n "uninstall <tool>" README.md` 鏈夎緭鍑?- [ ] 1.4 鎻愪氦锛歚docs: fix broken screenshots, add uninstall to CLI list, refresh version example`

## 2. 绉婚櫎 -full 姝诲弬鏁?
- [x] 2.1 `internal/cli/scan.go`锛氬垹 `full := fs.Bool(...)`锛?21锛夛紱`Options{Full: *full, ...}` 鏀?`Options{NoCache: *noCache, ToolFilter: filters}`锛?42锛?- [ ] 2.2 `internal/scanner/types.go`锛歄ptions 鍒?`Full bool` 瀛楁涓庢敞閲?- [ ] 2.3 `internal/scanner/scanner.go`锛歚:109` `Options{Full: opts.Full}` 鏀?`Options{}`
- [x] 2.4 楠岃瘉锛歚gofmt -l .` 绌?+ `go vet ./...` + `go test ./internal/scanner/ ./internal/cli/` 鍏ㄧ豢 + `grep -rn "\.Full\|Full:" --include="*.go" internal/` 鏃犺緭鍑?- [ ] 2.5 鎻愪氦锛歚refactor: remove dead -full flag (unattributed dirs are always measured)`

## 3. OpenSpec 鍙樻洿褰掓。

- [x] 3.1 褰掓。 5 涓凡瀹屾垚鍙樻洿锛堜紭鍏?`openspec archive <slug>`锛屽惁鍒?`git mv` 鍒?`openspec/changes/archive/`锛夛細2026-08-14-fix-orphan-gui-data / windows-test-portability / git-family-merge / npm-global-shim-attribution / cleanup-ui-tweaks
- [x] 3.2 楠岃瘉锛歚ls openspec/changes/` 涓虹┖锛堟棤娲昏穬鍙樻洿锛涙湰鎵瑰叾浣?4 涓?change 鍦ㄦ墽琛屾椂澶勪簬杩涜涓紝涓嶅綊妗ｏ級
- [x] 3.3 鎻愪氦锛歚chore(openspec): archive 5 completed changes`

# 鍙戝竷鑴氭湰鍥哄寲

> 鍙樻洿鐩綍锛?026-08-15-release-tooling锛涙棤鑳藉姏鍙樻洿锛堜粨搴撳伐鍏?娴佺▼锛?
> **鎵ц鐜锛堝繀璇伙級**锛氭湰浠撳簱鍦?DSH 娌欑涓嬪伐浣滐紝鎵€鏈?`pwsh` 鍛戒护锛坆ash / node / git 绛夛級褰撳墠琚矙绠辨嫤鎴紙`SetNamedSecurityInfoW failed: grantWrite`锛夈€傚疄鏂芥椂姣忎竴姝ラ獙璇?鎻愪氦鍛戒护缁?`pwsh` 鎵ц浼氬脊鍑烘巿鏉冭姹傗€斺€旀壒鍑?`workspace-write` 鍗冲彲鏀捐锛涙巿鏉冧负浼氳瘽绾с€傛湰鍙樻洿鍚?*鍒犻櫎鏂囦欢**鎿嶄綔锛堢 3.4 姝ュ垹闄?`.go-toolchain/` 涓存椂鏂囦欢锛岄渶 `pwsh Remove-Item`锛屽悓鏍烽渶瑕佹巿鏉冿級銆?
## 1. check.sh

- [x] 1.1 鍒涘缓 `scripts/release/check.sh`锛氳皟鐢?`./scripts/test-all.sh` + `(cd frontend && npx tsc --noEmit)` + 涓夊钩鍙?`GOOS=windows|linux|darwin go build ./...` + 锛堝彲閫?tag 鍙傛暟锛塯rep 鏍￠獙 wails.json productVersion
- [x] 1.2 楠岃瘉锛歚chmod +x`锛涙湰鍦?`./scripts/release/check.sh` 鍏ㄩ儴閫氳繃
- [x] 1.3 鎻愪氦锛歚chore(release): add pre-release check script (checklist automation)`

## 2. bump-version.sh

- [x] 2.1 鍒涘缓 `scripts/release/bump-version.sh`锛氱増鏈彿姝ｅ垯鏍￠獙锛沗sed -i.bak` 鏀?wails.json productVersion锛涘啓鍚庡洖璇?grep 鏍￠獙锛涘垹 .bak
- [x] 2.2 楠岃瘉锛歚./scripts/release/bump-version.sh 0.3.9` 鏀圭増 鈫?鍐嶆敼鍥?`0.3.8` 杩樺師锛涢潪娉曠増鏈彿锛堝 `abc`锛夋嫆缁?- [ ] 2.3 鎻愪氦锛歚chore(release): add version bump script (single source in wails.json)`

## 3. notes.js + 妯℃澘

- [x] 3.1 鍒涘缓 `scripts/release/notes-template.md`锛氫腑鑻卞弻璇鏋讹紙鏍囬/鍙樻洿鍒嗚妭/涓嬭浇浜х墿閫愰」/鏈鍚嶆彁绀?`---` 鍒嗛殧锛?- [ ] 3.2 鍒涘缓 `scripts/release/notes.js`锛氭寜 tag 鏌?release id锛圙ET /releases/tags/<tag>锛夛紱鎻愪氦鍓嶈嚜妫€锛圕JK + `---` + 涓嬭浇浜х墿娓呭崟锛屼换涓€涓嶆弧瓒虫嫆缁濓級锛汸ATCH body锛坉raft=false锛夛紱鎻愪氦鍚庡洖璇绘牎楠?body 鍚?CJK锛沗--verify` 浠呮湰鍦拌嚜妫€锛汫H_PAT 璁よ瘉
- [x] 3.3 楠岃瘉锛氱敤妯℃澘澶嶅埗 `docs/release-notes/v0.3.9.md` 璺?`node scripts/release/notes.js v0.3.9 --verify` 閫氳繃锛涘垹闄ゆ祴璇曟枃浠?- [ ] 3.4 鍒犻櫎 `.go-toolchain/fix-release.js`銆乣verify-body.txt`銆乣current-body.txt`銆乣release-notes-v0.3.8.md`
- [x] 3.5 鎻愪氦锛歚chore(release): formalize release notes submit/verify script (migrate from .go-toolchain)`

## 4. release-process.md 鏇存柊

- [x] 4.1 鏇存柊 `docs/release-process.md`锛氱 1 姝?鈫?check.sh锛涚 2 姝?鈫?bump-version.sh锛涚 5 姝?鈫?notes.js 娴佺▼锛堝惈 --verify锛夛紱GUI 鍐掔儫娓呭崟淇濈暀
- [x] 4.2 楠岃瘉锛歚grep -n "scripts/release" docs/release-process.md` 涓夊寮曠敤榻愬叏
- [x] 4.3 鎻愪氦锛歚docs: point release runbook at the new scripts`

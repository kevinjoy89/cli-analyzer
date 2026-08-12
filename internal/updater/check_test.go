package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/config"
)

// checkEnv 将 buildinfo.Version、API 地址与配置目录隔离到可控状态。
func checkEnv(t *testing.T, version string, releases []Release) func() {
	t.Helper()
	origVer := buildinfo.Version
	buildinfo.Version = version
	t.Cleanup(func() { buildinfo.Version = origVer })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	origAPI := APIBaseURL
	APIBaseURL = srv.URL
	t.Cleanup(func() { APIBaseURL = origAPI })

	restore := config.SetDataRoot(t.TempDir())
	t.Cleanup(restore)
	return srv.Close
}

func sampleAssets(version string) []ReleaseAsset {
	return []ReleaseAsset{
		{Name: "CLI-Analyzer-" + version + "-darwin-arm64.dmg", BrowserDownloadURL: "https://x.dmg", Size: 10},
	}
}

func TestCheckFindsUpdate(t *testing.T) {
	checkEnv(t, "0.2.3", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0", Assets: sampleAssets("0.3.0")},
	})
	res := CheckForUpdates(context.Background(), false)
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if !res.UpdateAvailable {
		t.Fatal("UpdateAvailable = false, want true")
	}
	if res.Latest != "0.3.0" {
		t.Errorf("Latest = %q, want 0.3.0", res.Latest)
	}
	if res.Current != "0.2.3" {
		t.Errorf("Current = %q, want 0.2.3", res.Current)
	}
}

func TestCheckUpToDate(t *testing.T) {
	checkEnv(t, "0.3.0", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"},
	})
	res := CheckForUpdates(context.Background(), false)
	if res.UpdateAvailable {
		t.Fatal("UpdateAvailable = true, want false when up to date")
	}
}

func TestCheckUsesCacheWithin24h(t *testing.T) {
	checkEnv(t, "0.2.3", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0", Assets: sampleAssets("0.3.0")},
	})
	first := CheckForUpdates(context.Background(), false)
	if !first.UpdateAvailable {
		t.Fatal("first check should find update")
	}
	// 第二次自动检查：命中缓存（Cached=true），不应再次请求
	second := CheckForUpdates(context.Background(), false)
	if !second.Cached {
		t.Fatal("second auto check should hit cache")
	}
	if !second.UpdateAvailable || second.Latest != "0.3.0" {
		t.Errorf("cached result lost: %+v", second)
	}
	// 手动检查（force=true）应绕过缓存
	third := CheckForUpdates(context.Background(), true)
	if third.Cached {
		t.Fatal("manual check must bypass cache")
	}
}

func TestCheckIgnoredVersionSuppresses(t *testing.T) {
	checkEnv(t, "0.2.3", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0", Assets: sampleAssets("0.3.0")},
	})
	cfg := config.Load()
	cfg.Update.IgnoredVersion = "0.3.0"
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	res := CheckForUpdates(context.Background(), false)
	if res.UpdateAvailable {
		t.Fatal("UpdateAvailable = true despite ignored version")
	}
	// 出现更新版本后恢复提示
	cfg.Update.IgnoredVersion = "0.3.0"
	// 手动检查绕过缓存，但忽略逻辑仍生效——用更新版本的 release 验证
	restore := checkEnv(t, "0.2.3", []Release{
		{TagName: "v0.4.0", HTMLURL: "https://github.com/x/releases/tag/v0.4.0", Assets: sampleAssets("0.4.0")},
	})
	defer restore()
	cfg = config.Load()
	cfg.Update.IgnoredVersion = "0.3.0"
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	res = CheckForUpdates(context.Background(), true)
	if !res.UpdateAvailable || res.Latest != "0.4.0" {
		t.Fatalf("update above ignored version should be offered: %+v", res)
	}
}

func TestCheckDevVersionSkipsAuto(t *testing.T) {
	checkEnv(t, "dev", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"},
	})
	res := CheckForUpdates(context.Background(), false)
	if res.UpdateAvailable || res.Error != "" {
		t.Fatalf("dev auto check should silently no-op, got %+v", res)
	}
	// 手动检查：明确报错
	res = CheckForUpdates(context.Background(), true)
	if !strings.Contains(res.Error, "无法确定当前版本") {
		t.Errorf("dev manual check error = %q", res.Error)
	}
}

func TestCheckNetworkFailureDoesNotCache(t *testing.T) {
	checkEnv(t, "0.2.3", []Release{})
	// 关闭 mock 服务器 → 请求失败
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	origAPI := APIBaseURL
	APIBaseURL = srv.URL
	srv.Close() // 立即关闭，制造连接失败
	t.Cleanup(func() { APIBaseURL = origAPI })

	res := CheckForUpdates(context.Background(), false)
	if res.Error == "" {
		t.Fatal("want error on network failure")
	}
	cfg := config.Load()
	if cfg.Update.LastCheckAt != "" {
		t.Error("failed check must not update lastCheckAt cache")
	}
	_ = time.Now()
}

// 回归：升级到最新版后，24h 缓存命中时“是否有更新”必须按新当前版本重算——
// 否则升级后重新打开会继续提示 “v0.3.2 → v0.3.2”。
func TestCacheRecomputesUpdateAfterUpgrade(t *testing.T) {
	restore := checkEnv(t, "0.2.3", []Release{
		{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0", Assets: sampleAssets("0.3.0")},
	})
	defer restore()
	first := CheckForUpdates(context.Background(), false)
	if !first.UpdateAvailable {
		t.Fatal("0.2.3 应发现有更新")
	}
	// 模拟升级到 0.3.0（构建期注入的新版本）
	buildinfo.Version = "0.3.0"
	second := CheckForUpdates(context.Background(), false)
	if !second.Cached {
		t.Fatal("应命中缓存")
	}
	if second.UpdateAvailable {
		t.Errorf("升级后缓存命中不应再提示更新: %+v", second)
	}
	if second.Latest != "0.3.0" {
		t.Errorf("Latest = %q, want 0.3.0", second.Latest)
	}
}

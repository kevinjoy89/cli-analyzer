package updater

import (
	"context"
	"encoding/json"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"
)

// cacheInterval 是自动检查的限流窗口：距上次成功检查不足该时长时复用缓存结果，
// 避免触发 GitHub 未认证接口的 60 req/h/IP 限流（design D3）。
const cacheInterval = 24 * time.Hour

// CheckResult 是一次更新检查的完整结果，GUI/CLI 以 JSON 序列化后输出。
type CheckResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	AssetName       string `json:"assetName,omitempty"`
	DownloadURL     string `json:"downloadURL,omitempty"`
	ReleaseURL      string `json:"releaseURL,omitempty"`
	InstallSource   string `json:"installSource"`
	Error           string `json:"error,omitempty"`
	// Cached 表示结果来自限流缓存而非本次网络请求（供日志/调试）。
	Cached bool `json:"cached,omitempty"`
}

// CheckForUpdates 执行一次更新检查。
//
// force=false（自动检查）：命中 24h 缓存时复用缓存结果；网络失败返回带 Error
// 的结果（调用方决定是否静默）；源码构建（Version=="dev"）直接跳过。
// force=true（手动检查）：不受缓存限制；Version=="dev" 时报“无法确定当前版本”。
func CheckForUpdates(ctx context.Context, force bool) CheckResult {
	cfg := config.Load()
	exePath, _ := executablePath()
	res := CheckResult{
		Current:       buildinfo.Version,
		InstallSource: ResolveInstallSource(exePath),
	}

	if buildinfo.Version == "dev" {
		if force {
			res.Error = i18n.T("err.updaterDevVersion")
		}
		return res
	}

	if !force {
		if cached, ok := cachedResult(cfg, res); ok {
			return cached
		}
	}

	release, err := LatestRelease(ctx, nil)
	if err != nil {
		res.Error = err.Error()
		return res // 不写缓存：失败不刷新 lastCheckAt，下次仍会重试
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	res.Latest = latest
	res.ReleaseURL = release.HTMLURL
	if res.ReleaseURL == "" {
		res.ReleaseURL = ReleaseURL()
	}

	// 忽略版本语义：用户忽略 vX 后，在出现比 vX 更新的版本前不提示（spec: 忽略版本）。
	if ignored := cfg.Update.IgnoredVersion; ignored != "" {
		if iv, err := ParseVersion(ignored); err == nil {
			if lv, err := ParseVersion(latest); err == nil && lv.Compare(iv) <= 0 {
				saveCache(cfg, res)
				return res
			}
		}
	}

	if lv, err := ParseVersion(latest); err == nil {
		if cv, err := ParseVersion(buildinfo.Version); err == nil && lv.Compare(cv) > 0 {
			res.UpdateAvailable = true
			if asset, aerr := SelectAsset(release, goruntime.GOOS, goruntime.GOARCH, res.InstallSource); aerr == nil {
				res.AssetName = asset.Name
				res.DownloadURL = asset.BrowserDownloadURL
			}
			// asset 匹配不到（如安装来源 unknown）：仍提示有更新，但无下载入口，
			// 前端展示 Release 页链接（design D6 兜底）。
		}
	}

	saveCache(cfg, res)
	return res
}

// cachedResult 在限流窗口内复用上次结果；仅网络相关字段来自缓存，
// Current/InstallSource 等运行时字段始终取当前值。
func cachedResult(cfg *config.Config, fresh CheckResult) (CheckResult, bool) {
	if cfg.Update.LastCheckAt == "" || cfg.Update.LastResult == "" {
		return CheckResult{}, false
	}
	t, err := time.Parse(time.RFC3339, cfg.Update.LastCheckAt)
	if err != nil || time.Since(t) >= cacheInterval {
		return CheckResult{}, false
	}
	var cached struct {
		Latest          string `json:"latest"`
		UpdateAvailable bool   `json:"updateAvailable"`
		AssetName       string `json:"assetName"`
		DownloadURL     string `json:"downloadURL"`
		ReleaseURL      string `json:"releaseURL"`
	}
	if err := json.Unmarshal([]byte(cfg.Update.LastResult), &cached); err != nil {
		return CheckResult{}, false
	}
	fresh.Latest = cached.Latest
	fresh.UpdateAvailable = cached.UpdateAvailable
	fresh.AssetName = cached.AssetName
	fresh.DownloadURL = cached.DownloadURL
	fresh.ReleaseURL = cached.ReleaseURL
	fresh.Cached = true
	return fresh, true
}

// saveCache 记录检查时间与网络相关结果，供限流窗口内复用。
func saveCache(cfg *config.Config, res CheckResult) {
	cfg.Update.LastCheckAt = time.Now().UTC().Format(time.RFC3339)
	cached := struct {
		Latest          string `json:"latest"`
		UpdateAvailable bool   `json:"updateAvailable"`
		AssetName       string `json:"assetName"`
		DownloadURL     string `json:"downloadURL"`
		ReleaseURL      string `json:"releaseURL"`
	}{res.Latest, res.UpdateAvailable, res.AssetName, res.DownloadURL, res.ReleaseURL}
	if b, err := json.Marshal(cached); err == nil {
		cfg.Update.LastResult = string(b)
	}
	_ = config.Save(cfg)
}

// executablePath 返回当前可执行文件路径；失败时返回空串。
var executablePath = func() (string, error) {
	return os.Executable()
}

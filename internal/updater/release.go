package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cli-analyzer/internal/i18n"
)

// repoOwner / repoName 是发布所在的 GitHub 仓库。
const (
	repoOwner = "kevinjoy89"
	repoName  = "cli-analyzer"
)

// APIBaseURL 可被测试替换为 httptest 服务器地址；生产保持 GitHub 官方地址。
var APIBaseURL = "https://api.github.com"

// defaultClient 供 API 类请求使用（查询版本）；测试可整体替换。
// 慢网络（如国内 → GitHub）下 15s 连响应头都可能等不到，放宽到 60s。
var defaultClient = &http.Client{Timeout: 60 * time.Second}

// downloadClient 专供大文件/校验和下载：不做总超时限制——慢网络下 dmg 可能远超
// 15s，过早超时会让安装包永远下载不完。取消依赖调用方 context。
var downloadClient = &http.Client{}

// ReleaseAsset 是 GitHub Release 的一个附件。
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Release 是 GitHub Release 的摘要（仅取更新所需字段）。
type Release struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	HTMLURL    string         `json:"html_url"`
	Assets     []ReleaseAsset `json:"assets"`
}

// LatestRelease 查询仓库最新正式发布：过滤 draft 与 prerelease，
// 取列表中第一个正式版（GitHub 按创建时间倒序返回）。
// 选择列表接口而非 /releases/latest：后者不返回 draft 但会返回 prerelease，过滤不完整。
func LatestRelease(ctx context.Context, client *http.Client) (*Release, error) {
	if client == nil {
		client = defaultClient
	}
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=10", APIBaseURL, repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// 显式声明版本，避免未来 API 变更影响
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("err.updateQuery", map[string]any{"err": err}))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		// 403 通常是未认证接口限流（60 次/时/IP），给用户更友好的提示而非裸状态码
		return nil, fmt.Errorf("%s", i18n.T("err.updateRateLimited"))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", i18n.T("err.updateApiStatus", map[string]any{"status": resp.Status}))
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("%s", i18n.T("err.updateDecode", map[string]any{"err": err}))
	}
	for i := range releases {
		if releases[i].Draft || releases[i].Prerelease {
			continue
		}
		return &releases[i], nil
	}
	return nil, fmt.Errorf("%s", i18n.T("err.updateNoStable"))
}

// ReleaseURL 返回仓库 Releases 页面地址（兜底入口，供"打开 Release 页面"使用）。
func ReleaseURL() string {
	return fmt.Sprintf("https://github.com/%s/%s/releases", repoOwner, repoName)
}

// AssetByChecksumName 查找名为 checksums.txt 的附件。
func (r *Release) AssetByChecksumName() *ReleaseAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == "checksums.txt" {
			return &r.Assets[i]
		}
	}
	return nil
}

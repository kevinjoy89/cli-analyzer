package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"cli-analyzer/internal/i18n"
)

// ProgressFunc 报告下载进度（downloaded 为已下载字节，total 为总字节）。
// total 在服务端未提供 Content-Length 时使用 release asset 的 size 兜底。
type ProgressFunc func(downloaded, total int64)

// downloadsDir 是下载目标目录，默认为 ~/Downloads；测试可替换为临时目录。
var downloadsDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(home, "Downloads")
}

// Download 将 url 流式下载到 <downloadsDir>/<name>。
// 先写 <name>.part，完成后原子 rename 去掉后缀；取消或失败时删除 .part，
// 保证目标目录不残留半截文件。返回最终文件路径。
func Download(ctx context.Context, client *http.Client, url, name string, sizeHint int64, progress ProgressFunc) (string, error) {
	if client == nil {
		// 下载不设总超时（见 release.go 的 downloadClient 注释）；取消由 context 驱动
		client = downloadClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": err}))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": "HTTP " + resp.Status}))
	}

	total := resp.ContentLength
	if total <= 0 {
		total = sizeHint
	}
	if err := os.MkdirAll(downloadsDir(), 0o755); err != nil {
		return "", fmt.Errorf("download: create dir: %w", err)
	}
	part := filepath.Join(downloadsDir(), name+".part")
	final := filepath.Join(downloadsDir(), name)

	f, err := os.Create(part)
	if err != nil {
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": err}))
	}
	cleanup := func() {
		f.Close()
		os.Remove(part)
	}

	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				cleanup()
				return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": werr}))
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			cleanup()
			return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": rerr}))
		}
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": err}))
	}
	if err := f.Close(); err != nil {
		os.Remove(part)
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": err}))
	}
	if err := os.Rename(part, final); err != nil {
		os.Remove(part)
		return "", fmt.Errorf("%s", i18n.T("err.updateDownload", map[string]any{"err": err}))
	}
	return final, nil
}

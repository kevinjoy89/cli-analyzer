package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"cli-analyzer/internal/i18n"
)

// checksumAssetName 是发布流程生成的校验和文件名（见 design D10 / release.yml）。
const checksumAssetName = "checksums.txt"

// FetchChecksums 下载 release 附带的 checksums.txt 内容。
// 发布未附带（历史版本）时返回 os.ErrNotExist，调用方按"校验缺失"处理。
func FetchChecksums(ctx context.Context, client *http.Client, r *Release) ([]byte, error) {
	asset := r.AssetByChecksumName()
	if asset == nil {
		return nil, os.ErrNotExist
	}
	if client == nil {
		// 校验和抓取位于“下载成功之后”的关键路径：慢网络下 15s 超时会把
		// 已完成的下载整个浪费掉（等待响应头超时）。与下载一致：不设总超时，
		// 取消由调用方 context 驱动。
		client = downloadClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("err.updateFetchChecksums", map[string]any{"err": err}))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", i18n.T("err.updateFetchChecksums", map[string]any{"err": "HTTP " + resp.Status}))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB 上限，防御异常文件
}

// ParseChecksums 解析 sha256sum 输出格式的 checksums 文件，
// 返回 map[文件名]十六进制哈希。
func ParseChecksums(content []byte) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// sha256sum 可能输出 "  filename"（binary 模式）或 " filename"
		name := strings.TrimPrefix(fields[1], "*")
		out[name] = strings.ToLower(fields[0])
	}
	return out
}

// VerifyChecksum 校验 filePath 的 SHA256 与 checksums 中 assetName 的条目一致。
// 返回的错误区分两类：条目缺失（os.ErrNotExist）与哈希不匹配。
func VerifyChecksum(filePath, assetName string, checksums map[string]string) error {
	want, ok := checksums[assetName]
	if !ok {
		return fmt.Errorf("%s: %w", i18n.T("err.updateChecksumMissing", map[string]any{"name": assetName}), os.ErrNotExist)
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("%s", i18n.T("err.updateChecksumMismatch", map[string]any{"name": assetName, "got": got, "want": want}))
	}
	return nil
}

package updater

import (
	"context"
	"net/http"
)

// DownloadInstaller 下载 release 中指定产物并校验 SHA256，返回最终文件路径。
//
// 校验顺序：下载 → 拉取该 release 的 checksums.txt → 比对哈希。
// checksums.txt 缺失（历史发布）或哈希不匹配时返回错误（可分别用
// errors.Is(err, os.ErrNotExist) 与错误信息区分），调用方必须按"校验缺失/
// 失败 → 不提供安装入口"处理（design D5，安全优先：宁可不给入口，不跳过校验）。
// 校验失败时已下载的文件保留在目标目录（是否删除由调用方决定）。
func DownloadInstaller(ctx context.Context, client *http.Client, release *Release, asset *ReleaseAsset, progress ProgressFunc) (string, error) {
	path, err := Download(ctx, client, asset.BrowserDownloadURL, asset.Name, asset.Size, progress)
	if err != nil {
		return "", err
	}
	checksums, err := FetchChecksums(ctx, client, release)
	if err != nil {
		return path, err
	}
	m := ParseChecksums(checksums)
	if err := VerifyChecksum(path, asset.Name, m); err != nil {
		return path, err
	}
	return path, nil
}

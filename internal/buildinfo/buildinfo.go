// Package buildinfo 承载构建期注入的版本与安装来源信息。
//
// 这些值由发布流程通过 -ldflags "-X" 注入（见 .github/workflows/release.yml），
// 本地 go build / go run 时保持默认值。它是版本号的唯一事实来源：
// CLI、GUI、安装包元数据均从这里派生，避免多处方手写导致漂移。
package buildinfo

// Version 是应用版本号，构建期注入，如：
//
//	go build -ldflags "-X cli-analyzer/internal/buildinfo.Version=0.3.0" .
//
// 未注入时为 "dev"，表示源码构建（此时跳过自动更新比较）。
var Version = "dev"

// InstallSource 标识当前二进制以何种安装方式分发，构建期注入：
//
//	darwin:  dmg
//	windows: installer | portable
//	linux:   deb | tarball
//
// 未注入时为 "unknown"，更新时依次尝试运行时探测（如 dpkg -S），
// 仍无法确定则打开 Release 页面由用户自行选择，绝不猜测产物。
var InstallSource = "unknown"

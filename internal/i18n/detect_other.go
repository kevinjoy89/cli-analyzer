//go:build !darwin && !windows && !linux

package i18n

// DetectSystem 在未覆盖平台返回空串（由 Resolve 回退 zh-CN）。
func DetectSystem() string { return "" }

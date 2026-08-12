// Package config 管理 cli-analyzer 的本地配置，持久化到应用数据目录下的 config.json
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"cli-analyzer/internal/platform"
)

// ExpireAction 控制回收站过期项目的处理方式
const (
	// ExpireActionSystemTrash 表示过期项目移动到系统回收站
	ExpireActionSystemTrash = "system-trash"
	// ExpireActionPermanent 表示过期项目彻底删除
	ExpireActionPermanent = "permanent"
)

// TrashConfig 是内置回收站的配置项
type TrashConfig struct {
	// RetentionDays 是回收站保留天数，默认 7
	RetentionDays int `json:"retentionDays"`
	// ExpireAction 是过期处理方式：system-trash | permanent
	ExpireAction string `json:"expireAction"`
	// UseTrash 表示清理默认走内置回收站（true）还是立即删除（false）
	UseTrash bool `json:"useTrash"`
}

// ReminderConfig 是 cleanable 阈值提醒的配置项
type ReminderConfig struct {
	// ThresholdBytes 是提醒阈值，默认 5 GB
	ThresholdBytes int64 `json:"thresholdBytes"`
}

// UpdateConfig 是版本更新相关的配置项
type UpdateConfig struct {
	// CheckUpdates 表示启动时是否自动检查更新（默认 true）。
	// 用指针区分“未设置（nil → 默认 true）”与“显式 false”。
	CheckUpdates *bool `json:"checkUpdates,omitempty"`
	// LastCheckAt 是上次成功检查的时间（RFC3339），用于 24h 限流缓存。
	LastCheckAt string `json:"lastCheckAt,omitempty"`
	// LastResult 是上次检查结果的缓存 JSON（限流期内复用，避免重复请求）。
	LastResult string `json:"lastResult,omitempty"`
	// IgnoredVersion 是用户选择“忽略该版本”的版本号；
	// 在出现比它更新的版本前不再提示。
	IgnoredVersion string `json:"ignoredVersion,omitempty"`
}

// Language 可选值：auto（跟随系统）| zh-CN | zh-TW | en
const (
	// LangAuto 表示跟随系统语言
	LangAuto = "auto"
	// LangZhCN 简体中文
	LangZhCN = "zh-CN"
	// LangZhTW 繁體中文
	LangZhTW = "zh-TW"
	// LangEn English
	LangEn = "en"
)

// Config 是应用的本地配置
type Config struct {
	Trash    TrashConfig    `json:"trash"`
	Reminder ReminderConfig `json:"reminder"`
	Update   UpdateConfig   `json:"update"`
	// Language 界面语言，默认 auto（跟随系统）
	Language string `json:"language,omitempty"`
}

// defaultThresholdBytes 是阈值提醒的默认值（5 GB）
const defaultThresholdBytes int64 = 5 * 1024 * 1024 * 1024

// defaultCheckUpdates 是自动检查更新的默认值（开启）
func defaultCheckUpdates() bool { return true }

// dataRoot 可被测试替换，指向隔离的临时目录
var dataRoot = platform.DataRoot

// SetDataRoot 临时覆盖配置数据根目录（测试隔离用），返回恢复函数。
func SetDataRoot(dir string) func() {
	orig := dataRoot
	dataRoot = func() string { return dir }
	return func() { dataRoot = orig }
}

// Path 返回配置文件路径：<DataRoot>/config.json
func Path() string { return filepath.Join(dataRoot(), "config.json") }

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Trash: TrashConfig{
			RetentionDays: 7,
			ExpireAction:  ExpireActionSystemTrash,
			UseTrash:      true,
		},
		Reminder: ReminderConfig{
			ThresholdBytes: defaultThresholdBytes,
		},
		Language: LangAuto,
	}
}

// Load 读取配置；文件不存在或内容非法时回退默认值
func Load() *Config {
	cfg := Default()
	b, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return Default()
	}
	cfg.normalize()
	return cfg
}

// Save 将配置原子写回 config.json（先写临时文件再 rename）
func Save(c *Config) error {
	if c == nil {
		c = Default()
	}
	c.normalize()
	dir := filepath.Dir(Path())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := Path() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, Path())
}

// normalize 将非法值回退到默认
func (c *Config) normalize() {
	if c.Trash.RetentionDays <= 0 {
		c.Trash.RetentionDays = 7
	}
	if c.Trash.ExpireAction != ExpireActionSystemTrash && c.Trash.ExpireAction != ExpireActionPermanent {
		c.Trash.ExpireAction = ExpireActionSystemTrash
	}
	if c.Reminder.ThresholdBytes <= 0 {
		c.Reminder.ThresholdBytes = defaultThresholdBytes
	}
	// UpdateConfig 缺失或未显式设置时默认开启自动检查；
	// 旧 config.json（无 update 段）加载后 CheckUpdates 为 nil → 默认 true
	if c.Update.CheckUpdates == nil {
		d := defaultCheckUpdates()
		c.Update.CheckUpdates = &d
	}
	// Language 非法值回退 auto（旧配置无此字段 → auto 跟随系统）
	switch c.Language {
	case LangAuto, LangZhCN, LangZhTW, LangEn:
	default:
		c.Language = LangAuto
	}
}

// CheckUpdatesEnabled 返回自动检查更新的生效值（nil 兜底为 true）。
func (u UpdateConfig) CheckUpdatesEnabled() bool {
	if u.CheckUpdates == nil {
		return defaultCheckUpdates()
	}
	return *u.CheckUpdates
}

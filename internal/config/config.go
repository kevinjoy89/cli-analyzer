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

// Config 是应用的本地配置
type Config struct {
	Trash TrashConfig `json:"trash"`
}

// dataRoot 可被测试替换，指向隔离的临时目录
var dataRoot = platform.DataRoot

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
}

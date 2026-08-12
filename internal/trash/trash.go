// Package trash 实现内置回收站：延迟删除、恢复与过期清除
package trash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
)

// ErrCrossFilesystem 表示待清理项与回收站不在同一文件系统，无法秒移
var ErrCrossFilesystem error = i18n.NewError("err.trashCrossFS")

// Item 是回收站中的一个项目
type Item struct {
	ID        string `json:"id"`
	Original  string `json:"original"`
	Tool      string `json:"tool"`
	Kind      string `json:"kind"`
	Bytes     int64  `json:"bytes"`
	TrashedAt string `json:"trashedAt"`
	ExpiresAt string `json:"expiresAt"`
	// Path 是回收站内实际内容路径（_data），不序列化
	Path string `json:"-"`
}

// Root 返回内置回收站根目录；作为函数变量可被测试替换
var Root = func() string { return filepath.Join(platform.DataRoot(), "trash") }

// devOfFn 可被测试替换，用于模拟跨文件系统场景
var devOfFn = devOf

func dataDirOf(itemDir string) string  { return filepath.Join(itemDir, "_data") }
func infoPathOf(itemDir string) string { return filepath.Join(itemDir, "info.json") }

// Trash 将 path 移入内置回收站；meta 需携带原路径、工具、类型与字节数
func Trash(path string, meta Item) error {
	if meta.Original == "" {
		meta.Original = path
	}
	// 回收站根可能尚不存在（首次清理），必须先创建，否则对 Root() 的
	// os.Stat 会失败，导致同文件系统判断误判为跨文件系统而拒绝移入
	if err := os.MkdirAll(Root(), 0o755); err != nil {
		return err
	}
	if !sameFS(path, Root()) {
		return ErrCrossFilesystem
	}
	cfg := config.Load()
	now := time.Now()
	meta.TrashedAt = now.Format(time.RFC3339)
	meta.ExpiresAt = now.AddDate(0, 0, cfg.Trash.RetentionDays).Format(time.RFC3339)
	meta.ID = newDirName(path)
	itemDir := filepath.Join(Root(), meta.ID)
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return err
	}
	if err := os.Rename(path, dataDirOf(itemDir)); err != nil {
		_ = os.Remove(itemDir)
		return err
	}
	meta.Path = dataDirOf(itemDir)
	return writeInfo(itemDir, meta)
}

// Restore 将指定项目还原到原路径；原路径被占用时改名还原并返回实际路径
func Restore(id string) (string, error) {
	if id == "" || filepath.Base(id) != id {
		return "", fmt.Errorf("%s", i18n.T("err.trashInvalidID", map[string]any{"id": id}))
	}
	itemDir := filepath.Join(Root(), id)
	meta, err := readInfo(itemDir)
	if err != nil {
		return "", err
	}
	target := meta.Original
	if _, err := os.Lstat(target); err == nil {
		target = uniquify(target)
	}
	if err := os.Rename(dataDirOf(itemDir), target); err != nil {
		return "", err
	}
	_ = os.Remove(infoPathOf(itemDir))
	_ = os.Remove(itemDir)
	return target, nil
}

// Sweep 清除所有已过期的项目，返回处理条数与逐项错误（扫描时顺带调用）
func Sweep() (int, []string) {
	action := config.Load().Trash.ExpireAction
	var removed int
	var errs []string
	dirs, err := os.ReadDir(Root())
	if err != nil {
		return 0, nil // 回收站不存在即无事可做
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		itemDir := filepath.Join(Root(), d.Name())
		meta, err := readInfo(itemDir)
		if err != nil {
			errs = append(errs, itemDir+": "+err.Error())
			continue
		}
		exp, err := time.Parse(time.RFC3339, meta.ExpiresAt)
		if err != nil {
			errs = append(errs, itemDir+": "+i18n.T("err.trashBadExpiry"))
			continue
		}
		if time.Now().Before(exp) {
			continue
		}
		if err := removeExpired(itemDir, action); err != nil {
			errs = append(errs, itemDir+": "+err.Error())
			continue
		}
		removed++
	}
	return removed, errs
}

// TrashInfo 是回收站的占用统计
type TrashInfo struct {
	Items      int   `json:"items"`
	TotalBytes int64 `json:"totalBytes"`
	// EarliestExp 是最早的到期时间，用于"何时开始释放空间"的展示
	EarliestExp string `json:"earliestExpiresAt"`
}

// Info 统计回收站总占用、项数与最早到期时间
func Info() *TrashInfo {
	info := &TrashInfo{}
	dirs, err := os.ReadDir(Root())
	if err != nil {
		return info
	}
	var earliest time.Time
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		meta, err := readInfo(filepath.Join(Root(), d.Name()))
		if err != nil {
			continue
		}
		info.Items++
		info.TotalBytes += meta.Bytes
		if exp, err := time.Parse(time.RFC3339, meta.ExpiresAt); err == nil && (earliest.IsZero() || exp.Before(earliest)) {
			earliest = exp
		}
	}
	if !earliest.IsZero() {
		info.EarliestExp = earliest.Format(time.RFC3339)
	}
	return info
}

// List 返回回收站全部项目，按移入时间倒序；空回收站返回空数组而非 nil
func List() ([]Item, error) {
	out := []Item{}
	dirs, err := os.ReadDir(Root())
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, err
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		if meta, err := readInfo(filepath.Join(Root(), d.Name())); err == nil {
			out = append(out, *meta)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TrashedAt > out[j].TrashedAt })
	return out, nil
}

// Purge 立即清出回收站中指定项目；处理方式遵循过期配置——默认移到系统回收站
// （最后一道保险），配置为 permanent 时才彻底删除
func Purge(ids []string) (deleted []string, errs []string) {
	action := config.Load().Trash.ExpireAction
	for _, id := range ids {
		if id == "" || filepath.Base(id) != id {
			errs = append(errs, id+": "+i18n.T("err.trashInvalidItem"))
			continue
		}
		if err := removeExpired(filepath.Join(Root(), id), action); err != nil {
			errs = append(errs, id+": "+err.Error())
			continue
		}
		deleted = append(deleted, id)
	}
	return deleted, errs
}

// ---- 内部实现 ----

// sameFS 判断两个路径是否位于同一文件系统（实现按平台拆分在 devof_*.go）
func sameFS(a, b string) bool {
	da, ea := devOfFn(a)
	db, eb := devOfFn(b)
	return ea == nil && eb == nil && da == db
}

// newDirName 生成唯一项目目录名：时间戳_清洗后的原 basename，冲突时追加序号
func newDirName(original string) string {
	base := sanitize(filepath.Base(original))
	if base == "" || base == "." {
		base = "item"
	}
	name := time.Now().Format("20060102-150405") + "_" + base
	for i := 2; ; i++ {
		if _, err := os.Lstat(filepath.Join(Root(), name)); os.IsNotExist(err) {
			break
		}
		name = fmt.Sprintf("%s_%d_%s", time.Now().Format("20060102-150405"), i, base)
	}
	return name
}

// sanitize 将路径分隔符与控制字符替换为下划线，保证目录名合法
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r < 32 || r == 0 {
			return '_'
		}
		return r
	}, name)
}

// uniquify 在目标已存在时追加序号，返回一个不冲突的路径
func uniquify(target string) string {
	if _, err := os.Lstat(target); os.IsNotExist(err) {
		return target
	}
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	for i := 1; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (restored %d)", base, i))
		if _, err := os.Lstat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

// writeInfo 写元数据并落盘，保证崩溃后可恢复
func writeInfo(itemDir string, meta Item) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	p := infoPathOf(itemDir)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return err
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// readInfo 读取项目元数据并补全 ID 与 Path
func readInfo(itemDir string) (*Item, error) {
	b, err := os.ReadFile(infoPathOf(itemDir))
	if err != nil {
		return nil, err
	}
	var it Item
	if err := json.Unmarshal(b, &it); err != nil {
		return nil, err
	}
	it.Path = dataDirOf(itemDir)
	it.ID = filepath.Base(itemDir)
	return &it, nil
}

// removeExpired 按配置清除一个过期项；系统回收站不可用时降级为彻底删除
func removeExpired(itemDir string, action string) error {
	content := dataDirOf(itemDir)
	if action == config.ExpireActionSystemTrash {
		if err := systemTrashFn(content); err != nil {
			_ = os.RemoveAll(content)
		}
	} else {
		if err := os.RemoveAll(content); err != nil {
			return err
		}
	}
	_ = os.Remove(infoPathOf(itemDir))
	_ = os.Remove(itemDir)
	return nil
}

// errNoSystemTrash 表示当前平台没有可用的系统回收站命令
var errNoSystemTrash error = i18n.NewError("err.trashNoSystem")

// systemTrashFn 可被测试替换，避免测试污染真实系统回收站；
// 实际实现按平台拆分在 systemtrash_darwin.go / systemtrash_linux.go / systemtrash_windows.go
var systemTrashFn = systemTrash

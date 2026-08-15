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
	if err := writeInfoFn(itemDir, meta); err != nil {
		// 元数据落盘失败（磁盘满/权限）：数据已被 Rename 移入回收站但无
		// info.json 索引，列表不可见、恢复不可达、Sweep 也会跳过 → 永久
		// 孤儿数据。立即把数据移回原路径回滚现场。
		if rbErr := os.Rename(dataDirOf(itemDir), path); rbErr == nil {
			_ = os.Remove(itemDir)
		}
		return err
	}
	return nil
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
	// 原父目录可能已被用户删除：恢复必须重建父目录，否则 Rename 直接
	// 失败（ENOENT），数据困在回收站无法恢复
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
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
			// 自愈：子项精确分类修复前写入的旧条目（_logs/*.log 被记成父项的
			// cache），列表时按与扫描器一致的规则纠正，避免错误标签保留整个
			// 保留期。仅升级可明确识别的组合，不改动其他类型。
			meta.Kind = refineKind(meta.Kind, meta.Original)
			out = append(out, *meta)
		}
	}
	// 确定性排序：同移入时间（同秒批量）按原路径升序，列表顺序可复现
	sort.Slice(out, func(i, j int) bool {
		if out[i].TrashedAt != out[j].TrashedAt {
			return out[i].TrashedAt > out[j].TrashedAt
		}
		return out[i].Original < out[j].Original
	})
	return out, nil
}

// refineKind 纠正修复前写入的错误类型：路径名明确是日志（_logs / *.log）
// 却记录为 cache 的旧条目 → logs。与 scanner.subKind 的分类规则一致。
func refineKind(kind, original string) string {
	if kind != "cache" {
		return kind
	}
	low := strings.ToLower(filepath.Base(original))
	if low == "_logs" || strings.HasSuffix(low, ".log") {
		return "logs"
	}
	return kind
}

// Purge 立即永久删除回收站中指定项目（"清空/彻底删除"的显式语义）。
// 调用方（CLI `trash empty` / GUI "彻底删除"）承诺的是不可恢复的永久删除，
// 不经过系统回收站——README、CLI 注释与前端文案（Delete permanently）均
// 如此声明，此前实现却复用过期配置（默认 system-trash），"清空"只是把项目
// 转到系统回收站，磁盘空间并未释放，行为与契约不符。
// 过期项目的自动清理（Sweep）才遵循 ExpireAction 配置（默认系统回收站，
// 作为无人值守路径的最后一道保险）。
func Purge(ids []string) (deleted []string, errs []string) {
	for _, id := range ids {
		if id == "" || filepath.Base(id) != id {
			errs = append(errs, id+": "+i18n.T("err.trashInvalidItem"))
			continue
		}
		if err := removeExpired(filepath.Join(Root(), id), config.ExpireActionPermanent); err != nil {
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

// writeInfo 写元数据并落盘，保证崩溃后可恢复。
// 原子写（唯一 tmp + rename）：进程崩溃时不会留下半截 info.json——
// 半截 JSON 会让 readInfo 失败，项目数据既不可见也不可恢复（永久孤儿）。
// 注意：必须用可写句柄打开再 Sync——Windows 的 FlushFileBuffers 对只读
// 句柄返回 ERROR_ACCESS_DENIED（unix fsync 无此限制）。
func writeInfo(itemDir string, meta Item) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(itemDir, "info-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return platform.RenameReplace(tmpName, infoPathOf(itemDir))
}

// writeInfoFn 可被测试替换，用于模拟元数据落盘失败（磁盘满/权限）场景
var writeInfoFn = writeInfo

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

// removeExpired 按配置清除一个过期项；系统回收站不可用时降级为彻底删除。
// 降级删除必须检查错误：RemoveAll 失败（权限）时保留 info.json 与项目目录，
// 让该项目留在回收站中待下次重试——此前忽略错误仍删 info，数据会变成
// 既不可见也不可恢复的孤儿。
func removeExpired(itemDir string, action string) error {
	content := dataDirOf(itemDir)
	if action == config.ExpireActionSystemTrash {
		if err := systemTrashFn(content); err != nil {
			if err2 := os.RemoveAll(content); err2 != nil {
				return err2
			}
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

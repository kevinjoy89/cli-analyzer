package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cli-analyzer/internal/platform"
)

// 用户规则包：内置规则表编译进二进制，新工具只能等发版。这里允许用户在
// <数据目录>/rules.d/*.json 放自己的条目，扫描时一并加载。
//
// 语义（与内置表的优先级）：
//   - 同名条目**整条替换**内置条目（用户想要完全控制，不做字段级合并——
//     合并会让「我明明写对了却还是老行为」变成猜谜）；
//   - 新名字**追加**，与内置条目同一套归因/清理管线，也能带 Category/Apps；
//   - 文件里任何一条不合法就跳过该条并记录错误，不影响其余条目与内置表：
//     规则文件写坏了不该让整个扫描失败。
//
// 用户规则不改变领域口径：Root/Sub/Path/Tier/Kind 的含义与内置条目完全一致。

// userRulesDir 可被测试替换；默认 <DataRoot>/rules.d。
var userRulesDir = func() string { return filepath.Join(platform.DataRoot(), "rules.d") }

// UserRulesDir 返回用户规则包目录（供 CLI/GUI 展示与用户手动放置文件）。
func UserRulesDir() string { return userRulesDir() }

// userFile 是一个规则包文件的顶层结构。
type userFile struct {
	// SchemaVersion 目前只接受 1 或缺失；其它值视为未来格式，跳过并报错。
	SchemaVersion int        `json:"schemaVersion"`
	Rules         []userRule `json:"rules"`
}

type userRule struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Aliases     []string        `json:"aliases"`
	Installer   string          `json:"installer"`
	Homepage    string          `json:"homepage"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Apps        []string        `json:"apps"`
	DataDirs    []userDataDir   `json:"dataDirs"`
	Cleanables  []userCleanRule `json:"cleanables"`
}

type userDataDir struct {
	Root string `json:"root"`
	Sub  string `json:"sub"`
	Path string `json:"path"`
	Tier string `json:"tier"`
	Kind string `json:"kind"`
	Form string `json:"form"`
}

type userCleanRule struct {
	Root string `json:"root"`
	Sub  string `json:"sub"`
	Path string `json:"path"`
	Tier string `json:"tier"`
	Kind string `json:"kind"`
	Keep string `json:"keep"`
	Desc string `json:"desc"`
}

// validRoots 是允许在规则文件里出现的根类型。
var validRoots = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range platform.AllKinds {
		m[string(k)] = true
	}
	return m
}()

// loadUserRules 读取目录下的全部 *.json，返回可用条目与逐条错误。
// 目录不存在是正常情况（大多数用户不会放规则包），不算错误。
func loadUserRules(dir string) ([]Rule, []string) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	// 按文件名排序：同一工具在多个文件里出现时，结果可预期（后加载者覆盖）。
	sort.Strings(names)

	var out []Rule
	var errs []string
	for _, name := range names {
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		var f userFile
		if err := json.Unmarshal(b, &f); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if f.SchemaVersion != 0 && f.SchemaVersion != 1 {
			errs = append(errs, fmt.Sprintf("%s: unsupported schemaVersion %d", name, f.SchemaVersion))
			continue
		}
		for i, ur := range f.Rules {
			r, err := ur.toRule()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s[%d]: %v", name, i, err))
				continue
			}
			out = append(out, r)
		}
	}
	return out, errs
}

// toRule validates and converts one user entry.
func (u userRule) toRule() (Rule, error) {
	name := strings.TrimSpace(u.Name)
	if name == "" {
		return Rule{}, fmt.Errorf("missing name")
	}
	r := Rule{
		Name:        name,
		DisplayName: strings.TrimSpace(u.DisplayName),
		Aliases:     cleanStrings(u.Aliases),
		Installer:   strings.TrimSpace(u.Installer),
		Homepage:    strings.TrimSpace(u.Homepage),
		Description: strings.TrimSpace(u.Description),
		Category:    strings.TrimSpace(u.Category),
		Apps:        cleanStrings(u.Apps),
	}
	for i, d := range u.DataDirs {
		rule, err := d.toRule()
		if err != nil {
			return Rule{}, fmt.Errorf("dataDirs[%d]: %w", i, err)
		}
		r.DataDirs = append(r.DataDirs, rule)
	}
	for i, c := range u.Cleanables {
		rule, err := c.toRule()
		if err != nil {
			return Rule{}, fmt.Errorf("cleanables[%d]: %w", i, err)
		}
		r.Cleanables = append(r.Cleanables, rule)
	}
	return r, nil
}

func (d userDataDir) toRule() (DataDirRule, error) {
	root := strings.TrimSpace(d.Root)
	path := strings.TrimSpace(d.Path)
	if root == "" && path == "" {
		return DataDirRule{}, fmt.Errorf("need root or path")
	}
	if root != "" && !validRoots[root] {
		return DataDirRule{}, fmt.Errorf("unknown root %q", root)
	}
	if path != "" && !filepath.IsAbs(path) {
		return DataDirRule{}, fmt.Errorf("path must be absolute: %q", path)
	}
	form := strings.TrimSpace(d.Form)
	if form != "" && form != FormDesktop {
		return DataDirRule{}, fmt.Errorf("unknown form %q", form)
	}
	return DataDirRule{
		Root: platform.RootKind(root),
		Sub:  strings.TrimSpace(d.Sub),
		Path: path,
		Tier: strings.TrimSpace(d.Tier),
		Kind: strings.TrimSpace(d.Kind),
		Form: form,
	}, nil
}

func (c userCleanRule) toRule() (CleanRule, error) {
	root := strings.TrimSpace(c.Root)
	path := strings.TrimSpace(c.Path)
	if root == "" && path == "" {
		return CleanRule{}, fmt.Errorf("need root or path")
	}
	if root != "" && !validRoots[root] {
		return CleanRule{}, fmt.Errorf("unknown root %q", root)
	}
	if path != "" && !filepath.IsAbs(path) {
		return CleanRule{}, fmt.Errorf("path must be absolute: %q", path)
	}
	return CleanRule{
		Root: platform.RootKind(root),
		Sub:  strings.TrimSpace(c.Sub),
		Path: path,
		Tier: strings.TrimSpace(c.Tier),
		Kind: strings.TrimSpace(c.Kind),
		Keep: strings.TrimSpace(c.Keep),
		Desc: strings.TrimSpace(c.Desc),
	}, nil
}

func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

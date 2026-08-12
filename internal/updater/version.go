package updater

import (
	"fmt"
	"strconv"
	"strings"
)

// Version 是语义化版本号（major.minor.patch[.patch2]），tag 前缀 "v" 会被忽略。
// 第四段（hotfix/build 号）可选：缺省视为 0，比较时参与数值比较。
type Version struct {
	Major  int
	Minor  int
	Patch  int
	Patch2 int
}

// ParseVersion 解析形如 "0.2.3"、"v0.2.3"、"0.3.2.1" 的版本号。
// 只接受纯数字的三段或四段式版本；预发布后缀（如 "0.3.0-beta.1"）视为非法，
// 因为发布流程只产生正式版 tag（见 design D3）。
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return Version{}, fmt.Errorf("invalid version %q: want major.minor.patch[.patch2]", s)
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("invalid version %q: component %q is not a non-negative integer", s, p)
		}
		nums[i] = n
	}
	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}
	if len(nums) == 4 {
		v.Patch2 = nums[3]
	}
	return v, nil
}

// Compare 返回 -1/0/1：a < b / a == b / a > b。
func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		return cmpInt(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return cmpInt(v.Minor, o.Minor)
	}
	if v.Patch != o.Patch {
		return cmpInt(v.Patch, o.Patch)
	}
	return cmpInt(v.Patch2, o.Patch2)
}

// String 返回 "major.minor.patch"（第四段非零时附加 ".patch2"）。
func (v Version) String() string {
	if v.Patch2 > 0 {
		return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Patch2)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

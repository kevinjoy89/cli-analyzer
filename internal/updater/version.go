package updater

import (
	"fmt"
	"strconv"
	"strings"
)

// Version 是语义化版本号（major.minor.patch），tag 前缀 "v" 会被忽略。
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion 解析形如 "0.2.3" 或 "v0.2.3" 的版本号。
// 只接受纯数字的三段式版本；预发布后缀（如 "0.3.0-beta.1"）视为非法，
// 因为发布流程只产生正式版 tag（见 design D3）。
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: want major.minor.patch", s)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("invalid version %q: component %q is not a non-negative integer", s, p)
		}
		nums[i] = n
	}
	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}, nil
}

// Compare 返回 -1/0/1：a < b / a == b / a > b。
func (v Version) Compare(o Version) int {
	switch {
	case v.Major != o.Major:
		return cmpInt(v.Major, o.Major)
	case v.Minor != o.Minor:
		return cmpInt(v.Minor, o.Minor)
	default:
		return cmpInt(v.Patch, o.Patch)
	}
}

// String 返回 "major.minor.patch"（无 "v" 前缀）。
func (v Version) String() string {
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

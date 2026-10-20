package uninstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"cli-analyzer/internal/platform"
)

// 残留闭环：卸载过一次不代表清理结束。工具卸掉后，配置/缓存/会话数据往往
// 还留在磁盘上，而用户不会主动再跑一次 uninstall 去确认。这里把「卸载过」
// 这件事落盘，之后由 `cli-analyzer residue` 提醒「N 天前卸载的某工具，残留
// 还在，约 X」。被动列表变成有始有终的流程。
//
// 刻意做得很轻：只记工具名、时间、当时的残留路径。不跟踪字节数变化（那要
// 重新扫描），也不做定时任务（CLI 不该常驻）。

// Event 是一次卸载的记录。
type Event struct {
	Tool string `json:"tool"`
	// At 是卸载动作发生的时间（RFC3339）。
	At string `json:"at"`
	// Residue 是当时检测到的残留路径；之后逐条判断是否仍存在。
	Residue []string `json:"residue"`
}

// trackingPath 可被测试替换。
var trackingPath = func() string {
	return filepath.Join(platform.DataRoot(), "uninstall-events.json")
}

// maxEvents 限制历史条数：这是一份提醒清单，不是审计日志。
const maxEvents = 200

// RecordEvent 记录一次卸载（同工具重复卸载时覆盖旧记录，保留最新时间）。
func RecordEvent(tool string, residuePaths []string) error {
	events, _ := loadEvents()
	now := time.Now().Format(time.RFC3339)
	kept := make([]Event, 0, len(events)+1)
	for _, e := range events {
		if e.Tool != tool {
			kept = append(kept, e)
		}
	}
	kept = append(kept, Event{Tool: tool, At: now, Residue: append([]string(nil), residuePaths...)})
	if len(kept) > maxEvents {
		kept = kept[len(kept)-maxEvents:]
	}
	return saveEvents(kept)
}

// Pending 返回「卸载已满 minDays 天、且残留仍在」的记录。
// 残留字节数现算（用 disk 口径），因为事件里存的是路径不是大小。
func Pending(minDays int) ([]PendingEvent, error) {
	events, err := loadEvents()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().AddDate(0, 0, -minDays)
	var out []PendingEvent
	for _, e := range events {
		at, err := time.Parse(time.RFC3339, e.At)
		if err != nil || at.After(cutoff) {
			continue
		}
		var still []Residue
		var total int64
		for _, p := range e.Residue {
			if r, ok := stillExists(p); ok {
				still = append(still, r)
				total += r.Bytes
			}
		}
		if len(still) == 0 {
			continue // 残留已清干净：不再打扰
		}
		out = append(out, PendingEvent{Tool: e.Tool, At: e.At, DaysAgo: daysSince(at), Residue: still, Bytes: total})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out, nil
}

// PendingEvent 是一条待处理的残留提醒。
type PendingEvent struct {
	Tool    string    `json:"tool"`
	At      string    `json:"at"`
	DaysAgo int       `json:"daysAgo"`
	Residue []Residue `json:"residue"`
	Bytes   int64     `json:"bytes"`
}

// Forget 删除某工具的跟踪记录（用户处置完残留后可显式收尾）。
func Forget(tool string) error {
	events, err := loadEvents()
	if err != nil {
		return err
	}
	kept := make([]Event, 0, len(events))
	for _, e := range events {
		if e.Tool != tool {
			kept = append(kept, e)
		}
	}
	return saveEvents(kept)
}

func daysSince(t time.Time) int {
	d := int(time.Since(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

func loadEvents() ([]Event, error) {
	b, err := os.ReadFile(trackingPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []Event
	if err := json.Unmarshal(b, &events); err != nil {
		// 跟踪文件损坏不该让卸载失败：当作没有记录，下次写入会覆盖。
		return nil, nil
	}
	return events, nil
}

func saveEvents(events []Event) error {
	path := trackingPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if events == nil {
		events = []Event{}
	}
	b, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

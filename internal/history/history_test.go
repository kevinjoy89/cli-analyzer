package history

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cli-analyzer/internal/scanner"
)

// withTempDB 将数据库路径指向临时目录，保证测试隔离真实文件系统
func withTempDB(t *testing.T) {
	t.Helper()
	orig := dbPath
	dir := t.TempDir()
	dbPath = func() string { return filepath.Join(dir, "history.db") }
	t.Cleanup(func() { dbPath = orig })
}

// mkRes 构造一次扫描快照，同时累加 totals
func mkRes(scannedAt string, tools ...scanner.Tool) *scanner.ScanResult {
	var tot scanner.Totals
	for _, t := range tools {
		tot.Footprint += t.Footprint
		tot.Cleanable += t.Cleanable
		tot.User += t.User
	}
	return &scanner.ScanResult{ScannedAt: scannedAt, Totals: tot, Tools: tools}
}

// tool 便捷构造一个工具快照行
func tool(name string, cleanable int64) scanner.Tool {
	return scanner.Tool{Name: name, Cleanable: cleanable, Footprint: cleanable + 10, User: 10}
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

func TestRecordAndTrendsRoundTrip(t *testing.T) {
	withTempDB(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		res := mkRes(rfc3339(now.AddDate(0, 0, -2+i)), tool("npm", int64(100*(i+1))))
		if err := Record(res); err != nil {
			t.Fatalf("Record #%d: %v", i, err)
		}
	}
	tr, err := Trends(30)
	if err != nil {
		t.Fatalf("Trends: %v", err)
	}
	if len(tr.Points) != 3 {
		t.Fatalf("Points = %d, want 3", len(tr.Points))
	}
	// 最后一条应等于最后一次扫描的 cleanable
	if last := tr.Points[len(tr.Points)-1]; last.Cleanable != 300 {
		t.Errorf("last point cleanable = %d, want 300", last.Cleanable)
	}
	if tr.Points[0].Date == "" {
		t.Error("point date 为空")
	}
}

func TestTopGrowers(t *testing.T) {
	withTempDB(t)
	now := time.Now()
	// 第一次：npm 100，uv 200
	res1 := mkRes(rfc3339(now.Add(-time.Hour)), tool("npm", 100), tool("uv", 200))
	if err := Record(res1); err != nil {
		t.Fatal(err)
	}
	// 第二次：npm 增长到 500（+400），uv 降到 150（-50）
	res2 := mkRes(rfc3339(now), tool("npm", 500), tool("uv", 150))
	if err := Record(res2); err != nil {
		t.Fatal(err)
	}
	tr, err := Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.TopGrowers) != 1 {
		t.Fatalf("TopGrowers = %d, want 1（uv 增量不为正，不应上榜）", len(tr.TopGrowers))
	}
	if g := tr.TopGrowers[0]; g.Tool != "npm" || g.DeltaBytes != 400 {
		t.Errorf("grower = %+v, want npm +400", g)
	}
}

func TestTopGrowersNeedsTwoScans(t *testing.T) {
	withTempDB(t)
	if err := Record(mkRes(rfc3339(time.Now()), tool("npm", 100))); err != nil {
		t.Fatal(err)
	}
	tr, err := Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.TopGrowers) != 0 {
		t.Errorf("历史不足两次时 TopGrowers 应为空, got %v", tr.TopGrowers)
	}
}

func TestTrendsFiltersByDays(t *testing.T) {
	withTempDB(t)
	now := time.Now()
	// 一条 40 天前的（超出 30 天窗口）
	if err := Record(mkRes(rfc3339(now.AddDate(0, 0, -40)), tool("npm", 100))); err != nil {
		t.Fatal(err)
	}
	// 一条今天的
	if err := Record(mkRes(rfc3339(now), tool("npm", 200))); err != nil {
		t.Fatal(err)
	}
	tr, err := Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Points) != 1 {
		t.Errorf("30 天窗口内 Points = %d, want 1", len(tr.Points))
	}
}

func TestPruneRemovesOldRecords(t *testing.T) {
	withTempDB(t)
	now := time.Now()
	// 写入一条 100 天前的（超过默认 90 天保留期）
	if err := Record(mkRes(rfc3339(now.AddDate(0, 0, -100)), tool("old", 1))); err != nil {
		t.Fatal(err)
	}
	// 写入一条今天的（触发 prune）
	if err := Record(mkRes(rfc3339(now), tool("new", 1))); err != nil {
		t.Fatal(err)
	}
	tr, err := Trends(365)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Points) != 1 {
		t.Errorf("prune 后 Points = %d, want 1（旧记录应被清理）", len(tr.Points))
	}
	if len(tr.Points) == 1 && tr.Points[0].Date != now.Format("2006-01-02") {
		t.Errorf("保留的应是新记录, got %s", tr.Points[0].Date)
	}
}

func TestRecordNilIgnored(t *testing.T) {
	withTempDB(t)
	if err := Record(nil); err == nil {
		t.Fatal("Record(nil) 应返回错误")
	}
}

// TestTrendsEmptySerializesWithoutNull verifies no-history Trends never emits
// null for points/topGrowers (nil slices would crash the frontend).
func TestTrendsEmptySerializesWithoutNull(t *testing.T) {
	withTempDB(t)
	tr, err := Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(tr)
	if strings.Contains(string(b), "null") {
		t.Errorf("无历史时序列化不应出现 null: %s", b)
	}
}

// Package history 持久化每次扫描的快照，供占用趋势分析与增量排行使用
package history

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"

	_ "modernc.org/sqlite" // 纯 Go 驱动，无 cgo
)

// defaultRetentionDays 是历史快照的默认保留天数
const defaultRetentionDays = 90

// dbPath 可被测试替换，指向隔离的临时数据库文件
var dbPath = func() string { return filepath.Join(platform.DataRoot(), "history.db") }

// Point 是某个时间点的整体占用快照
type Point struct {
	Date      string `json:"date"`
	Footprint int64  `json:"footprint"`
	Cleanable int64  `json:"cleanable"`
	User      int64  `json:"user"`
}

// Grower 是 cleanable 增量最大的工具
type Grower struct {
	Tool       string `json:"tool"`
	DeltaBytes int64  `json:"deltaBytes"`
}

// TrendsResult 是趋势视图的数据契约（GUI 与 CLI 共用）
type TrendsResult struct {
	Points     []Point  `json:"points"`
	TopGrowers []Grower `json:"topGrowers"`
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS scans (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scanned_at TEXT NOT NULL,
	footprint INTEGER NOT NULL,
	cleanable INTEGER NOT NULL,
	user INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS scan_tools (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id INTEGER NOT NULL REFERENCES scans(id),
	tool TEXT NOT NULL,
	footprint INTEGER NOT NULL,
	cleanable INTEGER NOT NULL,
	user INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_scan_tools_scan ON scan_tools(scan_id);
`

// open 打开数据库并确保表结构存在；首次运行会先创建父目录
func open() (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath()), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dbPath())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Record 将一次扫描快照追加到历史；写入失败返回 error（调用方自行决定是否静默）
func Record(res *scanner.ScanResult) error {
	if res == nil {
		return errors.New("nil scan result")
	}
	db, err := open()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ins, err := tx.Prepare(`INSERT INTO scans(scanned_at, footprint, cleanable, user) VALUES(?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	r, err := ins.Exec(res.ScannedAt, res.Totals.Footprint, res.Totals.Cleanable, res.Totals.User)
	if err != nil {
		return err
	}
	scanID, err := r.LastInsertId()
	if err != nil {
		return err
	}
	insTool, err := tx.Prepare(`INSERT INTO scan_tools(scan_id, tool, footprint, cleanable, user) VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for _, t := range res.Tools {
		if _, err := insTool.Exec(scanID, t.Name, t.Footprint, t.Cleanable, t.User); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return prune(db)
}

// scanRow 是 scans 表的一行，供 Trends 与 topGrowers 共享
type scanRow struct {
	id        int64
	scannedAt string
	footprint int64
	cleanable int64
	user      int64
}

// Trends 返回最近 days 天内的整体占用点与 cleanable 增量 Top 5
func Trends(days int) (*TrendsResult, error) {
	db, err := open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, scanned_at, footprint, cleanable, user FROM scans ORDER BY scanned_at`)
	if err != nil {
		return nil, err
	}
	var all []scanRow
	for rows.Next() {
		var s scanRow
		if err := rows.Scan(&s.id, &s.scannedAt, &s.footprint, &s.cleanable, &s.user); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, s)
	}
	// 循环正常结束不保证无错误：中途 IO/损坏错误只在 Err 里暴露，
	// 不检查会把不完整数据当全量返回（趋势图/增量排行静默失真）
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	cutoff := time.Now().AddDate(0, 0, -days)
	// 无历史时返回空数组而非 nil，保证 JSON 序列化为 [] 而不是 null
	out := &TrendsResult{Points: []Point{}, TopGrowers: []Grower{}}
	// 窗口内记录：Points 与 TopGrowers 共用同一过滤——TopGrowers 若用
	// 全量历史（含窗口外旧扫描）会把"40 天前的扫描"当作上次扫描计算
	// 增量，导致 `trends 7` 显示 40 天前的增长（契约不一致）。
	var inWindow []scanRow
	for _, s := range all {
		t, err := time.Parse(time.RFC3339, s.scannedAt)
		if err != nil || t.Before(cutoff) {
			continue
		}
		inWindow = append(inWindow, s)
		p := Point{
			Date:      t.Format("2006-01-02"),
			Footprint: s.footprint, Cleanable: s.cleanable, User: s.user,
		}
		// 同一天多次扫描只保留最后一个快照，避免趋势图出现重复日期与平直线
		if n := len(out.Points); n > 0 && out.Points[n-1].Date == p.Date {
			out.Points[n-1] = p
		} else {
			out.Points = append(out.Points, p)
		}
	}
	out.TopGrowers, err = topGrowers(db, inWindow, 5)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// topGrowers 用最近两次扫描的 cleanable 差计算增长排行；历史不足两次返回空数组
func topGrowers(db *sql.DB, all []scanRow, n int) ([]Grower, error) {
	if len(all) < 2 {
		return []Grower{}, nil
	}
	prev := all[len(all)-2].id
	last := all[len(all)-1].id
	delta := map[string]int64{}
	load := func(scanID int64) (map[string]int64, error) {
		m := map[string]int64{}
		rows, err := db.Query(`SELECT tool, cleanable FROM scan_tools WHERE scan_id = ?`, scanID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var tool string
			var clean int64
			if err := rows.Scan(&tool, &clean); err != nil {
				return nil, err
			}
			m[tool] = clean
		}
		// 中途错误必须向上传播：静默返回不完整数据会让增量排行失真，
		// 且调用方无法区分"正常空结果"与"查询失败"
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return m, nil
	}
	prevTools, err := load(prev)
	if err != nil {
		return nil, err
	}
	lastTools, err := load(last)
	if err != nil {
		return nil, err
	}
	for tool, clean := range lastTools {
		delta[tool] = clean - prevTools[tool]
	}
	// 只统计有增量的工具，按增量降序
	growers := []Grower{}
	for tool, d := range delta {
		if d > 0 {
			growers = append(growers, Grower{Tool: tool, DeltaBytes: d})
		}
	}
	// 确定性排序：同增量按工具名升序
	sort.Slice(growers, func(i, j int) bool {
		if growers[i].DeltaBytes != growers[j].DeltaBytes {
			return growers[i].DeltaBytes > growers[j].DeltaBytes
		}
		return growers[i].Tool < growers[j].Tool
	})
	if len(growers) > n {
		growers = growers[:n]
	}
	return growers, nil
}

// prune 删除超出保留期（默认 90 天）的旧历史，避免无限增长
func prune(db *sql.DB) error {
	cutoff := time.Now().AddDate(0, 0, -defaultRetentionDays).Format(time.RFC3339)
	if _, err := db.Exec(`DELETE FROM scan_tools WHERE scan_id IN (SELECT id FROM scans WHERE scanned_at < ?)`, cutoff); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM scans WHERE scanned_at < ?`, cutoff)
	return err
}

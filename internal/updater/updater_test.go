package updater

import (
	"cli-analyzer/internal/i18n"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// releaseServer 启动一个返回固定 releases 列表的 mock GitHub API。
func releaseServer(t *testing.T, releases []Release) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/kevinjoy89/cli-analyzer/releases" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	orig := APIBaseURL
	APIBaseURL = srv.URL
	t.Cleanup(func() { APIBaseURL = orig })
	return srv
}

func TestLatestReleaseSkipsDraftAndPrerelease(t *testing.T) {
	srv := releaseServer(t, []Release{
		{TagName: "v0.3.1", Prerelease: true, Assets: []ReleaseAsset{{Name: "x"}}}, // 预发布，应跳过
		{TagName: "v0.3.0", Draft: true, Assets: []ReleaseAsset{{Name: "y"}}},      // 草稿，应跳过
		{TagName: "v0.2.4", Assets: []ReleaseAsset{{Name: "z"}}},                   // 最新正式版
		{TagName: "v0.2.3"},
	})
	_ = srv
	rel, err := LatestRelease(context.Background(), nil)
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v0.2.4" {
		t.Errorf("TagName = %q, want v0.2.4", rel.TagName)
	}
}

func TestLatestReleaseNoStable(t *testing.T) {
	releaseServer(t, []Release{
		{TagName: "v0.3.1", Prerelease: true},
		{TagName: "v0.3.0", Draft: true},
	})
	if _, err := LatestRelease(context.Background(), nil); err == nil {
		t.Fatal("want error when no stable release exists")
	}
}

func TestLatestReleaseServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	orig := APIBaseURL
	APIBaseURL = srv.URL
	t.Cleanup(func() { APIBaseURL = orig })
	if _, err := LatestRelease(context.Background(), nil); err == nil {
		t.Fatal("want error on 500")
	}
}

func TestAssetByChecksumName(t *testing.T) {
	r := &Release{Assets: []ReleaseAsset{{Name: "a.deb"}, {Name: "checksums.txt"}, {Name: "b.zip"}}}
	got := r.AssetByChecksumName()
	if got == nil || got.Name != "checksums.txt" {
		t.Fatalf("AssetByChecksumName = %v, want checksums.txt", got)
	}
	if r.AssetByChecksumName() == nil {
		t.Fatal("want non-nil for missing checksums asset")
	}
	empty := &Release{}
	if empty.AssetByChecksumName() != nil {
		t.Fatal("want nil when no checksums asset")
	}
}

// ---- asset selection ----

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch, source, want string
	}{
		{"darwin", "arm64", "dmg", "CLI-Analyzer-0.3.0-macos-arm64.dmg"},
		{"darwin", "amd64", "dmg", "CLI-Analyzer-0.3.0-macos-amd64.dmg"},
		{"windows", "amd64", "installer", "CLI-Analyzer-0.3.0-windows-amd64-installer.exe"},
		{"windows", "amd64", "portable", "CLI-Analyzer-0.3.0-windows-amd64-portable.zip"},
		{"linux", "amd64", "deb", "CLI-Analyzer-0.3.0-linux-amd64.deb"},
		{"linux", "amd64", "tarball", "CLI-Analyzer-0.3.0-linux-amd64.tar.gz"},
		{"linux", "arm64", "deb", "CLI-Analyzer-0.3.0-linux-arm64.deb"},
		{"linux", "amd64", "unknown", ""}, // 无命名规则的组合 → 空
		{"windows", "amd64", "dmg", ""},
	}
	for _, c := range cases {
		if got := AssetName("v0.3.0", c.goos, c.goarch, c.source); got != c.want {
			t.Errorf("AssetName(%s,%s,%s) = %q, want %q", c.goos, c.goarch, c.source, got, c.want)
		}
	}
}

func TestSelectAsset(t *testing.T) {
	r := &Release{TagName: "v0.3.0", Assets: []ReleaseAsset{
		{Name: "CLI-Analyzer-0.3.0-linux-amd64.deb", BrowserDownloadURL: "https://x/deb", Size: 11},
		{Name: "CLI-Analyzer-0.3.0-linux-amd64.tar.gz", BrowserDownloadURL: "https://x/tgz", Size: 22},
	}}
	got, err := SelectAsset(r, "linux", "amd64", "deb")
	if err != nil {
		t.Fatalf("SelectAsset: %v", err)
	}
	if got.Name != "CLI-Analyzer-0.3.0-linux-amd64.deb" || got.BrowserDownloadURL != "https://x/deb" || got.Size != 11 {
		t.Errorf("SelectAsset = %+v", got)
	}
	if _, err := SelectAsset(r, "linux", "amd64", "tarball"); err != nil {
		t.Errorf("tarball should match: %v", err)
	}
	if _, err := SelectAsset(r, "darwin", "arm64", "dmg"); err == nil {
		t.Error("want error when asset missing")
	}
}

// ---- download ----

func TestDownloadProgressAndFinalize(t *testing.T) {
	body := strings.Repeat("x", 100_000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100000")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	orig := downloadsDir
	downloadsDir = func() string { return t.TempDir() }
	t.Cleanup(func() { downloadsDir = orig })

	var lastWritten, lastTotal int64
	final, err := Download(context.Background(), nil, srv.URL, "file.bin", 0, func(written, total int64) {
		lastWritten, lastTotal = written, total
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if lastWritten != 100_000 || lastTotal != 100_000 {
		t.Errorf("progress = (%d,%d), want (100000,100000)", lastWritten, lastTotal)
	}
	if !strings.HasSuffix(final, "file.bin") {
		t.Errorf("final path = %q", final)
	}
	if _, err := os.Stat(final); err != nil {
		t.Errorf("final file missing: %v", err)
	}
	if _, err := os.Stat(final + ".part"); !os.IsNotExist(err) {
		t.Error(".part file should be removed after success")
	}
}

func TestDownloadCancelRemovesPart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		chunk := strings.Repeat("y", 64*1024)
		for i := 0; i < 16; i++ {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	t.Cleanup(srv.Close)

	orig := downloadsDir
	downloadsDir = func() string { return t.TempDir() }
	t.Cleanup(func() { downloadsDir = orig })

	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // 手动触发
	done := make(chan error, 1)
	go func() {
		_, err := Download(ctx, nil, srv.URL, "big.bin", 1_000_000, func(written, _ int64) {
			if written > 100_000 {
				cancel()
			}
		})
		done <- err
	}()
	err := <-done
	if err == nil {
		t.Fatal("want error after cancel")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancel error must match errors.Is(err, context.Canceled), got %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(downloadsDir(), "big.bin*"))
	if len(matches) != 0 {
		t.Errorf("leftover files after cancel: %v", matches)
	}
}

func TestDownloadServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	if _, err := Download(context.Background(), nil, srv.URL, "x.bin", 0, nil); err == nil {
		t.Fatal("want error on 404")
	}
}

// ---- checksum ----

func TestParseChecksums(t *testing.T) {
	content := []byte(`abc123  CLI-Analyzer-0.3.0-linux-amd64.deb
def456 *CLI-Analyzer-0.3.0-linux-amd64.tar.gz
badline
`)
	m := ParseChecksums(content)
	if m["CLI-Analyzer-0.3.0-linux-amd64.deb"] != "abc123" {
		t.Errorf("deb entry = %q", m["CLI-Analyzer-0.3.0-linux-amd64.deb"])
	}
	if m["CLI-Analyzer-0.3.0-linux-amd64.tar.gz"] != "def456" {
		t.Errorf("tar.gz entry = %q", m["CLI-Analyzer-0.3.0-linux-amd64.tar.gz"])
	}
	if len(m) != 2 {
		t.Errorf("parsed %d entries, want 2", len(m))
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.bin")
	data := []byte("hello updater")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	m := map[string]string{"f.bin": sha256Hex(data)}
	if err := VerifyChecksum(path, "f.bin", m); err != nil {
		t.Errorf("matching checksum rejected: %v", err)
	}
	bad := map[string]string{"f.bin": strings.Repeat("0", 64)}
	if err := VerifyChecksum(path, "f.bin", bad); err == nil {
		t.Error("mismatched checksum accepted")
	}
	if err := VerifyChecksum(path, "missing.bin", m); err == nil {
		t.Error("missing entry should error")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing entry error should wrap ErrNotExist, got %v", err)
	}
}

// ---- dpkg probe ----

func TestProbeDPKGManaged(t *testing.T) {
	// dpkg 探测是 unix 语义：fake dpkg 是 shell 脚本，Windows 无法执行
	// （无 sh 解释器），探测恒为 false。
	if runtime.GOOS == "windows" {
		t.Skip("dpkg probing is unix-only")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "fake-cli")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho fake\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "dpkg")
	script := "#!/bin/sh\necho 'cli-analyzer: " + exe + "'\nexit 0\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := dpkgCommand
	dpkgCommand = fake
	t.Cleanup(func() { dpkgCommand = orig })
	if !probeDPKG(exe) {
		t.Error("probeDPKG = false for dpkg-managed file")
	}
}

func TestProbeDPKGNotManaged(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "fake-cli")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "dpkg")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho 'no path found'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := dpkgCommand
	dpkgCommand = fake
	t.Cleanup(func() { dpkgCommand = orig })
	if probeDPKG(exe) {
		t.Error("probeDPKG = true for unmanaged file")
	}
}

func TestProbeDPKGMissingCommand(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "fake-cli")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := dpkgCommand
	dpkgCommand = filepath.Join(dir, "no-such-dpkg")
	t.Cleanup(func() { dpkgCommand = orig })
	if probeDPKG(exe) {
		t.Error("probeDPKG = true when dpkg missing")
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestLatestReleaseErrorLocalized(t *testing.T) {
	orig := i18n.ActiveLocale()
	defer i18n.SetLocale(orig)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	origAPI := APIBaseURL
	APIBaseURL = srv.URL
	t.Cleanup(func() { APIBaseURL = origAPI })

	i18n.SetLocale("zh-CN")
	_, err := LatestRelease(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "限流") {
		t.Errorf("zh-CN 403 error = %v, want friendly rate-limit message", err)
	}
	i18n.SetLocale("en")
	_, err = LatestRelease(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("en 403 error = %v, want friendly rate-limit message", err)
	}
}

// TestParseChecksumsWithBOM 验证 checksums.txt 带 UTF-8 BOM（Windows 编辑
// 保存的常见形态）时仍能解析：首行 hash 前挂 \ufeff 会让 fields[0] 含 BOM，
// 与校验计算的 hex 不匹配导致哈希校验永远失败。
func TestParseChecksumsWithBOM(t *testing.T) {
	m := ParseChecksums([]byte("\ufeffabc123  CLI-Analyzer-0.3.0-darwin-arm64.dmg\n"))
	got, ok := m["CLI-Analyzer-0.3.0-darwin-arm64.dmg"]
	if !ok || got != "abc123" {
		t.Errorf("BOM 行解析失败: %v", m)
	}
}

// TestFetchChecksumsWrapsContextError 验证 FetchChecksums 的错误保留
// context.Canceled 身份（%w 包装）：取消下载时（恰在拉取校验和阶段）
// GUI 能识别为"已取消"而非通用错误——此前 %s 格式化丢掉了错误链。
func TestFetchChecksumsWrapsContextError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // 挂起直到客户端取消
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消：client.Do 立即返回 context.Canceled
	rel := &Release{Assets: []ReleaseAsset{{Name: "checksums.txt", BrowserDownloadURL: srv.URL}}}
	_, err := FetchChecksums(ctx, nil, rel)
	if err == nil {
		t.Fatal("已取消的请求应返回错误")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("错误应可被 errors.Is 匹配 context.Canceled，got %v", err)
	}
}

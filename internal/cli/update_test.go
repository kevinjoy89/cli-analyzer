package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/updater"
)

// mockRelease 是 GitHub Releases API 返回项的简化模型。
type mockRelease struct {
	TagName string      `json:"tag_name"`
	Draft   bool        `json:"draft"`
	HTMLURL string      `json:"html_url"`
	Assets  []mockAsset `json:"assets"`
}

type mockAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// setupUpdateTest 隔离网络（mock GitHub API）、版本号与配置目录。
func setupUpdateTest(t *testing.T, version string, releases []mockRelease) {
	t.Helper()
	origVer := buildinfo.Version
	buildinfo.Version = version
	t.Cleanup(func() { buildinfo.Version = origVer })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	origAPI := updater.APIBaseURL
	updater.APIBaseURL = srv.URL
	t.Cleanup(func() { updater.APIBaseURL = origAPI })

	restore := config.SetDataRoot(t.TempDir())
	t.Cleanup(restore)
	c := config.Default()
	c.Language = config.LangZhCN
	if err := config.Save(c); err != nil {
		t.Fatal(err)
	}
}

func TestRunUpdateCheckUpToDate(t *testing.T) {
	setupUpdateTest(t, "0.3.0", []mockRelease{{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"}})
	buf := captureStdout(t)
	if code := Run([]string{"update", "check"}); code != 0 {
		t.Fatalf("up-to-date exit = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "已是最新版本 v0.3.0") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestRunUpdateCheckHasUpdate(t *testing.T) {
	setupUpdateTest(t, "0.2.3", []mockRelease{{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"}})
	buf := captureStdout(t)
	if code := Run([]string{"update", "check"}); code != 2 {
		t.Fatalf("has-update exit = %d, want 2", code)
	}
	if !strings.Contains(buf.String(), "发现新版本: v0.3.0") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestRunUpdateCheckJSON(t *testing.T) {
	setupUpdateTest(t, "0.2.3", []mockRelease{{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"}})
	buf := captureStdout(t)
	if code := Run([]string{"update", "check", "--json"}); code != 2 {
		t.Fatalf("json exit = %d, want 2", code)
	}
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, buf.String())
	}
	if parsed["latest"] != "0.3.0" || parsed["updateAvailable"] != true {
		t.Errorf("parsed = %v", parsed)
	}
}

func TestRunUpdateCheckDevVersion(t *testing.T) {
	setupUpdateTest(t, "dev", []mockRelease{})
	captureStdout(t)
	errBuf := captureStderr(t)
	if code := Run([]string{"update", "check"}); code != 1 {
		t.Fatalf("dev exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "无法确定当前版本") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

func TestRunUpdateCheckBadSubcommand(t *testing.T) {
	captureStdout(t)
	captureStderr(t)
	if code := Run([]string{"update", "install"}); code != 1 {
		t.Fatalf("bad subcommand exit = %d, want 1", code)
	}
}

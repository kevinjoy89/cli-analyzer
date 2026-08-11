package scanner

import "testing"

func TestClassifyVersioned(t *testing.T) {
	c := classify("/Users/wei/.local/share/claude/versions/2.1.226", "claude")
	if c.ToolID != "claude" || c.Installer != InstVersioned || c.CurrentVersion != "2.1.226" {
		t.Errorf("versioned: %+v", c)
	}
}

func TestClassifyBrew(t *testing.T) {
	// Requires a brew prefix on the test machine; fall back to HOMEBREW_PREFIX.
	c := classify(brewPrefix()+"/Cellar/gh/2.5.0/bin/gh", "gh")
	if c.ToolID != "gh" || c.Installer != InstBrew || c.CurrentVersion != "2.5.0" {
		t.Errorf("brew: %+v", c)
	}
}

func TestClassifyNpmScoped(t *testing.T) {
	c := classify("/opt/homebrew/lib/node_modules/@anthropic-ai/claude-code/bin/claude", "claude")
	if c.ToolID != "@anthropic-ai/claude-code" || c.Installer != InstNpm {
		t.Errorf("npm scoped: %+v", c)
	}
}

func TestClassifyPyenvShim(t *testing.T) {
	c := classify("/Users/wei/.pyenv/shims/python3", "python3")
	if c.ToolID != "pyenv" || c.Installer != InstPyenv {
		t.Errorf("pyenv shim: %+v", c)
	}
	c2 := classify("/Users/wei/.pyenv/versions/3.12.2/bin/python3", "python3")
	if c2.ToolID != "pyenv" || c2.Installer != InstPyenv {
		t.Errorf("pyenv build: %+v", c2)
	}
}

func TestClassifyGoCargoOther(t *testing.T) {
	if c := classify("/Users/wei/go/bin/dlv", "dlv"); c.Installer != InstGo {
		t.Errorf("go: %+v", c)
	}
	if c := classify("/Users/wei/.cargo/bin/cargo", "cargo"); c.Installer != InstCargo {
		t.Errorf("cargo: %+v", c)
	}
	if c := classify("/Users/wei/.local/bin/mytool", "mytool"); c.ToolID != "mytool" || c.Installer != InstOther {
		t.Errorf("other: %+v", c)
	}
}

func TestUnder(t *testing.T) {
	if !under("/a/b/c", "/a/b") {
		t.Error("/a/b/c should be under /a/b")
	}
	if !under("/a/b", "/a/b") {
		t.Error("/a/b should be under /a/b")
	}
	if under("/a/bc", "/a/b") {
		t.Error("/a/bc should NOT be under /a/b")
	}
}

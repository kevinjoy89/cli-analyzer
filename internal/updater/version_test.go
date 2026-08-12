package updater

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in      string
		want    Version
		wantErr bool
	}{
		{"0.2.3", Version{0, 2, 3}, false},
		{"v0.2.3", Version{0, 2, 3}, false},
		{"v1.0.0", Version{1, 0, 0}, false},
		{"10.20.30", Version{10, 20, 30}, false},
		{"0.2", Version{}, true},
		{"0.2.3.4", Version{}, true},
		{"abc", Version{}, true},
		{"0.2.x", Version{}, true},
		{"v0.3.0-beta.1", Version{}, true},
		{"", Version{}, true},
		{" 0.2.3 ", Version{0, 2, 3}, false},
	}
	for _, c := range cases {
		got, err := ParseVersion(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseVersion(%q) = %v, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVersion(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b Version
		want int
	}{
		{Version{0, 2, 3}, Version{0, 2, 3}, 0},
		{Version{1, 0, 0}, Version{0, 9, 9}, 1},
		{Version{0, 2, 3}, Version{0, 3, 0}, -1},
		{Version{0, 2, 3}, Version{0, 2, 4}, -1},
		{Version{0, 2, 4}, Version{0, 2, 3}, 1},
		{Version{0, 3, 0}, Version{0, 3, 0}, 0},
	}
	for _, c := range cases {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	if got := (Version{0, 2, 3}).String(); got != "0.2.3" {
		t.Errorf("String() = %q, want 0.2.3", got)
	}
}

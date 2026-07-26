package cli

import "testing"

func TestParseSize(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1073741824", 1 << 30, false},
		{"1GiB", 1 << 30, false},
		{"64MiB", 64 << 20, false},
		{"512MiB", 512 << 20, false},
		{"2KiB", 2 << 10, false},
		{"42", 42, false},
		{"0", 0, true},      // explicit zero: rejected (FR-005)
		{"-1GiB", 0, true},  // negative: rejected
		{"1.5GiB", 0, true}, // whole numbers only — budgets are exact
		{"1GB", 0, true},    // SI suffixes: rejected, binary only
		{"", 0, true},
		{"GiB", 0, true},
		{"999999999999GiB", 0, true}, // overflow
	}
	for _, c := range cases {
		got, err := parseSize(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "unlimited"},
		{1 << 30, "1.0 GiB"},
		{64 << 20, "64.0 MiB"},
		{512 << 20, "512.0 MiB"},
		{1536 << 20, "1.5 GiB"},
		{2 << 10, "2.0 KiB"},
		{42, "42 B"},
	}
	for _, c := range cases {
		if got := formatSize(c.in); got != c.want {
			t.Errorf("formatSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

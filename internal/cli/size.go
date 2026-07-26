package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// Binary size units. Budgets are exact byte counts, so parsing sticks to binary
// powers — mixed SI/binary parsing is a known source of quiet capacity mistakes.
const (
	kib = 1 << 10
	mib = 1 << 20
	gib = 1 << 30
)

// parseSize converts a whole number with an optional binary suffix ("1073741824",
// "64MiB", "1GiB") into bytes. Zero and negatives are rejected: a budget flag that
// is passed must mean a real roof (an explicit 0 could only masquerade as
// "unlimited", which is spelled by not passing the flag).
func parseSize(s string) (int64, error) {
	num, unit := s, int64(1)
	for _, u := range []struct {
		suffix string
		factor int64
	}{{"KiB", kib}, {"MiB", mib}, {"GiB", gib}} {
		if rest, ok := strings.CutSuffix(s, u.suffix); ok {
			num, unit = rest, u.factor
			break
		}
	}
	n, err := strconv.ParseInt(strings.TrimSpace(num), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("not a size (whole number with optional KiB/MiB/GiB): %q", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("size must be positive: %q", s)
	}
	if n > (1<<63-1)/unit {
		return 0, fmt.Errorf("size overflows: %q", s)
	}
	return n * unit, nil
}

// formatSize renders a byte roof for the provision report: the largest binary unit
// with one decimal, or "unlimited" for zero.
func formatSize(v int64) string {
	switch {
	case v == 0:
		return "unlimited"
	case v >= gib:
		return fmt.Sprintf("%.1f GiB", float64(v)/gib)
	case v >= mib:
		return fmt.Sprintf("%.1f MiB", float64(v)/mib)
	case v >= kib:
		return fmt.Sprintf("%.1f KiB", float64(v)/kib)
	default:
		return fmt.Sprintf("%d B", v)
	}
}

// sizeFlag is a flag.Value for budget flags: remembers whether it was set, so the
// provision command can tell "not passed" from any parsed value.
type sizeFlag struct {
	set   bool
	bytes int64
}

func (f *sizeFlag) String() string {
	if !f.set {
		return ""
	}
	return formatSize(f.bytes)
}

func (f *sizeFlag) Set(s string) error {
	v, err := parseSize(s)
	if err != nil {
		return err
	}
	f.set, f.bytes = true, v
	return nil
}

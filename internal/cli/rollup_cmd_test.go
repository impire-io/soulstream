package cli

import (
	"strings"
	"testing"
)

func TestRollupCommand(t *testing.T) {
	connect := testConnector(t)
	base := []string{"--realm", "acme", "--persona", "daan"}
	run(connect, append(base, "provision")...)

	_, out, _ := run(connect, append(base, "start", "Long Topic")...)
	path := strings.TrimSpace(out)
	for _, body := range []string{"one", "two", "three"} {
		if code, _, errs := run(connect, append(base, "post", path, body)...); code != 0 {
			t.Fatalf("post: %s", errs)
		}
	}

	code, out, errs := run(connect, append(base, "rollup", path)...)
	if code != 0 {
		t.Fatalf("rollup exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "compacted "+path) {
		t.Errorf("rollup output: %q", out)
	}

	// The view is intact after compaction.
	code, out, errs = run(connect, append(base, "show", path)...)
	if code != 0 {
		t.Fatalf("show exit %d: %s", code, errs)
	}
	for _, body := range []string{"one", "two", "three"} {
		if !strings.Contains(out, body) {
			t.Errorf("post-rollup show missing %q:\n%s", body, out)
		}
	}

	// Nothing left to compact: friendly no-op, exit 0.
	code, out, errs = run(connect, append(base, "rollup", path)...)
	if code != 0 {
		t.Fatalf("second rollup exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "nothing to compact") {
		t.Errorf("second rollup output: %q", out)
	}

	// A rollup needs a persona (it publishes).
	if code, _, _ := run(connect, "--realm", "acme", "rollup", path); code == 0 {
		t.Error("rollup without persona should fail")
	}
}

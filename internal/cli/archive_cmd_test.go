package cli

import (
	"strings"
	"testing"
)

func TestCloseCompactsAndArchiveIsFinal(t *testing.T) {
	connect := testConnector(t)
	base := []string{"--realm", "acme", "--persona", "daan"}
	run(connect, append(base, "provision")...)

	// close tidies up: the topic still reads fully afterwards.
	_, out, _ := run(connect, append(base, "start", "To Close")...)
	closePath := strings.TrimSpace(out)
	run(connect, append(base, "post", closePath, "some content")...)
	if code, _, errs := run(connect, append(base, "close", closePath)...); code != 0 {
		t.Fatalf("close: %s", errs)
	}
	code, out, errs := run(connect, append(base, "show", closePath)...)
	if code != 0 {
		t.Fatalf("show after close: %s", errs)
	}
	if !strings.Contains(out, "lifecycle: closed") || !strings.Contains(out, "some content") {
		t.Errorf("closed topic reads wrong:\n%s", out)
	}

	// archive: loud, terminal, refuses everything after.
	_, out, _ = run(connect, append(base, "start", "To Archive")...)
	archPath := strings.TrimSpace(out)
	run(connect, append(base, "post", archPath, "last words")...)

	code, out, errs = run(connect, append(base, "archive", archPath)...)
	if code != 0 {
		t.Fatalf("archive: %s", errs)
	}
	if !strings.Contains(out, "archived "+archPath) || !strings.Contains(out, "read-only") {
		t.Errorf("archive output: %q", out)
	}

	// Reads still work, forever.
	code, out, errs = run(connect, append(base, "show", archPath)...)
	if code != 0 {
		t.Fatalf("show after archive: %s", errs)
	}
	if !strings.Contains(out, "lifecycle: archived") || !strings.Contains(out, "last words") {
		t.Errorf("archived topic reads wrong:\n%s", out)
	}

	// Writes refuse with a clear error.
	for _, attempt := range [][]string{
		append(base, "post", archPath, "nope"),
		append(base, "comment", archPath, "some-op", "nope"),
		append(base, "close", archPath),
		append(base, "rollup", archPath),
		append(base, "archive", archPath),
	} {
		code, _, errs := run(connect, attempt...)
		if code == 0 {
			t.Errorf("%v succeeded on an archived topic", attempt[len(attempt)-2:])
		}
		if !strings.Contains(errs, "archived") {
			t.Errorf("%v error does not name archival: %q", attempt[len(attempt)-2:], errs)
		}
	}
}

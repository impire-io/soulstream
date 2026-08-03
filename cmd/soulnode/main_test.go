package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestInitPrintsTokenExactlyOnce is the CLI contract's founding block:
// one token on the founding run, none on the verify run.
func TestInitPrintsTokenExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	var out, errw bytes.Buffer
	if code := run([]string{"init", "--state", dir, "--listen", "127.0.0.1:0", "--door-listen", "127.0.0.1:0", "--fold-listen", "127.0.0.1:0"}, &out, &errw); code != 0 {
		t.Fatalf("init exit %d: %s", code, errw.String())
	}
	if got := strings.Count(out.String(), "sit_"); got != 1 {
		t.Fatalf("founding run printed %d tokens, want exactly 1:\n%s", got, out.String())
	}
	if !strings.Contains(out.String(), "shown once") {
		t.Fatal("founding output lacks the shown-once warning")
	}

	out.Reset()
	errw.Reset()
	if code := run([]string{"init", "--state", dir}, &out, &errw); code != 0 {
		t.Fatalf("re-init exit %d: %s", code, errw.String())
	}
	if strings.Contains(out.String(), "sit_") {
		t.Fatalf("re-init printed a token:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "verified") {
		t.Fatalf("re-init did not report verification:\n%s", out.String())
	}
}

// TestUpRefusesUninitialized: up never implicitly initializes.
func TestUpRefusesUninitialized(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"up", "--state", t.TempDir()}, &out, &errw); code != 1 {
		t.Fatalf("up on empty state exited %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "init") {
		t.Fatalf("refusal does not point at init: %s", errw.String())
	}
}

// TestWorkloadRefusals is SC-003: the three refusal paths, each fast and
// named. One founded state dir is shared; the node is never up.
func TestWorkloadRefusals(t *testing.T) {
	dir := t.TempDir() + "/state"
	var out, errw bytes.Buffer
	if code := run([]string{"init", "--state", dir, "--listen", "127.0.0.1:0", "--door-listen", "127.0.0.1:0", "--fold-listen", "127.0.0.1:0"}, &out, &errw); code != 0 {
		t.Fatalf("init exit %d: %s", code, errw.String())
	}

	write := func(name, body string) string {
		t.Helper()
		p := t.TempDir() + "/" + name
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"bad-declaration",
			[]string{"workload", "start", "--state", dir,
				write("bad.json", `{"role":"alien","lifecycle":"service","persona":"x","topic":"t","artifact":"file:///bin/sh"}`)},
			"role"},
		{"missing-artifact",
			[]string{"workload", "start", "--state", dir,
				write("noart.json", `{"role":"agent","lifecycle":"service","persona":"x","topic":"t","artifact":"file:///nonexistent-artifact"}`)},
			"artifact"},
		{"node-down",
			[]string{"workload", "start", "--state", dir,
				write("ok.json", `{"role":"agent","lifecycle":"service","persona":"x","topic":"t","artifact":"file:///bin/sh"}`)},
			"soulnode up"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errw bytes.Buffer
			if code := run(tc.args, &out, &errw); code != 1 {
				t.Fatalf("exit %d, want 1 (out %s, err %s)", code, out.String(), errw.String())
			}
			if !strings.Contains(errw.String(), tc.want) {
				t.Fatalf("refusal does not name %q: %s", tc.want, errw.String())
			}
		})
	}
}

func TestVersion(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"version"}, &out, &errw); code != 0 {
		t.Fatalf("version exit %d", code)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("version printed nothing")
	}
}

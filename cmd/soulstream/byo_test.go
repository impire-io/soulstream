package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

// TestInitBYOPhaseOneEmitsTheKit: the self-hosted founding's first run
// writes and prints the kit, exits 0, and re-runs re-emit it unchanged —
// idempotent, never an error, never a token (spec 010 US1.1/US1.4).
func TestInitBYOPhaseOneEmitsTheKit(t *testing.T) {
	dir := t.TempDir()
	var out, errw bytes.Buffer
	if code := run([]string{"init", "--state", dir, "--byo", "self-hosted",
		"--url", "nats://byo.example:4222", "--realm", "home"}, &out, &errw); code != 0 {
		t.Fatalf("phase 1 exit %d: %s", code, errw.String())
	}
	for _, want := range []string{"nsc add account soulstream-home", "nsc push -A", "--auth-account"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("phase-1 output lacks %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "sit_") {
		t.Fatal("phase 1 printed a token — no founding happened yet")
	}
	kit, err := os.ReadFile(ceremony.KitPath(dir))
	if err != nil {
		t.Fatalf("kit not written: %v", err)
	}

	// Re-run without the hand-back: same kit, still exit 0.
	out.Reset()
	errw.Reset()
	if code := run([]string{"init", "--state", dir}, &out, &errw); code != 0 {
		t.Fatalf("awaiting re-run exit %d: %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "hand back") {
		t.Fatalf("awaiting re-run lacks the guidance:\n%s", out.String())
	}
	kitAgain, _ := os.ReadFile(ceremony.KitPath(dir))
	if string(kit) != string(kitAgain) {
		t.Fatal("re-run regenerated a different kit")
	}

	// `up` before the account half names the phase, not damage.
	out.Reset()
	errw.Reset()
	if code := run([]string{"up", "--state", dir}, &out, &errw); code != 1 ||
		!strings.Contains(errw.String(), "awaits its account half") {
		t.Fatalf("up on awaiting state: exit %d, %s", code, errw.String())
	}
}

// TestInitBYOFlagRefusals: the flags refuse by name (design 0003 §6's
// mutual exclusions, the flavour set, the hand-back's timing).
func TestInitBYOFlagRefusals(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"url without byo", []string{"--url", "nats://x:4222"}, "BYO flags"},
		{"unknown flavour", []string{"--byo", "ngs"}, "flavour"},
		{"listen with byo", []string{"--byo", "self-hosted", "--url", "nats://x:4222", "--listen", "127.0.0.1:4222"}, "embedded server's flag"},
		{"hand-back on run one", []string{"--byo", "self-hosted", "--url", "nats://x:4222", "--auth-account", "AX"}, "second run"},
		{"byo without url", []string{"--byo", "self-hosted"}, "--url"},
		{"synadia without system", []string{"--byo", "synadia-cloud", "--url", "nats://x:4222"}, "--synadia-system"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errw bytes.Buffer
			args := append([]string{"init", "--state", t.TempDir()}, tc.args...)
			if code := run(args, &out, &errw); code == 0 {
				t.Fatalf("exit 0, wanted a refusal:\n%s", out.String())
			}
			if !strings.Contains(errw.String(), tc.want) {
				t.Fatalf("refusal does not name %q: %s", tc.want, errw.String())
			}
		})
	}
}

// TestInitBYOSynadiaNeedsToken: the synadia flavour without its PAT is
// refused by name before anything is created.
func TestInitBYOSynadiaNeedsToken(t *testing.T) {
	t.Setenv("SOULSTREAM_SYNADIA_TOKEN", "")
	var out, errw bytes.Buffer
	code := run([]string{"init", "--state", t.TempDir(), "--byo", "synadia-cloud",
		"--url", "nats://byon.example:4222", "--synadia-system", "dev"}, &out, &errw)
	if code == 0 || !strings.Contains(errw.String(), "SOULSTREAM_SYNADIA_TOKEN") {
		t.Fatalf("missing token not refused by name: exit %d, %s", code, errw.String())
	}
}

// TestInitBYOSubstrateFixedAtFounding: an embedded realm refuses --byo,
// and a BYO realm refuses --listen and a different flavour.
func TestInitBYOSubstrateFixedAtFounding(t *testing.T) {
	embedded := t.TempDir()
	var out, errw bytes.Buffer
	if code := run([]string{"init", "--state", embedded, "--listen", "127.0.0.1:0",
		"--mcp-listen", "127.0.0.1:0", "--signin-listen", "127.0.0.1:0"}, &out, &errw); code != 0 {
		t.Fatalf("embedded init: %s", errw.String())
	}
	out.Reset()
	errw.Reset()
	code := run([]string{"init", "--state", embedded, "--byo", "self-hosted", "--url", "nats://x:4222"}, &out, &errw)
	if code == 0 || !strings.Contains(errw.String(), "fixed at founding") {
		t.Fatalf("byo on embedded state not refused: exit %d, %s", code, errw.String())
	}

	byoDir := t.TempDir()
	out.Reset()
	errw.Reset()
	if code := run([]string{"init", "--state", byoDir, "--byo", "self-hosted",
		"--url", "nats://byo.example:4222"}, &out, &errw); code != 0 {
		t.Fatalf("byo phase 1: %s", errw.String())
	}
	out.Reset()
	errw.Reset()
	code = run([]string{"init", "--state", byoDir, "--byo", "synadia-cloud"}, &out, &errw)
	if code == 0 || !strings.Contains(errw.String(), "fixed at founding") {
		t.Fatalf("flavour change not refused: exit %d, %s", code, errw.String())
	}
	out.Reset()
	errw.Reset()
	code = run([]string{"init", "--state", byoDir, "--url", "nats://other.example:4222"}, &out, &errw)
	if code == 0 || !strings.Contains(errw.String(), "already dials") {
		t.Fatalf("url change not refused: exit %d, %s", code, errw.String())
	}
}

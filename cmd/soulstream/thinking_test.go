package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

// The thinking house's three verbs refuse before they dial, and each
// refusal names what to do next (specs/014 FR-010). One founded state dir
// is shared; the node is never up.
func TestThinkingVerbRefusals(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	var out, errw bytes.Buffer
	if code := run([]string{"init", "--state", dir, "--listen", "127.0.0.1:0",
		"--mcp-listen", "127.0.0.1:0", "--signin-listen", "127.0.0.1:0"}, &out, &errw); code != 0 {
		t.Fatalf("init exit %d: %s", code, errw.String())
	}
	write := func(name, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	refuse := func(t *testing.T, args []string, want string) {
		t.Helper()
		var out, errw bytes.Buffer
		if code := run(args, &out, &errw); code != 1 {
			t.Fatalf("%v exited %d, want 1 (out %s, err %s)", args, code, out.String(), errw.String())
		}
		if !strings.Contains(errw.String(), want) {
			t.Fatalf("refusal does not name %q: %s", want, errw.String())
		}
	}

	agent := write("agent.json",
		`{"role":"agent","lifecycle":"service","persona":"thinker","topic":"t-ab12",`+
			`"artifact":"file:///dev/null","wake":[{"kind":"mention"}]}`)

	// A realm with no dispatcher plane: nothing would ever serve the
	// placement, and the operator hears it now rather than later.
	refuse(t, []string{"agent", "submit", "--state", dir, agent}, "no dispatcher plane")
	refuse(t, []string{"provider", "set", "--state", dir, "anthropic"}, providerKeyEnv)

	// The operator turns the plane on — the ordinary act, config.json is
	// the configuration.
	st, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	st.DispatcherEnabled = true
	st.DispatcherPlacements = ceremony.DefaultPlacements
	st.DispatcherHarness = "claude"
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := ceremony.Verify(dir); err != nil {
		t.Fatalf("the dispatcher block did not survive a round trip: %v", err)
	}

	// A declaration nothing wakes belongs to the other verb.
	refuse(t, []string{"agent", "submit", "--state", dir,
		write("nowake.json", `{"role":"agent","lifecycle":"service","persona":"x","topic":"t-ab12","artifact":"file:///dev/null"}`)},
		"soulstream workload start")
	// And with the plane on and the node down, every verb that talks to a
	// running realm says so in the same words.
	refuse(t, []string{"agent", "submit", "--state", dir, agent}, "soulstream up")
	refuse(t, []string{"model", "ls", "--state", dir}, "soulstream up")
	t.Setenv(providerKeyEnv, "sk-not-a-real-key")
	refuse(t, []string{"provider", "set", "--state", dir, "anthropic"}, "soulstream up")

	// Argument shapes, refused without touching any state at all.
	refuse(t, []string{"agent", "place", "--state", dir, agent}, "submit")
	refuse(t, []string{"model", "rename", "--state", dir}, "set | ls")
	refuse(t, []string{"provider", "list", "--state", dir}, "set")
}

// The usage text carries the thinking house's verbs, so a person holding
// only the binary can find them.
func TestUsageCarriesTheThinkingVerbs(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"help"}, &out, &errw); code != 0 {
		t.Fatalf("help exit %d", code)
	}
	for _, want := range []string{"soulstream agent submit", "soulstream model set", "soulstream provider set"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("usage does not carry %q:\n%s", want, out.String())
		}
	}
}

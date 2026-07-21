package cli

import (
	"strings"
	"testing"
)

// TestWorkCommands walks US2/US3 through the CLI: open, race two claims (the
// verdict names the winner), evidence in `work show`, done, and abandon/reclaim.
func TestWorkCommands(t *testing.T) {
	connect := testConnector(t)
	run(connect, "--realm", "acme", "--persona", "daan", "provision")

	_, out, _ := run(connect, "--realm", "acme", "--persona", "daan", "start", "gadget plan")
	path := strings.TrimSpace(out)

	// Open an item.
	code, out, errs := run(connect, "--realm", "acme", "--persona", "daan",
		"work", "open", path, "draft the intro", "--body", "any takers?")
	if code != 0 {
		t.Fatalf("work open exit %d: %s", code, errs)
	}
	itemID := strings.TrimSpace(out)
	if itemID == "" {
		t.Fatal("work open printed no item id")
	}

	// First claim wins…
	code, out, errs = run(connect, "--realm", "acme", "--persona", "scribe", "work", "claim", path, itemID)
	if code != 0 || !strings.Contains(out, "claimed — you own it") {
		t.Fatalf("winning claim: exit %d, out %q, err %q", code, out, errs)
	}
	// …the later claim is void, and the loser is told who owns it.
	code, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "work", "claim", path, itemID)
	if code != 0 || !strings.Contains(out, "void — owned by scribe") {
		t.Fatalf("losing claim: exit %d, out %q", code, out)
	}

	_, out, _ = run(connect, "--realm", "acme", "work", "list", path)
	if !strings.Contains(out, "claimed") || !strings.Contains(out, "scribe") || !strings.Contains(out, "draft the intro") {
		t.Errorf("work list = %q", out)
	}

	// Evidence anchors to the item; `work show` surfaces it with the void claim.
	if code, _, errs = run(connect, "--realm", "acme", "--persona", "scribe", "comment", path, itemID, "on it"); code != 0 {
		t.Fatalf("evidence comment exit %d: %s", code, errs)
	}
	code, _, errs = run(connect, "--realm", "acme", "--persona", "scribe", "work", "done", path, itemID)
	if code != 0 {
		t.Fatalf("work done exit %d: %s", code, errs)
	}
	_, out, _ = run(connect, "--realm", "acme", "work", "show", path, itemID)
	if !strings.Contains(out, "status: done") || !strings.Contains(out, "VOID") || !strings.Contains(out, "on it") {
		t.Errorf("work show = %q", out)
	}

	// A claim on a done item is void with a reason.
	_, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "work", "claim", path, itemID)
	if !strings.Contains(out, "void — the item is already done") {
		t.Errorf("claim on done = %q", out)
	}

	// Abandon reopens; the next claim wins fresh.
	_, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "work", "open", path, "polish diagrams")
	item2 := strings.TrimSpace(out)
	run(connect, "--realm", "acme", "--persona", "scribe", "work", "claim", path, item2)
	code, out, _ = run(connect, "--realm", "acme", "--persona", "scribe", "work", "abandon", path, item2)
	if code != 0 || !strings.Contains(out, "abandoned") {
		t.Fatalf("work abandon: exit %d, out %q", code, out)
	}
	_, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "work", "claim", path, item2)
	if !strings.Contains(out, "claimed — you own it") {
		t.Errorf("reclaim after abandon = %q", out)
	}

	// `show` includes the work items section.
	_, out, _ = run(connect, "--realm", "acme", "show", path)
	if !strings.Contains(out, "work items:") {
		t.Errorf("show missing work items: %q", out)
	}
}

// TestWorkCommandUsage: missing arguments are usage errors, not crashes.
func TestWorkCommandUsage(t *testing.T) {
	connect := testConnector(t)
	if code, _, _ := run(connect, "--realm", "acme", "work"); code != 2 {
		t.Errorf("bare work exit = %d, want 2", code)
	}
	if code, _, _ := run(connect, "--realm", "acme", "--persona", "daan", "work", "open", "some-path"); code != 2 {
		t.Errorf("work open without title exit = %d, want 2", code)
	}
	if code, _, _ := run(connect, "--realm", "acme", "--persona", "daan", "work", "claim", "some-path"); code != 2 {
		t.Errorf("work claim without id exit = %d, want 2", code)
	}
	if code, _, _ := run(connect, "--realm", "acme", "work", "bogus"); code != 2 {
		t.Errorf("unknown subcommand exit = %d, want 2", code)
	}
}

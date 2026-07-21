package cli

import (
	"os"
	"strings"
	"testing"
)

// TestUpkeepCommands: reply, edit (own words only), resolve, detach, and
// mark-dormant through the CLI.
func TestUpkeepCommands(t *testing.T) {
	connect := testConnector(t)
	run(connect, "--realm", "acme", "--persona", "daan", "provision")

	_, out, _ := run(connect, "--realm", "acme", "--persona", "daan", "start", "gadget plan")
	path := strings.TrimSpace(out)

	_, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "post", path, "lets ship thursdy")
	turnID := strings.TrimSpace(out)

	// Edit own words; the view renders the correction with the marker.
	if code, _, errs := run(connect, "--realm", "acme", "--persona", "daan", "edit", path, turnID, "let's ship Thursday"); code != 0 {
		t.Fatalf("edit exit %d: %s", code, errs)
	}
	_, out, _ = run(connect, "--realm", "acme", "show", path)
	if !strings.Contains(out, "let's ship Thursday") || !strings.Contains(out, "edited") {
		t.Errorf("show after edit = %q", out)
	}

	// Comment, reply, resolve.
	_, out, _ = run(connect, "--realm", "acme", "--persona", "scribe", "comment", path, turnID, "which Thursday?")
	cmntID := strings.TrimSpace(out)
	if code, _, errs := run(connect, "--realm", "acme", "--persona", "daan", "reply", path, cmntID, "the 30th"); code != 0 {
		t.Fatalf("reply exit %d: %s", code, errs)
	}
	code, out, _ := run(connect, "--realm", "acme", "--persona", "daan", "resolve", path, cmntID)
	if code != 0 || !strings.Contains(out, "resolved") {
		t.Fatalf("resolve: exit %d, out %q", code, out)
	}
	_, out, _ = run(connect, "--realm", "acme", "show", path)
	if !strings.Contains(out, "(reply -> ") || !strings.Contains(out, "resolved by daan") {
		t.Errorf("show after reply/resolve = %q", out)
	}

	// A foreign edit lands but the view shows the original, with a warning in
	// the JSON view.
	run(connect, "--realm", "acme", "--persona", "scribe", "edit", path, turnID, "let's ship Friday")
	_, out, _ = run(connect, "--realm", "acme", "show", path, "--json")
	if !strings.Contains(out, "let's ship Thursday") || !strings.Contains(out, "only the author may edit") {
		t.Errorf("foreign edit handling = %q", out)
	}

	// Detach: withdrawn marker in show; artefact list drops the lineage.
	dir := t.TempDir()
	file := dir + "/x.md"
	if err := writeFile(file, "bytes"); err != nil {
		t.Fatal(err)
	}
	run(connect, "--realm", "acme", "--persona", "daan", "attach", path, file)
	_, out, _ = run(connect, "--realm", "acme", "show", path, "--json")
	attnID := extractOpID(t, out, "x.md")
	code, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "detach", path, attnID)
	if code != 0 || !strings.Contains(out, "withdrawn") {
		t.Fatalf("detach: exit %d, out %q", code, out)
	}
	_, out, _ = run(connect, "--realm", "acme", "show", path)
	if !strings.Contains(out, "removed by daan") {
		t.Errorf("show after detach = %q", out)
	}
	_, out, _ = run(connect, "--realm", "acme", "artefacts", path)
	if strings.Contains(out, "x.md") {
		t.Errorf("fully-withdrawn artefact still listed: %q", out)
	}

	// mark-dormant: fresh topic reports not idle and posts nothing; with a zero
	// window it marks; a post wakes it.
	code, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "mark-dormant", path)
	if code != 0 || !strings.Contains(out, "not idle") {
		t.Fatalf("mark-dormant fresh: exit %d, out %q", code, out)
	}
	code, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "mark-dormant", path, "--idle", "1ns")
	if code != 0 || !strings.Contains(out, "marked dormant") {
		t.Fatalf("mark-dormant idle: exit %d, out %q", code, out)
	}
	_, out, _ = run(connect, "--realm", "acme", "--persona", "daan", "mark-dormant", path, "--idle", "1ns")
	if !strings.Contains(out, "already dormant") {
		t.Errorf("second mark = %q", out)
	}
	run(connect, "--realm", "acme", "--persona", "daan", "post", path, "awake")
	_, out, _ = run(connect, "--realm", "acme", "show", path)
	if !strings.Contains(out, "lifecycle: active") {
		t.Errorf("post did not wake the topic: %q", out)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// extractOpID pulls the op_id of the attachment named name from show --json output.
func extractOpID(t *testing.T, jsonOut, name string) string {
	t.Helper()
	idx := strings.Index(jsonOut, name)
	if idx < 0 {
		t.Fatalf("attachment %s not in view: %s", name, jsonOut)
	}
	// op_id precedes the name in the attachment object; scan backwards.
	head := jsonOut[:idx]
	key := `"op_id": "`
	at := strings.LastIndex(head, key)
	if at < 0 {
		t.Fatalf("no op_id before %s", name)
	}
	rest := head[at+len(key):]
	end := strings.Index(rest, `"`)
	return rest[:end]
}

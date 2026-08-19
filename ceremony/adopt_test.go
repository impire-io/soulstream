package ceremony_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

// stripVersion makes a founded directory look pre-break: exactly the
// shape a realm founded before hq episode 0112 has on disk.
func stripVersion(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	delete(cfg, "record_version")
	out, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestAdoptStampsAndPreservesEverything (hq 0112): adoption is the one
// act that resolves the pre-v2 refusal — it stamps the version and
// leaves every other field founding wrote exactly as it was, including
// keys the running build does not know about.
func TestAdoptStampsAndPreservesEverything(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	stripVersion(t, dir)

	// A field this build has no struct member for: adoption must not
	// drop it, because it may be a newer build's or an operator's.
	path := filepath.Join(dir, "config.json")
	raw, _ := os.ReadFile(path)
	var before map[string]any
	if err := json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	before["operator_note"] = "keep me"
	out, _ := json.Marshal(before)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}

	pre, err := ceremony.PreV2(dir)
	if err != nil || !pre {
		t.Fatalf("PreV2 = %v (%v), want true", pre, err)
	}
	if _, err := ceremony.Load(dir); err == nil {
		t.Fatal("a pre-v2 directory loaded before adoption")
	}

	if err := ceremony.AdoptV2(dir); err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if pre, err := ceremony.PreV2(dir); err != nil || pre {
		t.Fatalf("still pre-v2 after adoption (%v)", err)
	}
	if _, err := ceremony.Load(dir); err != nil {
		t.Fatalf("an adopted directory does not load: %v", err)
	}

	raw, _ = os.ReadFile(path)
	var after map[string]any
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatal(err)
	}
	if after["operator_note"] != "keep me" {
		t.Fatal("adoption dropped a field it did not recognise")
	}
	if after["realm"] != before["realm"] {
		t.Fatalf("adoption changed the realm name: %v -> %v", before["realm"], after["realm"])
	}
	if got, ok := after["record_version"].(float64); !ok || int(got) != ceremony.RecordVersion() {
		t.Fatalf("record_version = %v, want %d", after["record_version"], ceremony.RecordVersion())
	}

	// Idempotent: adopting an adopted directory is a no-op.
	if err := ceremony.AdoptV2(dir); err != nil {
		t.Fatalf("second adopt: %v", err)
	}
}

// TestOpsConnectionNamesTheRealmsServer: maintenance acts reach the
// realm's own server — the BYO url when there is one, the configured
// listener otherwise — with the ops credential founding left behind.
func TestOpsConnectionNamesTheRealmsServer(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:4222", "home")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	url, creds, err := ceremony.OpsConnection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if url != "nats://127.0.0.1:4222" {
		t.Fatalf("url = %q", url)
	}
	if !strings.HasSuffix(creds, "ops.creds") {
		t.Fatalf("creds = %q, want the ops credential", creds)
	}
}

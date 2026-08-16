package ceremony_test

// Pre-v1 renames are clean breaks (design 0001 §2): one schema, one
// code path. A fresh found writes only the functional names, and a
// realm founded under the byname-era spellings is refused by name —
// with the hand-migration in the refusal — never silently misread.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

func TestBynameEraShapesAreRefusedByName(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	st.SignInEnabled = true
	st.SignInListen = "127.0.0.1:8378"
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}

	// A fresh found writes the functional names and nothing else.
	cfgRaw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"signin"`, `"mcp"`} {
		if !strings.Contains(string(cfgRaw), want) {
			t.Fatalf("a fresh found does not write %s:\n%s", want, cfgRaw)
		}
	}
	for _, banned := range []string{`"fold"`, `"door"`} {
		if strings.Contains(string(cfgRaw), banned) {
			t.Fatalf("a fresh found still writes %s:\n%s", banned, cfgRaw)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "users", "signin.creds")); err != nil {
		t.Fatalf("a fresh found writes no users/signin.creds: %v", err)
	}

	// Reshape the directory to the byname era: loading refuses by name
	// and says how to migrate, rather than silently dropping the plane.
	var cfg map[string]any
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		t.Fatal(err)
	}
	planes := cfg["planes"].(map[string]any)
	planes["door"] = planes["mcp"]
	delete(planes, "mcp")
	planes["fold"] = planes["signin"]
	delete(planes, "signin")
	legacyRaw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), legacyRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ceremony.Load(dir); err == nil ||
		!strings.Contains(err.Error(), "byname-era") ||
		!strings.Contains(err.Error(), "fold→signin") {
		t.Fatalf("the byname-era shape was not refused with the migration named: %v", err)
	}

	// The refusal names the whole break, not half of it: with the config
	// migrated but the creds file still under its old name, the missing
	// functional file is the refusal.
	if err := os.WriteFile(filepath.Join(dir, "config.json"), cfgRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "users", "signin.creds"),
		filepath.Join(dir, "users", "fold.creds")); err != nil {
		t.Fatal(err)
	}
	if _, err := ceremony.Load(dir); err == nil ||
		!strings.Contains(err.Error(), "users/signin.creds") {
		t.Fatalf("a missing signin.creds with the plane enabled was not refused by name: %v", err)
	}
}

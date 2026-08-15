package ceremony_test

// A realm founded before the functional plane names keeps working
// untouched (design 0001 §2): the byname-era config keys and creds
// filename are read forever, and a fresh found writes only the
// functional names.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

func TestLegacyNamesAreReadForever(t *testing.T) {
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

	// Reshape the same directory to the byname era: legacy keys, legacy
	// creds filename. Loading must see the identical realm.
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
	if err := os.Rename(filepath.Join(dir, "users", "signin.creds"),
		filepath.Join(dir, "users", "fold.creds")); err != nil {
		t.Fatal(err)
	}

	loaded, err := ceremony.Load(dir)
	if err != nil {
		t.Fatalf("the byname-era shape no longer loads: %v", err)
	}
	if !loaded.SignInEnabled || loaded.SignInListen != "127.0.0.1:8378" {
		t.Fatalf("the legacy sign-in block loaded differently: enabled=%v listen=%q",
			loaded.SignInEnabled, loaded.SignInListen)
	}
	if len(loaded.SignInCreds) == 0 {
		t.Fatal("the legacy creds filename was not read")
	}
	if !loaded.MCPEnabled {
		t.Fatal("the legacy door block did not enable the MCP plane")
	}
}

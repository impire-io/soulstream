package ceremony

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadRefusesClientConfigDir: the soulstream client tooling keeps
// context/persona config and ed25519 persona keys under the same
// default path the product's state dir resolves to. Landing init there
// is a named refusal pointing at --state — never a missing-file error,
// and never a write into a directory the client owns (first hit
// 2026-08-16: a machine that used the component CLI before the
// product).
func TestLoadRefusesClientConfigDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"context":"personal","realm":"soulstream","persona":"daan"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "keys"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keys", "soulstream-daan.ed25519"),
		[]byte("not a ceremony artifact"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("client config dir loaded as a ceremony")
	}
	for _, want := range []string{"client's configuration", `context "personal"`, "--state"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal does not carry %q: %v", want, err)
		}
	}
	// And nothing was written: the directory stays exactly the client's.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("the refusal left writes behind: %v", entries)
	}
}

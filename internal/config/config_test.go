package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFileStrictAndRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ProjectFileName)

	write(t, path, `{"realm":"acme","key_file":"./keys/ci.ed25519","pins_file":"/abs/pins.json"}`)
	f, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Realm != "acme" {
		t.Errorf("realm = %q", f.Realm)
	}
	if want := filepath.Join(dir, "keys", "ci.ed25519"); f.KeyFile != want {
		t.Errorf("relative key_file = %q, want %q (resolved against the file's dir)", f.KeyFile, want)
	}
	if f.PinsFile != "/abs/pins.json" {
		t.Errorf("absolute pins_file changed: %q", f.PinsFile)
	}

	// Empty object contributes nothing and is not an error.
	write(t, path, `{}`)
	if f, err = loadFile(path); err != nil || f != (File{}) {
		t.Errorf("empty object: %+v, %v", f, err)
	}

	// Malformed JSON and unknown fields fail loud, naming the file.
	for _, bad := range []string{`{not json`, `{"presona":"x"}`} {
		write(t, path, bad)
		if _, err := loadFile(path); err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("content %q: err = %v, want error naming %s", bad, err, path)
		}
	}
}

func TestFindProjectFileNearestOnly(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, ok := findProjectFile(deep); ok {
		t.Fatal("found a project file in an empty tree")
	}

	outer := filepath.Join(root, ProjectFileName)
	write(t, outer, `{}`)
	if got, ok := findProjectFile(deep); !ok || got != outer {
		t.Errorf("walk-up: got %q ok=%v, want %q", got, ok, outer)
	}

	// A nearer file shadows the outer one entirely.
	inner := filepath.Join(root, "a", "b", ProjectFileName)
	write(t, inner, `{}`)
	if got, _ := findProjectFile(deep); got != inner {
		t.Errorf("nearest: got %q, want %q", got, inner)
	}
}

package node_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCycleGuard (T037): the dependency rule that makes the adapter position
// safe (FR-014). The node module imports BOTH core repos; neither core repo
// imports the other or the node. Asserted from the module files, the mirror
// of soulstream's 017 rule.
func TestCycleGuard(t *testing.T) {
	repoRoot := "../"     // the soulstream core module
	nodeMod := "./go.mod" // this module

	// 1. The core soulstream module must not depend on soulidentity or the node.
	coreMod := readFile(t, filepath.Join(repoRoot, "go.mod"))
	for _, forbidden := range []string{"impire-io/soulidentity", "impire-io/soulstream/node"} {
		if strings.Contains(coreMod, forbidden) {
			t.Errorf("core soulstream go.mod depends on %q — the cycle guard forbids it", forbidden)
		}
	}

	// 2. The node module imports both core repos (it is the adapter position).
	nm := readFile(t, nodeMod)
	for _, want := range []string{"impire-io/soulidentity", "impire-io/soulstream"} {
		if !strings.Contains(nm, want) {
			t.Errorf("node go.mod is missing %q — the node is the consumer of both", want)
		}
	}

	// 3. The promoted mcpserver package (core) imports no node code — the
	//    surface is embeddable downward, never upward.
	for _, f := range goFiles(t, filepath.Join(repoRoot, "mcpserver")) {
		src := readFile(t, f)
		if strings.Contains(src, "impire-io/soulstream/node") {
			t.Errorf("%s imports the node module — mcpserver must not depend on its embedder", f)
		}
		if strings.Contains(src, "impire-io/soulidentity") {
			t.Errorf("%s imports soulidentity — the core must stay identity-plane-agnostic", f)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

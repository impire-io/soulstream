package ceremony_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulnode/ceremony"
)

// TestHelmWiring covers the helm plane's config contract: on by default
// at founding, absent-block-means-disabled on old state dirs, loopback
// and collision refusals, and the sign-in issuer requirement.
func TestHelmWiring(t *testing.T) {
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	if !st.HelmEnabled || st.HelmListen != "127.0.0.1:8500" {
		t.Fatalf("founding defaults: enabled=%v listen=%q", st.HelmEnabled, st.HelmListen)
	}
	if iss, aud := st.SessionIssuer(); iss != st.FoldIssuer || aud != st.FoldAudience {
		t.Fatalf("session issuer = %q/%q, want the bundled fold", iss, aud)
	}

	dir := t.TempDir()
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	back, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !back.HelmEnabled || back.HelmListen != "127.0.0.1:8500" {
		t.Fatalf("roundtrip: enabled=%v listen=%q", back.HelmEnabled, back.HelmListen)
	}

	rewrite := func(mutate func(map[string]any)) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		mutate(cfg)
		out, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "config.json"), out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	planes := func(cfg map[string]any) map[string]any {
		return cfg["planes"].(map[string]any)
	}

	// A state dir founded before the helm existed must not sprout one.
	rewrite(func(cfg map[string]any) { delete(planes(cfg), "helm") })
	old, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if old.HelmEnabled {
		t.Fatal("absent planes.helm block must mean disabled")
	}

	// A listener collision is refused by name.
	rewrite(func(cfg map[string]any) {
		planes(cfg)["helm"] = map[string]any{"enabled": true, "listen": "127.0.0.1:8378"}
	})
	if _, err := ceremony.Verify(dir); err == nil ||
		!strings.Contains(err.Error(), "planes.helm.listen") {
		t.Fatalf("fold/helm collision not refused: %v", err)
	}

	// The helm without any sign-in issuer is refused by name.
	rewrite(func(cfg map[string]any) {
		planes(cfg)["helm"] = map[string]any{"enabled": true, "listen": "127.0.0.1:8500"}
		planes(cfg)["fold"] = map[string]any{"enabled": false}
	})
	if _, err := ceremony.Verify(dir); err == nil ||
		!strings.Contains(err.Error(), "sign-in issuer") {
		t.Fatalf("issuer-less helm not refused: %v", err)
	}
}

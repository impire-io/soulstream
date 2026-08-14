package ceremony_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

// TestHelmWiring covers the shell plane's config contract: on by default
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

	// A state dir founded before the shell existed must not sprout one.
	rewrite(func(cfg map[string]any) { delete(planes(cfg), "shell") })
	old, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if old.HelmEnabled {
		t.Fatal("absent planes.shell block must mean disabled")
	}

	// A listener collision is refused by name.
	rewrite(func(cfg map[string]any) {
		planes(cfg)["shell"] = map[string]any{"enabled": true, "listen": "127.0.0.1:8378"}
	})
	if _, err := ceremony.Verify(dir); err == nil ||
		!strings.Contains(err.Error(), "planes.shell.listen") {
		t.Fatalf("fold/shell collision not refused: %v", err)
	}

	// The shell without any sign-in issuer is refused by name.
	rewrite(func(cfg map[string]any) {
		planes(cfg)["shell"] = map[string]any{"enabled": true, "listen": "127.0.0.1:8500"}
		planes(cfg)["fold"] = map[string]any{"enabled": false}
	})
	if _, err := ceremony.Verify(dir); err == nil ||
		!strings.Contains(err.Error(), "sign-in issuer") {
		t.Fatalf("issuer-less shell not refused: %v", err)
	}
}

// TestAdminSurface covers the other fact the shell plane is handed: where
// this deployment administers the people who can sign in. It is declared,
// never probed — a consumer reads a plane's absence as a fact before
// anything is running.
func TestAdminSurface(t *testing.T) {
	// The bundled shape: the fold runs, and it is what people sign in
	// against, so it is what this node administers.
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	if got := st.AdminSurface(); got != st.FoldIssuer {
		t.Fatalf("bundled: admin surface = %q, want the fold's own %q", got, st.FoldIssuer)
	}

	// The external-IdP shape: the plane is off and sessions ride an AS
	// this node does not run. There is nobody here to administer.
	ext := *st
	ext.FoldEnabled = false
	ext.DoorPublicURL = "http://node.example:8080"
	ext.DoorAuthIssuer = "http://as.example"
	ext.DoorAuthAudience = "soulstream-home"
	if got := ext.AdminSurface(); got != "" {
		t.Fatalf("external AS: admin surface = %q, want none", got)
	}

	// The half-way shape: the plane still runs, but people sign in
	// somewhere else. The fold's own records are not the people signing
	// in, so this node has no standing to administer them either.
	both := *st
	both.DoorPublicURL = "http://node.example:8080"
	both.DoorAuthIssuer = "http://as.example"
	both.DoorAuthAudience = "soulstream-home"
	if got := both.AdminSurface(); got != "" {
		t.Fatalf("fold beside an external AS: admin surface = %q, want none", got)
	}

	// An ephemeral fold listener resolves to no issuer at all, and an
	// unreachable issuer is not a surface to administer through.
	eph := *st
	eph.FoldIssuer = "http://localhost:0"
	if got := eph.AdminSurface(); got != "" {
		t.Fatalf("ephemeral fold: admin surface = %q, want none", got)
	}
}

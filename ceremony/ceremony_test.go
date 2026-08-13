package ceremony

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func generate(t *testing.T) *State {
	t.Helper()
	s, err := Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return s
}

func TestRoundtripAndVerify(t *testing.T) {
	dir := t.TempDir()
	s := generate(t)
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Verify(dir)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if loaded.RealmPub != s.RealmPub || loaded.AuthPub != s.AuthPub ||
		loaded.OperatorPub != s.OperatorPub || loaded.Listen != s.Listen {
		t.Fatal("loaded state does not match generated state")
	}
	if loaded.Realm != "home" || !loaded.MemoryEnabled {
		t.Fatalf("config roundtrip: realm %q, memory %v", loaded.Realm, loaded.MemoryEnabled)
	}
	if len(loaded.ArchivistCreds) == 0 {
		t.Fatal("archivist creds missing from the loaded inventory")
	}
	// Verify again — idempotent.
	if _, err := Verify(dir); err != nil {
		t.Fatalf("second verify: %v", err)
	}

	// The disabled-plane block survives the roundtrip too.
	s2 := generate(t)
	s2.MemoryEnabled = false
	dir2 := t.TempDir()
	if err := s2.Save(dir2); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded2, err := Verify(dir2)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if loaded2.MemoryEnabled {
		t.Fatal("memory plane should load disabled")
	}
}

func TestModesAreOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	s := generate(t)
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := WriteSentinel(dir, "-----BEGIN NATS USER JWT-----\nfake\n"); err != nil {
		t.Fatalf("sentinel: %v", err)
	}
	err := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s has group/other bits: %o", path, info.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestFoundedAndEmpty(t *testing.T) {
	dir := t.TempDir()
	if empty, _ := Empty(dir); !empty {
		t.Fatal("fresh temp dir should be empty")
	}
	if empty, _ := Empty(filepath.Join(dir, "absent")); !empty {
		t.Fatal("absent dir should count as empty")
	}
	s := generate(t)
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	if empty, _ := Empty(dir); empty {
		t.Fatal("saved dir should not be empty")
	}
	if Founded(dir) {
		t.Fatal("no sentinel yet — must not read as founded (the incomplete state)")
	}
	if err := WriteSentinel(dir, "creds"); err != nil {
		t.Fatalf("sentinel: %v", err)
	}
	if !Founded(dir) {
		t.Fatal("sentinel written — must read as founded")
	}
}

// TestDamageMatrix: every damage class is refused with the artifact named.
func TestDamageMatrix(t *testing.T) {
	cases := []struct {
		name   string
		damage func(t *testing.T, dir string)
		want   string
	}{
		{"missing-jwt", func(t *testing.T, dir string) {
			t.Helper()
			if err := os.Remove(filepath.Join(dir, "keys/auth.jwt")); err != nil {
				t.Fatal(err)
			}
		}, "keys/auth.jwt"},
		{"corrupt-seed", func(t *testing.T, dir string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(dir, "keys/operator.nk"), []byte("not-a-seed"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "keys/operator.nk"},
		{"jwt-seed-mismatch", func(t *testing.T, dir string) {
			t.Helper()
			other := generateForTest(t)
			if err := os.WriteFile(filepath.Join(dir, "keys/sys.jwt"), []byte(other.SysJWT), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "keys/sys.jwt"},
		{"non-loopback-listen", func(t *testing.T, dir string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"listen":"0.0.0.0:4222"}`+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "not loopback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			s := generate(t)
			if err := s.Save(dir); err != nil {
				t.Fatalf("save: %v", err)
			}
			tc.damage(t, dir)
			_, err := Verify(dir)
			if err == nil {
				t.Fatal("damaged state verified clean")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not name %q", err, tc.want)
			}
		})
	}
}

func generateForTest(t *testing.T) *State {
	t.Helper()
	return generate(t)
}

// TestFoldWiring covers the fold plane's defaults and guards (D25 /
// soulstream fold URLs): the bundled fold is on by default with a
// localhost issuer (WebAuthn refuses a bare IP), and the two footguns
// refuse at load — a bare-IP issuer and a fold/door listener collision.
func TestFoldWiring(t *testing.T) {
	// Default: fold enabled, localhost issuer, distinct port from the door.
	s := generate(t)
	if !s.FoldEnabled || s.FoldIssuer != "http://localhost:8378" {
		t.Fatalf("fold default: enabled=%v issuer=%q", s.FoldEnabled, s.FoldIssuer)
	}
	dir := t.TempDir()
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := Verify(dir)
	if err != nil {
		t.Fatalf("verify default fold config: %v", err)
	}
	if loaded.FoldIssuer != "http://localhost:8378" || loaded.FoldAudience != "soulstream-home" {
		t.Fatalf("fold roundtrip: issuer=%q audience=%q", loaded.FoldIssuer, loaded.FoldAudience)
	}

	// A bare-IP issuer refuses (WebAuthn RP-id rule).
	badIP := generate(t)
	badIP.FoldIssuer = "http://127.0.0.1:8378"
	d2 := t.TempDir()
	if err := badIP.Save(d2); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(d2); err == nil {
		t.Fatal("a bare-IP fold issuer was accepted")
	}

	// A fold/door listener collision refuses.
	clash := generate(t)
	clash.FoldListen = "127.0.0.1:8080" // == the door default
	d3 := t.TempDir()
	if err := clash.Save(d3); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(d3); err == nil {
		t.Fatal("a fold/door listener collision was accepted")
	}
}

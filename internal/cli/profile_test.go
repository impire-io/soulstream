package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/internal/keystore"
)

func TestProfilePublishAndShow(t *testing.T) {
	connect := testConnector(t)
	keyFile := filepath.Join(t.TempDir(), "acme-daan.ed25519")
	pinsFile := filepath.Join(t.TempDir(), "acme.json")
	base := []string{"--realm", "acme", "--persona", "daan", "--key-file", keyFile, "--pins-file", pinsFile}

	run(connect, append(base, "provision")...)
	if code, _, errs := run(connect, append(base, "key", "init")...); code != 0 {
		t.Fatalf("key init: %s", errs)
	}
	key, err := keystore.LoadKey(keyFile)
	if err != nil || key == nil {
		t.Fatalf("load key: %v", err)
	}

	code, out, errs := run(connect, append(base, "profile", "publish", "--display-name", "Daan")...)
	if code != 0 {
		t.Fatalf("profile publish exit %d: %s", code, errs)
	}
	if !strings.Contains(out, key.PublicKey()) {
		t.Errorf("publish output missing the public key: %q", out)
	}

	code, out, errs = run(connect, append(base, "profile", "show", "daan")...)
	if code != 0 {
		t.Fatalf("profile show exit %d: %s", code, errs)
	}
	for _, want := range []string{"name:         daan", "display name: Daan", key.PublicKey(), "(current)"} {
		if !strings.Contains(out, want) {
			t.Errorf("profile show missing %q:\n%s", want, out)
		}
	}

	// Metadata update: same persona republished with new metadata keeps the key.
	code, _, errs = run(connect, append(base, "profile", "publish", "--display-name", "Daan G")...)
	if code != 0 {
		t.Fatalf("metadata update exit %d: %s", code, errs)
	}
	_, out, _ = run(connect, append(base, "profile", "show", "daan")...)
	if !strings.Contains(out, "Daan G") || !strings.Contains(out, key.PublicKey()) {
		t.Errorf("metadata update lost data:\n%s", out)
	}

	// Unknown persona: helpful failure.
	if code, _, _ = run(connect, append(base, "profile", "show", "stranger")...); code == 0 {
		t.Error("profile show for an absent persona should fail")
	}
}

func TestProfileProvisionListsPersonasArtefact(t *testing.T) {
	connect := testConnector(t)
	code, out, errs := run(connect, "--realm", "acme", "--persona", "daan", "provision")
	if code != 0 {
		t.Fatalf("provision exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "personas") {
		t.Errorf("provision output missing the personas artefact: %q", out)
	}
}

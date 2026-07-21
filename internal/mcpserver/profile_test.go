package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/registry"
)

func TestPublishProfileWithSessionKey(t *testing.T) {
	url, key := signedSetupURL(t)
	c := signedClientOn(t, url, "bookkeeper-agent", key)
	h := newHandlers(c)
	ctx := context.Background()

	res, _, err := h.publishProfile(ctx, nil, publishProfileInput{
		DisplayName: "The Bookkeeper",
		OperatedBy:  "daan",
	})
	if err != nil {
		t.Fatalf("publishProfile: %v", err)
	}
	out := resultText(t, res)
	for _, want := range []string{`"name": "bookkeeper-agent"`, `"kind": "agent"`, key.PublicKey(), `"operated_by": "daan"`} {
		if !strings.Contains(out, want) {
			t.Errorf("result missing %s:\n%s", want, out)
		}
	}

	// Metadata update keeps the stored key.
	if _, _, err := h.publishProfile(ctx, nil, publishProfileInput{DisplayName: "Bookkeeper v2"}); err != nil {
		t.Fatalf("metadata update: %v", err)
	}
	p, ok, err := registry.Lookup(ctx, c, "bookkeeper-agent")
	if err != nil || !ok {
		t.Fatalf("Lookup: ok=%v err=%v", ok, err)
	}
	if p.DisplayName != "Bookkeeper v2" || p.SigningKey == nil || p.SigningKey.Ed25519 != key.PublicKey() {
		t.Errorf("update lost data: %+v", p)
	}
}

// signedSetupURL starts a provisioned realm and returns its URL plus a fresh key.
func signedSetupURL(t *testing.T) (string, *identity.SigningKey) {
	t.Helper()
	h, url := setup(t, "provisioner")
	_ = h
	key, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	return url, key
}

// signedClientOn mirrors clientOn with a signer attached.
func signedClientOn(t *testing.T, url, persona string, key *identity.SigningKey) *realm.Client {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	c, err := realm.NewClient(context.Background(), nc, realm.Config{Realm: "acme", Persona: persona, Signer: key})
	if err != nil {
		nc.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

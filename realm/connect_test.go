package realm

import (
	"context"
	"testing"

	"github.com/impire/soulstream/identity"
)

// Connect must reject malformed names before it makes any server contact.
func TestConnectRejectsInvalidNamesBeforeContact(t *testing.T) {
	ctx := context.Background()

	if _, err := Connect(ctx, Config{ContextName: "irrelevant", Realm: "Bad Realm"}); err == nil {
		t.Error("Connect with invalid realm name: got nil, want error")
	}
	if _, err := Connect(ctx, Config{ContextName: "irrelevant", Realm: "acme", Persona: "Bad.Persona"}); err == nil {
		t.Error("Connect with invalid persona name: got nil, want error")
	}
}

// The client carries its optional signer verbatim: set → returned, unset → nil
// (nil is the publishes-unsigned mode every pre-signing caller relies on).
func TestClientCarriesSigner(t *testing.T) {
	key, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signed := &Client{cfg: Config{Realm: "acme", Persona: "daan", Signer: key}}
	if signed.Signer() != key {
		t.Error("Signer() did not return the configured key")
	}
	unsigned := &Client{cfg: Config{Realm: "acme", Persona: "daan"}}
	if unsigned.Signer() != nil {
		t.Error("Signer() on a key-less client: want nil")
	}
}

// Conn exposes the raw connection for request-reply surfaces, same as JetStream()
// exposes the stream-side handle.
func TestClientExposesConn(t *testing.T) {
	c := &Client{cfg: Config{Realm: "acme"}}
	if c.Conn() != c.nc {
		t.Error("Conn() did not return the underlying connection")
	}
}

// Connect must error (never panic, never partially mutate) when the named context
// does not exist / the server cannot be reached.
func TestConnectMissingContextErrors(t *testing.T) {
	ctx := context.Background()

	_, err := Connect(ctx, Config{
		ContextName: "soulstream-nonexistent-context-zzq",
		Realm:       "acme",
	})
	if err == nil {
		t.Fatal("Connect with nonexistent context: got nil, want error")
	}
}

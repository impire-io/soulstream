package topic

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream/identity"
	"github.com/impire-io/soulstream/internal/natstest"
	"github.com/impire-io/soulstream/realm"
)

// testServer starts an in-process JetStream server and returns its client URL.
func testServer(t *testing.T) string {
	t.Helper()
	url, shutdown := natstest.StartJetStream(t)
	t.Cleanup(shutdown)
	return url
}

// connectClient connects a realm client (as persona) to url.
func connectClient(t *testing.T, url, persona string) *realm.Client {
	t.Helper()
	return connectClientSigned(t, url, persona, nil)
}

// connectClientSigned connects a realm client (as persona) that signs with key.
func connectClientSigned(t *testing.T, url, persona string, key *identity.SigningKey) *realm.Client {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	c, err := realm.NewClient(context.Background(), nc, realm.Config{Realm: "test-realm", Persona: persona, Signer: key})
	if err != nil {
		nc.Close()
		t.Fatalf("new client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// provisionedClient starts a server, connects as persona, and provisions the realm.
func provisionedClient(t *testing.T, persona string) *realm.Client {
	t.Helper()
	c := connectClient(t, testServer(t), persona)
	if _, err := c.Provision(context.Background()); err != nil {
		t.Fatalf("provision: %v", err)
	}
	return c
}

package realm

import (
	"context"
	"testing"
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

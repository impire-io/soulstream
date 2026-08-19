package node

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-identity/client"

	"github.com/impire-io/soulstream/ceremony"
)

// Found performs the founding administrative acts on a freshly started
// node, through the public client over the node's own connection — the
// same acts any operator performs (design 0001 §4, steps 7–8): both
// signing keys into the vault, the first access token, and the sentinel,
// written LAST as the founding-complete marker. It returns the token's
// plaintext — the caller prints it once and forgets it.
func Found(n *Node, st *ceremony.State, stateDir string) (string, error) {
	// The record substrate is founded with everything else (design §4
	// step 7): create-or-report, so re-running is always safe.
	js, err := jetstream.New(n.ncOps)
	if err != nil {
		return "", fmt.Errorf("node: jetstream: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := realm.ProvisionOn(realm.WithConn(ctx, n.ncOps), js); err != nil {
		return "", fmt.Errorf("node: provision realm substrate: %w", err)
	}

	ops := n.Ops()
	if _, err := ops.ImportKey("realm", client.KindNATSAccountSigningKey,
		string(st.RealmSigningSeed), st.RealmPub, ""); err != nil {
		return "", fmt.Errorf("node: import realm signing key: %w", err)
	}
	if _, err := ops.ImportKey("auth/issuer", client.KindNATSAccountSigningKey,
		string(st.AuthSigningSeed), st.AuthPub, ""); err != nil {
		return "", fmt.Errorf("node: import auth signing key: %w", err)
	}
	created, err := ops.CreateToken(st.RealmPub, ceremony.FoundingPersona, "founding token", 0)
	if err != nil {
		return "", fmt.Errorf("node: founding token: %w", err)
	}
	sentinel, err := ops.MintSentinel()
	if err != nil {
		return "", fmt.Errorf("node: sentinel: %w", err)
	}
	if err := ceremony.WriteSentinel(stateDir, sentinel.Creds); err != nil {
		return "", err
	}
	return created.Token, nil
}

// newAuditLogger builds the identity plane's logger: slog text at debug —
// the level the audit lines (`callout REFUSED`/`ADMITTED`) ride on.
func newAuditLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

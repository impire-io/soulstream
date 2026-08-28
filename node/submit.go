package node

import (
	"context"
	"fmt"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/topic"
	siclient "github.com/impire-io/soulstream-identity/client"
	"github.com/impire-io/soulstream-workloads/declaration"
	"github.com/impire-io/soulstream-workloads/dispatcher"
	"github.com/impire-io/soulstream-workloads/fleet"

	"github.com/impire-io/soulstream/ceremony"
)

// SubmitAgent places a declared agent on the deployment and returns the
// placement's id. Submission is submit-and-forget: an ordinary work item
// carrying the declaration, published on the operator's own connection —
// after which this process may exit, and a dispatcher node races for it,
// serves it, and answers on the agent's behalf for as long as the
// deployment runs.
//
// It refuses before publishing anything a dispatcher could not serve: a
// declaration the wake engine cannot run (there is nothing to wake it),
// and a realm whose config declares no dispatcher plane — a placement
// nobody serves is a work item that sits open forever, and the operator
// should hear that now rather than wonder later.
func SubmitAgent(ctx context.Context, cfg Config, url, declPath string) (string, error) {
	st := cfg.State
	if !st.DispatcherEnabled {
		return "", fmt.Errorf("node: this deployment runs no dispatcher plane — nothing would serve the placement; enable planes.dispatcher in config.json (or run the agent here with `soulstream wrap`)")
	}
	data, err := os.ReadFile(declPath)
	if err != nil {
		return "", fmt.Errorf("node: read declaration: %w", err)
	}
	d, err := declaration.Parse(data)
	if err != nil {
		return "", fmt.Errorf("node: %w", err)
	}
	if err := d.Validate(); err != nil {
		return "", fmt.Errorf("node: %w", err)
	}
	if !dispatcher.Servable(d) {
		return "", fmt.Errorf("node: %s declares nothing that wakes it — the dispatcher serves declared agents through the wake engine, so a declaration with no wake entries belongs to `soulstream workload start`", declPath)
	}

	nc, err := nats.Connect(url,
		nats.UserCredentials(ceremony.UserCredsPath(cfg.StateDir, "ops")),
		nats.Name("soulstream-submit"),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		return "", fmt.Errorf("node: cannot reach the realm at %s — is `soulstream up` running? (%w)", url, err)
	}
	signer, err := siclient.New(nc, st.RealmPub, "ops").PersonaSigner("ops")
	if err != nil {
		nc.Close()
		return "", fmt.Errorf("node: submitter signer: %w", err)
	}
	c, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: st.Realm, Persona: "ops", Signer: signer,
	})
	if err != nil {
		nc.Close()
		return "", fmt.Errorf("node: submitter realm client: %w", err)
	}
	defer func() { _ = c.Close() }()

	path, err := ensurePlacements(ctx, c, st.DispatcherPlacements)
	if err != nil {
		return "", err
	}
	id, err := fleet.Submit(ctx, topic.Open(c, path), d)
	if err != nil {
		return "", fmt.Errorf("node: submit: %w", err)
	}
	return id, nil
}

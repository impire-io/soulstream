package node

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulrealm/backend/native"
	"github.com/impire-io/soulrealm/declaration"
	"github.com/impire-io/soulrealm/minter"
	"github.com/impire-io/soulrealm/runner"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/topic"

	"github.com/impire-io/soulnode/ceremony"
)

// workloadCredTTL bounds a workload's minted credential — long enough for
// any sane run, short enough that a leaked credential dies on its own.
const workloadCredTTL = time.Hour

// RunWorkload is the runtime plane, invocation-scoped (research R2): it
// parses and validates one declaration, connects as the runner persona to
// a *running* node at url, mints the workload's scoped credential with
// the ceremony's plain workload signing key, launches through the native
// backend with scratch under the state directory, and supervises to the
// terminal work op (or until ctx ends → an intentional stop). Everything
// it composes is upstream public surface.
func RunWorkload(ctx context.Context, cfg Config, url, declPath string) error {
	st := cfg.State
	data, err := os.ReadFile(declPath)
	if err != nil {
		return fmt.Errorf("node: read declaration: %w", err)
	}
	d, err := declaration.Parse(data)
	if err != nil {
		return fmt.Errorf("node: %w", err)
	}
	if err := d.Validate(); err != nil {
		return fmt.Errorf("node: %w", err)
	}
	artifact, err := d.ArtifactPath()
	if err != nil {
		return fmt.Errorf("node: %w", err)
	}
	if _, err := os.Stat(artifact); err != nil {
		return fmt.Errorf("node: declaration artifact %s: %w", artifact, err)
	}

	nc, err := nats.Connect(url,
		nats.UserCredentials(ceremony.UserCredsPath(cfg.StateDir, "runner")),
		nats.Name("soulnode-runner"),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0),
		nats.Timeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("node: cannot reach the node at %s — is `soulnode up` running? (%w)", url, err)
	}
	// The runner is a persona like everyone else: lifecycle ops are
	// attributed to it, its signing key vault-held (research R3).
	signer, err := client.New(nc, st.RealmPub, "runner").PersonaSigner("runner")
	if err != nil {
		nc.Close()
		return fmt.Errorf("node: runner signer: %w", err)
	}
	rc, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: st.Realm, Persona: "runner", Signer: signer,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("node: runner realm client: %w", err)
	}
	defer func() { _ = rc.Close() }()

	m, err := minter.NewSigningKeyMinter(st.WorkloadSigningSeed, st.RealmPub, []string{url})
	if err != nil {
		return fmt.Errorf("node: workload minter: %w", err)
	}
	r := &runner.Runner{
		Minter:      m,
		Backend:     native.New(),
		Realm:       st.Realm,
		CredTTL:     workloadCredTTL,
		ScratchRoot: filepath.Join(cfg.StateDir, "scratch"),
	}
	rw, err := r.Launch(ctx, topic.Open(rc, d.Topic), d)
	if err != nil {
		return fmt.Errorf("node: launch: %w", err)
	}
	return rw.Serve(ctx)
}

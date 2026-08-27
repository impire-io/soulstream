package node

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/registry"
	"github.com/impire-io/soulstream-core/topic"
	"github.com/impire-io/soulstream-identity/client"
	"github.com/impire-io/soulstream-workloads/backend/native"
	"github.com/impire-io/soulstream-workloads/declaration"
	"github.com/impire-io/soulstream-workloads/minter"
	"github.com/impire-io/soulstream-workloads/runner"

	"github.com/impire-io/soulstream/ceremony"
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
	// Capability preflight (specs/013): both refusals are named, and both
	// fire before any connection or op — a refused declaration publishes
	// nothing.
	if d.Capabilities != nil {
		if len(st.AgentSigningSeed) == 0 {
			return fmt.Errorf("node: this realm has no agent capability key (%s) — it was founded before capability-minting, or on a substrate whose account half does not carry the agent scope yet; declarations with capabilities need it. Re-init a fresh realm (pre-v1 clean break) or drop the capabilities block", ceremony.AgentSigningFile)
		}
		if d.Capabilities.Role != ceremony.AgentRole {
			return fmt.Errorf("node: capability role %q is not declared in this realm — the founding declares one capability role, %q", d.Capabilities.Role, ceremony.AgentRole)
		}
	}

	nc, err := nats.Connect(url,
		nats.UserCredentials(ceremony.UserCredsPath(cfg.StateDir, "runner")),
		nats.Name("soulstream-runner"),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0),
		nats.Timeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("node: cannot reach the node at %s — is `soulstream up` running? (%w)", url, err)
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
	// F1 (core v0.9.0): make the runner's signing key reader-resolvable at
	// signer construction. Loud on failure, never fatal to the launch.
	if err := registry.EnsureSigningKey(ctx, rc, signer); err != nil {
		fmt.Fprintf(os.Stderr, "soulstream: WARNING: runner signing key not published to the directory (records will read unknown-key): %v\n", err)
	}

	plain, err := minter.NewSigningKeyMinter(st.WorkloadSigningSeed, st.RealmPub, []string{url})
	if err != nil {
		return fmt.Errorf("node: workload minter: %w", err)
	}
	var m minter.Minter = plain
	if len(st.AgentSigningSeed) > 0 {
		scoped, err := minter.NewScopedSigningKeyMinter(st.AgentSigningSeed, st.RealmPub, []string{url})
		if err != nil {
			return fmt.Errorf("node: agent capability minter: %w", err)
		}
		m = &capabilityMinter{scoped: scoped, plain: plain}
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

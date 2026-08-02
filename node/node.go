// Package node composes a SoulNode: the embedded operator-mode NATS
// server (loopback listener, JetStream on the state directory) and the
// identity plane (soulidentity's public embed surface), each plane on an
// ordinary NATS connection — never an in-process transport (constitution
// III). ceremony generates what this package boots; cmd/soulnode owns
// flags and signals.
package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulidentity/embed"
	"github.com/impire-io/soulstream-archivist/archive"
	"github.com/impire-io/soulstream-archivist/keeper"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/topic"

	"github.com/impire-io/soulnode/ceremony"
)

// Config describes a composition to start. State is a verified ceremony;
// StateDir is its home (creds files, JetStream store).
type Config struct {
	StateDir string
	State    *ceremony.State

	// AuditWriter receives the identity plane's audit/serving log
	// (slog text, includes the `callout REFUSED` lines). Default:
	// os.Stderr — the daemon convention.
	AuditWriter io.Writer
}

// Node is a running composition.
type Node struct {
	srv    *natsserver.Server
	cancel context.CancelFunc
	planes chan error // the identity plane's Run result

	ncService *nats.Conn
	ncIssuer  *nats.Conn
	ncOps     *nats.Conn

	// The memory plane (nil when disabled): the realm client owns the
	// archivist connection; the two loops report into memErr.
	realmClient *realm.Client
	memErr      chan error
	audit       *slog.Logger

	ops *client.Client
	url string
}

// URL is the client URL of the embedded server's loopback listener.
func (n *Node) URL() string { return n.url }

// Ops is the founding/administrative client (the ops bypass-lane user).
func (n *Node) Ops() *client.Client { return n.ops }

// Start boots the composition: pre-flight the listener, start the server,
// connect the planes' ordinary loopback connections, run the identity
// plane, and wait until the sealed surface answers. Failures name their
// stage and leave nothing running.
func Start(cfg Config) (*Node, error) {
	st := cfg.State
	if st == nil {
		return nil, errors.New("node: Config.State is required (a verified ceremony)")
	}
	audit := cfg.AuditWriter
	if audit == nil {
		audit = os.Stderr
	}

	// Pre-flight the bind so a conflict is a named refusal, not a
	// timeout (contracts/cli.md).
	probe, err := net.Listen("tcp", st.Listen)
	if err != nil {
		return nil, fmt.Errorf("node: cannot listen on %s (change \"listen\" in %s): %w",
			st.Listen, filepath.Join(cfg.StateDir, "config.json"), err)
	}
	_ = probe.Close()

	host, portStr, err := net.SplitHostPort(st.Listen)
	if err != nil {
		return nil, fmt.Errorf("node: listen %q: %w", st.Listen, err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return nil, fmt.Errorf("node: listen port %q: %w", portStr, err)
	}

	res := &natsserver.MemAccResolver{}
	for pub, token := range map[string]string{
		st.SysPub: st.SysJWT, st.AuthPub: st.AuthJWT, st.RealmPub: st.RealmJWT,
	} {
		if err := res.Store(pub, token); err != nil {
			return nil, fmt.Errorf("node: account resolver: %w", err)
		}
	}
	srv, err := natsserver.NewServer(&natsserver.Options{
		Host:            host,
		Port:            port,
		JetStream:       true,
		StoreDir:        filepath.Join(cfg.StateDir, "jetstream"),
		TrustedKeys:     []string{st.OperatorPub},
		SystemAccount:   st.SysPub,
		AccountResolver: res,
		NoLog:           true,
		NoSigs:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("node: server: %w", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		srv.Shutdown()
		return nil, fmt.Errorf("node: server did not become ready on %s", st.Listen)
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{srv: srv, cancel: cancel, planes: make(chan error, 1), url: srv.ClientURL()}
	fail := func(err error) (*Node, error) {
		n.Stop()
		return nil, err
	}

	connect := func(name string) (*nats.Conn, error) {
		nc, err := nats.Connect(n.url,
			nats.UserCredentials(ceremony.UserCredsPath(cfg.StateDir, name)),
			nats.Name("soulnode-"+name))
		if err != nil {
			return nil, fmt.Errorf("node: %s connection: %w", name, err)
		}
		return nc, nil
	}
	if n.ncService, err = connect("service"); err != nil {
		return fail(err)
	}
	if n.ncIssuer, err = connect("issuer"); err != nil {
		return fail(err)
	}
	if n.ncOps, err = connect("ops"); err != nil {
		return fail(err)
	}

	logger := newAuditLogger(audit)
	go func() {
		n.planes <- embed.Run(ctx, embed.Options{
			Conn:        n.ncService,
			CalloutConn: n.ncIssuer,
			FirstKey:    string(st.VaultFirstSeed),
			SurfaceKey:  string(st.SurfaceSeed),
			CalloutKey:  string(st.CalloutSeed),
			AuthAccount: st.AuthPub,
			Logger:      logger,
		})
	}()

	// Ready means the sealed surface answers. A vault-key mismatch or a
	// plane startup failure surfaces here, named by the plane.
	n.ops = client.New(n.ncOps, st.RealmPub, "ops")
	n.audit = logger
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := n.ops.Status(); err == nil {
			break
		}
		select {
		case err := <-n.planes:
			return fail(fmt.Errorf("node: identity plane failed to start: %w", err))
		case <-time.After(100 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return fail(errors.New("node: identity plane did not become ready"))
		}
	}

	if st.MemoryEnabled {
		if err := n.startMemory(ctx, cfg); err != nil {
			return fail(err)
		}
	}
	return n, nil
}

// startMemory wires the memory plane (design §6): the substrate guard,
// the archivist's realm client with its vault-held persona signer, the
// keeper, and the memory witness. Startup failures are named; runtime
// exits land in memErr and are reported loud at Stop.
func (n *Node) startMemory(ctx context.Context, cfg Config) error {
	st := cfg.State

	// Create-or-verify the record substrate before the keeper needs it
	// (research R2 — makes "is the realm provisioned?" unreachable).
	js, err := jetstream.New(n.ncOps)
	if err != nil {
		return fmt.Errorf("node: memory plane: jetstream: %w", err)
	}
	provCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := realm.ProvisionOn(provCtx, js); err != nil {
		return fmt.Errorf("node: memory plane: realm substrate: %w", err)
	}

	ncArch, err := nats.Connect(n.url,
		nats.UserCredentials(ceremony.UserCredsPath(cfg.StateDir, "archivist")),
		nats.Name("soulnode-archivist"))
	if err != nil {
		return fmt.Errorf("node: memory plane: archivist connection: %w", err)
	}
	// The persona's signing key is vault-held, materialized on first
	// touch — nothing persona-shaped on disk (research R3).
	signer, err := client.New(ncArch, st.RealmPub, "archivist").PersonaSigner("archivist")
	if err != nil {
		ncArch.Close()
		return fmt.Errorf("node: memory plane: persona signer: %w", err)
	}
	rc, err := realm.NewClient(ctx, ncArch, realm.Config{
		Realm: st.Realm, Persona: "archivist", Signer: signer,
	})
	if err != nil {
		ncArch.Close()
		return fmt.Errorf("node: memory plane: realm client: %w", err)
	}
	n.realmClient = rc // owns ncArch from here

	store, err := archive.Open(filepath.Join(cfg.StateDir, "archive"))
	if err != nil {
		return fmt.Errorf("node: memory plane: archive: %w", err)
	}
	kept, err := store.LoadState()
	if err != nil {
		return fmt.Errorf("node: memory plane: archive state: %w", err)
	}
	w := keeper.Witness(store, kept.CoverageFrom)
	w.OnServed = func(kind string, served int, err error) {
		if err != nil {
			n.audit.Warn("memory serve failed", "kind", kind, "err", err)
			return
		}
		n.audit.Info("memory served", "kind", kind, "n", served)
	}

	n.memErr = make(chan error, 2)
	go func() { n.memErr <- keeper.Run(ctx, rc, store, nil) }()
	go func() { n.memErr <- topic.RespondMemory(ctx, rc, w) }()
	return nil
}

// Stop drains the planes, closes the node's connections, and shuts the
// server down. The state directory remains valid for the next Start.
func (n *Node) Stop() {
	if n.cancel != nil {
		n.cancel()
		select {
		case <-n.planes: // embed.Run drained and returned
		case <-time.After(10 * time.Second):
		}
	}
	if n.memErr != nil {
		// Both memory loops end with the ctx; anything that is not the
		// ctx ending is reported loud (constitution III: never silent).
		for range 2 {
			select {
			case err := <-n.memErr:
				if err != nil && !errors.Is(err, context.Canceled) {
					n.audit.Error("memory plane exited", "err", err)
				}
			case <-time.After(10 * time.Second):
			}
		}
	}
	if n.realmClient != nil {
		_ = n.realmClient.Close()
	}
	for _, nc := range []*nats.Conn{n.ncOps, n.ncIssuer, n.ncService} {
		if nc != nil {
			nc.Close()
		}
	}
	n.srv.Shutdown()
}

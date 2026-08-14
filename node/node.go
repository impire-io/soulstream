// Package node composes a SoulNode: the embedded operator-mode NATS
// server (loopback listener, JetStream on the state directory) and the
// identity plane (soulstream-identity's public embed surface), each plane on an
// ordinary NATS connection — never an in-process transport (constitution
// III). ceremony generates what this package boots; cmd/soulstream owns
// flags and signals.
package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream-archivist/archive"
	"github.com/impire-io/soulstream-archivist/keeper"
	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/topic"
	"github.com/impire-io/soulstream-identity/client"
	"github.com/impire-io/soulstream-identity/embed"
	foldembed "github.com/impire-io/soulstream-idp/embed"
	door "github.com/impire-io/soulstream-mcp"
	helmembed "github.com/impire-io/soulstream-shell/embed"

	"github.com/impire-io/soulstream/ceremony"
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

	// The door plane (nil when disabled): upstream's remote MCP node on
	// a loopback HTTP listener SoulNode holds.
	doorNode *door.Node
	doorSrv  *http.Server
	doorErr  chan error
	doorURL  string

	// The fold plane (nil when disabled): the bundled OIDC provider
	// through soulstream-idp's public embed seam, storing on this node's
	// JetStream under its own bucket prefix. foldInvite holds the
	// owner's founding enrollment invite when this start minted one
	// (soulstream-idp M3: enrollment requires an invite; the fold's seam
	// delivers it here and the founding output prints it once).
	foldErr    chan error
	foldURL    string
	foldInvite string

	// The shell plane (soulstream-shell — the human cockpit) through soulstream-shell's
	// public embed seam: observe and act, sessions signing in against
	// the deployment's AS through the identity plane's OIDC lane.
	helmErr chan error
	helmURL string

	ops *client.Client
	url string
}

// DoorURL is the door plane's HTTP endpoint ("" when disabled).
func (n *Node) DoorURL() string { return n.doorURL }

// FoldURL is the bundled fold's issuer URL ("" when disabled).
func (n *Node) FoldURL() string { return n.foldURL }

// FoldInvite is the owner's founding enrollment invite when this start
// minted one ("" otherwise). Single-use, digest-stored on the fold's
// side — print once, never persist.
func (n *Node) FoldInvite() string { return n.foldInvite }

// HelmURL is the shell plane's HTTP endpoint ("" when disabled).
func (n *Node) HelmURL() string { return n.helmURL }

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
	// Port 0 means "any free port" everywhere else in the config's
	// vocabulary (and in the pre-flight probe above); nats-server would
	// read 0 as its default 4222, silently disagreeing with the probe.
	// Its own random-port spelling is -1.
	if port == 0 {
		port = -1
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
			nats.Name("soulstream-"+name))
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
	n.audit = logger

	// The fold plane starts before the identity plane: the callout's
	// OIDC validator discovers its issuer at startup, and in the
	// bundled default that issuer IS the fold. The fold itself needs
	// only its bypass-lane connection — the server verifies it
	// natively, no callout in the path.
	if st.FoldEnabled {
		if err := n.startFold(ctx, cfg); err != nil {
			return fail(err)
		}
	}

	sessionIssuer, sessionAudience := st.SessionIssuer()
	go func() {
		n.planes <- embed.Run(ctx, embed.Options{
			Conn:        n.ncService,
			CalloutConn: n.ncIssuer,
			FirstKey:    string(st.VaultFirstSeed),
			SurfaceKey:  string(st.SurfaceSeed),
			CalloutKey:  string(st.CalloutSeed),
			AuthAccount: st.AuthPub,
			// The OIDC lane: on for public door mode (external AS) and
			// for the shell plane (the bundled fold by default) — browser
			// users hold tokens the callout validates against exactly
			// this issuer and audience.
			OIDCIssuer:   sessionIssuer,
			OIDCAudience: sessionAudience,
			Logger:       logger,
		})
	}()

	// Ready means the sealed surface answers. A vault-key mismatch or a
	// plane startup failure surfaces here, named by the plane.
	n.ops = client.New(n.ncOps, st.RealmPub, "ops")
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
	if st.DoorEnabled {
		if err := n.startDoor(cfg); err != nil {
			return fail(err)
		}
	}
	if st.HelmEnabled {
		if err := n.startHelm(ctx, cfg); err != nil {
			return fail(err)
		}
	}
	return n, nil
}

// startHelm runs the human cockpit (soulstream-shell's public embed seam) —
// composition, not invention: the shell reads through the node's ops
// lane, and every session opens its own admission through the identity
// plane's OIDC lane (which the ceremony wires whenever the shell is on).
// It starts last: it discovers the sign-in issuer at startup, so the
// fold (or external AS) must already answer.
func (n *Node) startHelm(ctx context.Context, cfg Config) error {
	st := cfg.State
	helmIssuer, _ := st.SessionIssuer()
	if helmIssuer == "" {
		// Loud, never silent: with no resolvable sign-in issuer (an
		// ephemeral fold listener) the surface cannot authenticate
		// anyone, so it does not serve.
		n.audit.Warn("shell plane skipped: no resolvable sign-in issuer")
		return nil
	}
	n.helmErr = make(chan error, 1)
	ready := make(chan string, 1)
	go func() {
		n.helmErr <- helmembed.Run(ctx, helmembed.Options{
			Listen:       st.HelmListen,
			NATSURL:      n.url,
			CredsPath:    ceremony.UserCredsPath(cfg.StateDir, "ops"),
			CredsUser:    "ops",
			SentinelPath: ceremony.SentinelPath(cfg.StateDir),
			Realm:        st.Realm,
			Account:      st.RealmPub,
			Issuer:       helmIssuer,
			// What this deployment declares about administering its own
			// people: the bundled fold's surface, or nothing at all when
			// the people signing in live on an AS this node does not run.
			// The shell's people-and-sign-in module reads exactly this to
			// know whether it is part of this build.
			AdminBase: st.AdminSurface(),
			// What this deployment declares about issuing agent
			// credentials: the address it tells an agent to dial, which
			// for a node serving its own machine is the address this node
			// answers on. Composed as this node's plane, the shell holds
			// the node's own standing, which is what minting a credential
			// in somebody else's name needs (design 0001 §4, class (b)) —
			// so this node runs that surface and says where to reach it.
			// A shell run beside a deployment rather than as its plane has
			// no such standing, declares nothing here, and runs no agents
			// surface at all.
			AgentsDial: n.url,
			Ready:      func(addr string) { ready <- addr },
		})
	}()
	select {
	case addr := <-ready:
		n.helmURL = "http://" + addr
		return nil
	case err := <-n.helmErr:
		n.helmErr = nil
		return fmt.Errorf("node: shell plane failed to start: %w", err)
	case <-time.After(20 * time.Second):
		return errors.New("node: shell plane did not become ready")
	}
}

// startFold runs the bundled OIDC provider (soulstream-idp's public embed
// seam): the fold's buckets live on this node's JetStream over its own
// bypass-lane connection, the seal seed under <state>/fold/, DCR on
// (hosted clients register themselves), the deployment audience fixed,
// and the founding persona seeded with the realm role — their first
// browser sign-in enrolls their passkey.
func (n *Node) startFold(ctx context.Context, cfg Config) error {
	st := cfg.State
	n.foldErr = make(chan error, 1)
	ready := make(chan string, 1)
	go func() {
		n.foldErr <- foldembed.Run(ctx, foldembed.Options{
			Issuer:        st.FoldIssuer,
			Listen:        st.FoldListen,
			StateDir:      filepath.Join(cfg.StateDir, "fold"),
			NATSURL:       n.url,
			NATSCreds:     ceremony.UserCredsPath(cfg.StateDir, "fold"),
			TokenAudience: st.FoldAudience,
			EnableDCR:     true,
			SeedUsers: []foldembed.SeedUser{{
				Username:    ceremony.FoundingPersona,
				DisplayName: ceremony.FoundingPersona,
				Roles:       []string{"admin", "realm"},
			}},
			InviteSink: func(_, token string) { n.foldInvite = token },
			Ready:      func(addr string) { ready <- addr },
		})
	}()
	select {
	case <-ready:
		n.foldURL = st.FoldIssuer
		return nil
	case err := <-n.foldErr:
		n.foldErr = nil
		return fmt.Errorf("node: fold plane failed to start: %w", err)
	case <-time.After(15 * time.Second):
		return errors.New("node: fold plane did not become ready")
	}
}

// startDoor wires the MCP door plane (design §8, as landed upstream in
// soulstream 018): streamable HTTP in, bearer passthrough to this node's
// own callout admission, the public tool surface out. The door custodies
// nothing — it reads only the public sentinel. SoulNode holds the
// listener so a bind conflict is a named refusal and tests get real
// ports.
func (n *Node) startDoor(cfg Config) error {
	st := cfg.State
	l, err := net.Listen("tcp", st.DoorListen)
	if err != nil {
		return fmt.Errorf("node: door cannot listen on %s (change planes.door.listen in %s): %w",
			st.DoorListen, filepath.Join(cfg.StateDir, "config.json"), err)
	}
	d, err := door.New(door.Config{
		Listen:       st.DoorListen,
		Realm:        st.Realm,
		NATSURL:      n.url,
		SentinelPath: ceremony.SentinelPath(cfg.StateDir),
		// Public mode (upstream's own switch): the advertised resource
		// identifier and the AS the resource metadata names. HTTPS is
		// deployment fronting before the loopback listener.
		PublicURL:  st.DoorPublicURL,
		AuthIssuer: st.DoorAuthIssuer,
		Logger:     n.audit,
	})
	if err != nil {
		_ = l.Close()
		return fmt.Errorf("node: door: %w", err)
	}
	n.doorNode = d
	n.doorURL = "http://" + l.Addr().String()
	n.doorSrv = &http.Server{Handler: d.Handler()}
	n.doorErr = make(chan error, 1)
	go func() { n.doorErr <- n.doorSrv.Serve(l) }()
	return nil
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
		nats.Name("soulstream-archivist"))
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
	// The door goes first: no new sessions while the planes drain, and
	// its pool closes before the connections underneath it do.
	if n.doorSrv != nil {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = n.doorSrv.Shutdown(shCtx)
		cancel()
		n.doorNode.Close()
		select {
		case err := <-n.doorErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				n.audit.Error("door plane exited", "err", err)
			}
		case <-time.After(5 * time.Second):
		}
	}
	if n.cancel != nil {
		n.cancel()
		select {
		case <-n.planes: // embed.Run drained and returned
		case <-time.After(10 * time.Second):
		}
		if n.foldErr != nil {
			select {
			case err := <-n.foldErr: // the fold rides the same ctx
				if err != nil && !errors.Is(err, context.Canceled) {
					n.audit.Error("fold plane exited", "err", err)
				}
			case <-time.After(10 * time.Second):
			}
		}
		if n.helmErr != nil {
			select {
			case err := <-n.helmErr: // the shell rides the same ctx
				if err != nil && !errors.Is(err, context.Canceled) {
					n.audit.Error("shell plane exited", "err", err)
				}
			case <-time.After(10 * time.Second):
			}
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

// The no-toolchain agent path (design soulstream/0002): the same binary
// that runs the realm answers `wrap` — the personal wrapper of workloads
// design 0004 — and provides the stdio MCP server it points the harness at
// (`mcp`). Both verbs are thin mains over libraries go.mod already pins,
// and both read the same lane: the five SOULSTREAM_* names the Agents
// screen mints. Nothing here reads a config file, because a config file
// can never carry a credential.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream-core/mcpserver"
	"github.com/impire-io/soulstream-core/realm"

	"github.com/impire-io/soulstream-workloads/declaration"
	"github.com/impire-io/soulstream-workloads/wrap"
	"github.com/impire-io/soulstream/door"
)

// laneFromEnv is the whole of how an agent's identity arrives: the block
// the Agents screen shows once, exported as environment. Flags may
// override individual values where a verb offers them.
func laneFromEnv() wrap.Lane {
	lane := wrap.Lane{
		URL:       os.Getenv("SOULSTREAM_URL"),
		CredsFile: os.Getenv("SOULSTREAM_CREDS"),
		Token:     os.Getenv("SOULSTREAM_TOKEN"),
		Realm:     os.Getenv("SOULSTREAM_REALM"),
		Persona:   os.Getenv("SOULSTREAM_PERSONA"),
	}
	// The door's outbound identity (external-tools.md D41), declared
	// through the lane because wrap scrubs the harness environment on
	// purpose: the subject a personal agent acts for, the delegation
	// authorizing it, and the account segment when the deployment wants
	// it explicit rather than derived.
	extra := map[string]string{}
	for _, key := range []string{
		"SOULSTREAM_ACCOUNT", "SOULSTREAM_SUBJECT",
		"SOULSTREAM_DELEGATION_PAYLOAD", "SOULSTREAM_DELEGATION_SIG",
	} {
		if v := os.Getenv(key); v != "" {
			extra[key] = v
		}
	}
	if len(extra) > 0 {
		lane.MCPExtraEnv = extra
	}
	return lane
}

// checkLane refuses a lane that cannot connect, naming the missing piece —
// an agent that will not start should say why in the first line.
func checkLane(lane wrap.Lane) error {
	switch {
	case lane.Persona == "":
		return fmt.Errorf("a persona is required (SOULSTREAM_PERSONA — the Agents screen's block carries it)")
	case lane.Realm == "":
		return fmt.Errorf("a realm is required (SOULSTREAM_REALM — the Agents screen's block carries it)")
	case lane.URL == "":
		return fmt.Errorf("a connection is required (SOULSTREAM_URL — the Agents screen's block carries it)")
	}
	return nil
}

// cmdWrap runs one agent where you are: it wraps the assistant already
// installed (and signed in) on this machine so mentions of the agent's
// persona become invocations, and every wake leaves exactly one outcome in
// the topic. The harness reaches its tools through this same executable (`mcp` below)
// — no second binary exists anywhere in the path.
func cmdWrap(args []string, errw io.Writer) error {
	fs := flag.NewFlagSet("wrap", flag.ContinueOnError)
	fs.SetOutput(errw)
	harness := fs.String("harness", "", "preset: claude | codex")
	templateFile := fs.String("template", "", "custom template file (overrides --harness)")
	scratch := fs.String("scratch", "", "run-directory root (default: a temp dir)")
	runTimeout := fs.Duration("run-timeout", 150*time.Second, "harness time budget per attempt")
	retries := fs.Int("retries", 2, "harness attempts per wake before the self-report")
	inboxLimit := fs.Int("inbox-limit", 0, "catch-up depth (0 = the default of 50)")
	declFile := fs.String("declaration", "", "agent declaration file — its wake entries drive the engine (mention-only without it)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	lane := laneFromEnv()
	if err := checkLane(lane); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding my own executable for the MCP server: %w", err)
	}
	lane.MCPCommandLoc = exe
	lane.MCPArgs = []string{"mcp"}

	var tpl wrap.Template
	switch {
	case *templateFile != "":
		tpl, err = wrap.LoadTemplate(*templateFile)
	case *harness != "":
		tpl, err = wrap.Preset(*harness, lane)
	default:
		return fmt.Errorf("pick an assistant: --harness claude|codex, or --template <file>")
	}
	if err != nil {
		return err
	}

	root := *scratch
	if root == "" {
		root = filepath.Join(os.TempDir(), "soulstream-wrap", lane.Persona)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The connection IS the credential check: a revoked agent is refused
	// here, loudly, and nothing is ever posted in its name.
	client, err := realm.Connect(ctx, realm.Config{
		URL: lane.URL, CredsFile: lane.CredsFile, Token: lane.Token,
		Realm: lane.Realm, Persona: lane.Persona,
	})
	if err != nil {
		return fmt.Errorf("this agent could not get in (revoked, or the realm is unreachable): %w", err)
	}
	defer func() { _ = client.Close() }()

	log := slog.New(slog.NewTextHandler(errw, nil))
	// The agent announces itself and lights its lamp (upstream ask #3,
	// episode 0124): a directory floor into absence only, and the
	// presence lease held beside the run loop. Both advisory — the run
	// loop never waits on them, and their failures are log lines.
	ensureProfile(ctx, client, log)
	farewell := holdPresence(ctx, client, log)

	cfg := wrap.Config{
		Persona:    lane.Persona,
		Template:   tpl,
		Scratch:    root,
		RunTimeout: *runTimeout,
		Retries:    *retries,
		InboxLimit: *inboxLimit,
	}
	if *declFile != "" {
		// Declaration-driven operation (workloads design 0005): the
		// record's wake vocabulary drives the engine. The declaration's
		// persona must be the credential's persona — the connection is
		// the authority.
		raw, err := os.ReadFile(*declFile)
		if err != nil {
			return fmt.Errorf("read declaration: %w", err)
		}
		d, err := declaration.Parse(raw)
		if err != nil {
			return err
		}
		cfg, err = wrap.DeclaredConfig(cfg, d, client)
		if err != nil {
			return err
		}
	}
	w := &wrap.Wrapper{
		Config: cfg,
		Client: client,
		Log:    log,
	}
	runErr := w.Run(ctx)
	// The goodbye lands before the deferred Close tears the connection
	// down — Hold writes it on its own short context once ctx ends.
	farewell()
	if runErr != nil && runErr != context.Canceled {
		return runErr
	}
	return nil
}

// cmdMCP is the stdio MCP server out of this binary: the realm's tools
// served to whatever assistant launched it. Environment-only on purpose —
// no context files, no keystores; an agent with no signing key speaks
// unsigned, and this verb does not pretend otherwise.
func cmdMCP(args []string, errw io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(errw)
	url := fs.String("url", "", "server address to dial (env: SOULSTREAM_URL)")
	creds := fs.String("creds", "", "NATS credentials file — the deployment's sentinel (env: SOULSTREAM_CREDS)")
	token := fs.String("token", "", "access token; prefer SOULSTREAM_TOKEN — a flag is visible to every process on the machine")
	realmName := fs.String("realm", "", "realm name (env: SOULSTREAM_REALM)")
	persona := fs.String("persona", "", "persona name (env: SOULSTREAM_PERSONA)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	lane := laneFromEnv()
	for flagVal, target := range map[*string]*string{
		url: &lane.URL, creds: &lane.CredsFile, token: &lane.Token,
		realmName: &lane.Realm, persona: &lane.Persona,
	} {
		if *flagVal != "" {
			*target = *flagVal
		}
	}
	if err := checkLane(lane); err != nil {
		return err
	}

	ctx := context.Background()
	client, err := realm.Connect(ctx, realm.Config{
		URL: lane.URL, CredsFile: lane.CredsFile, Token: lane.Token,
		Realm: lane.Realm, Persona: lane.Persona,
	})
	if err != nil {
		return fmt.Errorf("this agent could not get in (revoked, or the realm is unreachable): %w", err)
	}
	defer func() { _ = client.Close() }()

	srv := mcpserver.NewServer(client)
	// The forwarding half (external-tools.md D41): the realm's tool
	// catalog re-exposed beside the record's tools — endpoints only,
	// authority fetched per call, refusals in words. A realm with no
	// catalog leaves the door exactly as it always was; an entry that
	// isn't serving is a stderr note, never a failed door.
	for _, note := range door.Attach(ctx, srv, client, door.Config{
		Account:           os.Getenv("SOULSTREAM_ACCOUNT"),
		Subject:           os.Getenv("SOULSTREAM_SUBJECT"),
		DelegationPayload: os.Getenv("SOULSTREAM_DELEGATION_PAYLOAD"),
		DelegationSig:     os.Getenv("SOULSTREAM_DELEGATION_SIG"),
	}) {
		fmt.Fprintln(errw, "door:", note)
	}
	return srv.Run(ctx, &mcp.StdioTransport{})
}

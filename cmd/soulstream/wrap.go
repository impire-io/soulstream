// The no-toolchain agent path (design soulstream/0002): the same binary
// that runs the realm answers `wrap` — the personal wrapper of workloads
// design 0004 — and provides the stdio tool door it points the harness at
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
	"github.com/impire-io/soulstream-workloads/wrap"
)

// laneFromEnv is the whole of how an agent's identity arrives: the block
// the Agents screen shows once, exported as environment. Flags may
// override individual values where a verb offers them.
func laneFromEnv() wrap.Lane {
	return wrap.Lane{
		URL:       os.Getenv("SOULSTREAM_URL"),
		CredsFile: os.Getenv("SOULSTREAM_CREDS"),
		Token:     os.Getenv("SOULSTREAM_TOKEN"),
		Realm:     os.Getenv("SOULSTREAM_REALM"),
		Persona:   os.Getenv("SOULSTREAM_PERSONA"),
	}
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
// the topic. The harness's tool door is this same executable (`mcp` below)
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
	if err := fs.Parse(args); err != nil {
		return err
	}

	lane := laneFromEnv()
	if err := checkLane(lane); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding my own executable for the tool door: %w", err)
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

	w := &wrap.Wrapper{
		Config: wrap.Config{
			Persona:    lane.Persona,
			Template:   tpl,
			Scratch:    root,
			RunTimeout: *runTimeout,
			Retries:    *retries,
			InboxLimit: *inboxLimit,
		},
		Client: client,
		Log:    slog.New(slog.NewTextHandler(errw, nil)),
	}
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

// cmdMCP is the stdio tool door out of this binary: the realm's MCP tools
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

	return mcpserver.NewServer(client).Run(ctx, &mcp.StdioTransport{})
}

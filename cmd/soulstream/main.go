// Command soulstream is the single-binary distribution of the Soulstream
// ecosystem: `init` founds a realm into a state directory (the whole
// ceremony, zero manual steps — constitution V), `up` runs it (embedded
// operator-mode server + the identity plane, everything on ordinary
// loopback connections — constitution III).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/impire-io/soulstream/ceremony"
	"github.com/impire-io/soulstream/internal/version"
	"github.com/impire-io/soulstream/node"
)

const usage = `soulstream — your realm in one binary

Usage:
  soulstream init [--state DIR] [--listen ADDR] [--realm NAME]
                [--mcp-listen ADDR] [--signin-listen ADDR]
                                                found a realm (prints your token ONCE)
  soulstream init --byo self-hosted --url nats://HOST:PORT [--realm NAME]
                                                found on your own operator-mode server:
                                                emits the kit; re-run with --auth-account
                                                and --realm-account once it is applied
  soulstream init --byo synadia-cloud --url ... --synadia-system NAME
                                                found on a Synadia Cloud BYON (reads
                                                SOULSTREAM_SYNADIA_TOKEN from the env)
  soulstream up   [--state DIR]                   run it until interrupted
  soulstream workload start <declaration.json> [--state DIR]
                                                run one declared workload (node must be up)
  soulstream wrap --harness claude|codex | --template FILE
                                                run your agent here: mentions become answers.
                                                Reads the five SOULSTREAM_* values from the
                                                Agents screen's block; needs nothing else.
  soulstream mcp                                the stdio MCP server wrap launches from this
                                                same binary (same five values; flags override)
  soulstream adopt [--state DIR] [--force]
                                                adopt a realm founded before the canonical
                                                v2 break (hq 0112) — refuses when the realm
                                                already holds v1-signed records
  soulstream version

State dir: --state, else $SOULSTREAM_STATE, else <user config dir>/soulstream.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "init":
		err = cmdInit(args[1:], out, errw)
	case "up":
		err = cmdUp(args[1:], out, errw)
	case "workload":
		err = cmdWorkload(args[1:], out, errw)
	case "wrap":
		err = cmdWrap(args[1:], errw)
	case "mcp":
		err = cmdMCP(args[1:], errw)
	case "adopt":
		err = cmdAdopt(args[1:], out, errw)
	case "version":
		fmt.Fprintln(out, version.Version)
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
	default:
		fmt.Fprintf(errw, "soulstream: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(errw, "soulstream:", err)
		return 1
	}
	return 0
}

// stateDir resolves the state directory: flag, env, OS config dir.
func stateDir(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if v := os.Getenv("SOULSTREAM_STATE"); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no state dir: %w (set --state or SOULSTREAM_STATE)", err)
	}
	return filepath.Join(base, "soulstream"), nil
}

func cmdInit(args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	listen := fs.String("listen", "127.0.0.1:4222", "loopback listener (written to config.json on the founding run)")
	realmName := fs.String("realm", "home", "realm name (written to config.json on the founding run)")
	mcpListen := fs.String("mcp-listen", "127.0.0.1:8080", "the MCP endpoint's loopback listener (written to config.json on the founding run)")
	signinListen := fs.String("signin-listen", "127.0.0.1:8378", "the sign-in service's loopback listener (written to config.json)")
	helmListen := fs.String("shell-listen", "127.0.0.1:8500", "the shell console's loopback listener (written to config.json)")
	byoFlavour := fs.String("byo", "", "found on a bring-your-own server: self-hosted or synadia-cloud (design 0003)")
	byoURL := fs.String("url", "", "the BYO server's client URL (nats://host:port)")
	authAccount := fs.String("auth-account", "", "BYO hand-back: the AUTH account's public key (from the kit's §3)")
	realmAccount := fs.String("realm-account", "", "BYO hand-back: the realm account's public key (from the kit's §3)")
	synadiaSystem := fs.String("synadia-system", "", "synadia-cloud: the Synadia Cloud system (name or id)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	listenSet, realmSet, urlSet := false, false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "listen":
			listenSet = true
		case "realm":
			realmSet = true
		case "url":
			urlSet = true
		}
	})
	dir, err := stateDir(*stateFlag)
	if err != nil {
		return err
	}

	empty, err := ceremony.Empty(dir)
	if err != nil {
		return err
	}
	if !empty {
		return initExisting(dir, out, errw, initFlags{
			listen: *listen, listenSet: listenSet,
			realm: *realmName, realmSet: realmSet,
			byoFlavour: *byoFlavour, byoURL: *byoURL, urlSet: urlSet,
			authAccount: *authAccount, realmAccount: *realmAccount,
		})
	}

	if *byoFlavour != "" {
		if listenSet {
			return fmt.Errorf("--listen is the embedded server's flag — a BYO realm has no embedded listener (design 0003 §6)")
		}
		st, err := ceremony.GenerateBYO(*byoFlavour, *byoURL, *realmName)
		if err != nil {
			return err
		}
		st.MCPListen = *mcpListen
		st.SignInListen = *signinListen
		if _, port, err := net.SplitHostPort(*signinListen); err == nil {
			st.SignInIssuer = "http://localhost:" + port
		}
		st.HelmListen = *helmListen
		switch *byoFlavour {
		case ceremony.FlavourSelfHosted:
			if *authAccount != "" || *realmAccount != "" {
				return fmt.Errorf("--auth-account and --realm-account belong to the second run — this run generates the kit those keys answer")
			}
			if err := st.Save(dir); err != nil {
				return err
			}
			return emitKit(st, dir, out)
		case ceremony.FlavourSynadiaCloud:
			if *synadiaSystem == "" {
				return fmt.Errorf("the synadia-cloud flavour needs --synadia-system (the Cloud system name)")
			}
			st.SynadiaSystem = *synadiaSystem
			if err := st.Save(dir); err != nil {
				return err
			}
			if err := synadiaAccountHalf(st, dir, errw); err != nil {
				return err
			}
			return byoFound(st, dir, out, errw)
		}
	}
	if *byoURL != "" || *authAccount != "" || *realmAccount != "" || *synadiaSystem != "" {
		return fmt.Errorf("--url, --auth-account, --realm-account, and --synadia-system are BYO flags — pass --byo self-hosted|synadia-cloud")
	}

	// The founding run: generate, persist, boot transiently, perform the
	// founding acts, print the one secret.
	st, err := ceremony.Generate(*listen, *realmName)
	if err != nil {
		return err
	}
	st.MCPListen = *mcpListen
	st.SignInListen = *signinListen
	// Keep the issuer's port in step with the chosen listener (host stays
	// localhost — WebAuthn refuses a bare IP); a public deployment sets
	// planes.fold.issuer to its fronted name in config afterward.
	if _, port, err := net.SplitHostPort(*signinListen); err == nil {
		st.SignInIssuer = "http://localhost:" + port
	}
	st.HelmListen = *helmListen
	if err := st.Save(dir); err != nil {
		return err
	}
	n, err := node.Start(node.Config{StateDir: dir, State: st, AuditWriter: errw})
	if err != nil {
		return err
	}
	token, err := node.Found(n, st, dir)
	n.Stop()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "soulstream: realm %q founded at %s\n", st.Realm, dir)
	fmt.Fprintf(out, "your access token (shown once, never stored):\n\n    %s\n\n", token)
	if invite := n.FoldInvite(); invite != "" {
		fmt.Fprintf(out, "your passkey enrollment invite (single use, shown once):\n\n    %s/enroll?invite=%s\n\n", n.SignInURL(), invite)
	}
	fmt.Fprintln(out, "run `soulstream up` to serve — it prints the MCP, sign-in, and shell URLs.")
	if st.MCPEnabled {
		fmt.Fprintf(out, "point an MCP client at http://%s with that token as its bearer,\n", st.MCPListen)
		fmt.Fprintf(out, "or a NATS client at nats://%s with sentinel %s\n",
			st.Listen, ceremony.SentinelPath(dir))
	} else {
		fmt.Fprintf(out, "point a client at nats://%s with sentinel %s\n",
			st.Listen, ceremony.SentinelPath(dir))
	}
	return nil
}

// initFlags carries what the init run said, so a re-run on existing
// state can refuse disagreements by name.
type initFlags struct {
	listen       string
	listenSet    bool
	realm        string
	realmSet     bool
	byoFlavour   string
	byoURL       string
	urlSet       bool
	authAccount  string
	realmAccount string
}

// initExisting is init on an already-touched directory: verify and
// report for a founded realm (regenerate nothing, mint nothing —
// contracts/cli.md); for a BYO realm still awaiting its account half,
// resume the phase the state is in.
func initExisting(dir string, out, errw io.Writer, f initFlags) error {
	st, err := ceremony.Verify(dir)
	if err != nil {
		return err
	}
	if f.byoFlavour != "" && !st.BYO() {
		return fmt.Errorf("state at %s is an embedded-server realm — the substrate is fixed at founding; found a new realm to bring your own server", dir)
	}
	if f.byoFlavour != "" && st.BYO() && f.byoFlavour != st.BYOFlavour {
		return fmt.Errorf("state at %s was founded with byo.flavour %q — the substrate is fixed at founding", dir, st.BYOFlavour)
	}
	if f.urlSet && st.BYO() && f.byoURL != st.BYOURL {
		return fmt.Errorf("state at %s already dials %s — config.json is the configuration; edit byo.url there", dir, st.BYOURL)
	}
	if f.listenSet && st.BYO() {
		return fmt.Errorf("--listen is the embedded server's flag — this realm dials %s", st.BYOURL)
	}
	if f.realmSet && f.realm != st.Realm {
		return fmt.Errorf("state at %s is realm %q — the name is fixed at founding", dir, st.Realm)
	}
	if ceremony.Founded(dir) {
		if f.listenSet && !st.BYO() && f.listen != st.Listen {
			return fmt.Errorf("state at %s already listens on %s — config.json is the configuration; edit it there or found a new realm", dir, st.Listen)
		}
		where := "listener " + st.Listen
		if st.BYO() {
			where = "server " + st.BYOURL
		}
		fmt.Fprintf(out, "soulstream: state at %s verified — %d artifacts, realm %q, %s\n",
			dir, st.ArtifactCount(), st.Realm, where)
		return nil
	}
	if !st.BYO() {
		return ceremony.ErrIncomplete
	}

	// A BYO realm between the phases: merge the hand-back, then either
	// found or re-emit the kit — idempotent, never an error.
	if f.authAccount != "" {
		st.AuthPub = f.authAccount
	}
	if f.realmAccount != "" {
		st.RealmPub = f.realmAccount
	}
	if st.BYOFlavour == ceremony.FlavourSynadiaCloud {
		if err := synadiaAccountHalf(st, dir, errw); err != nil {
			return err
		}
		return byoFound(st, dir, out, errw)
	}
	if st.AuthPub == "" || st.RealmPub == "" {
		return emitKit(st, dir, out)
	}
	if err := st.Save(dir); err != nil { // persist the hand-back
		return err
	}
	return byoFound(st, dir, out, errw)
}

// emitKit writes and prints the kit — regenerated identically on every
// run until the account half exists (design 0003 §4).
func emitKit(st *ceremony.State, dir string, out io.Writer) error {
	kit := ceremony.Kit(st, dir)
	if err := os.WriteFile(ceremony.KitPath(dir), []byte(kit), 0o600); err != nil {
		return fmt.Errorf("write kit: %w", err)
	}
	fmt.Fprint(out, kit)
	fmt.Fprintf(out, "\nsoulstream: the kit is at %s — apply it, then hand back the two account keys (its §3).\n", ceremony.KitPath(dir))
	return nil
}

// byoFound is the BYO wire half (design 0003 §3): mint the bypass-lane
// users, verify the substrate behaviourally, boot the planes against it,
// perform the founding acts, and run the one callout smoke round. The
// sentinel marks the realm founded; a failed smoke takes the marker back
// out — a realm whose callout does not admit is not founded.
func byoFound(st *ceremony.State, dir string, out, errw io.Writer) error {
	if len(st.OpsCreds) == 0 || len(st.IssuerCreds) == 0 || len(st.ServiceCreds) == 0 ||
		len(st.ArchivistCreds) == 0 || len(st.RunnerCreds) == 0 || len(st.SignInCreds) == 0 {
		if err := st.MintBYOUsers(); err != nil {
			return err
		}
	}
	if err := st.Save(dir); err != nil {
		return err
	}
	if err := node.ProbeSubstrate(st, dir); err != nil {
		return err
	}
	n, err := node.Start(node.Config{StateDir: dir, State: st, AuditWriter: errw})
	if err != nil {
		return err
	}
	token, err := node.Found(n, st, dir)
	if err != nil {
		n.Stop()
		return err
	}
	smokeErr := node.SmokeAdmission(st.BYOURL, ceremony.SentinelPath(dir), token)
	invite := n.FoldInvite()
	signInURL := n.SignInURL()
	n.Stop()
	if smokeErr != nil {
		_ = os.Remove(ceremony.SentinelPath(dir))
		return smokeErr
	}
	fmt.Fprintf(out, "soulstream: realm %q founded on %s (%s)\n", st.Realm, st.BYOURL, st.BYOFlavour)
	fmt.Fprintf(out, "your access token (shown once, never stored):\n\n    %s\n\n", token)
	if invite != "" {
		fmt.Fprintf(out, "your passkey enrollment invite (single use, shown once):\n\n    %s/enroll?invite=%s\n\n", signInURL, invite)
	}
	fmt.Fprintln(out, "run `soulstream up` to serve — it prints the MCP, sign-in, and shell URLs.")
	fmt.Fprintf(out, "point a NATS client at %s with sentinel %s and that token\n",
		st.BYOURL, ceremony.SentinelPath(dir))
	return nil
}

func cmdUp(args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("up", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := stateDir(*stateFlag)
	if err != nil {
		return err
	}

	empty, err := ceremony.Empty(dir)
	if err != nil {
		return err
	}
	if empty {
		return fmt.Errorf("state at %s is not initialized — run `soulstream init` first", dir)
	}
	st, err := ceremony.Verify(dir)
	if err != nil {
		return err
	}
	if !ceremony.Founded(dir) {
		return notFounded(st, dir)
	}

	n, err := node.Start(node.Config{StateDir: dir, State: st, AuditWriter: errw})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "soulstream: state %s (realm %q)\n", dir, st.Realm)
	if st.BYO() {
		fmt.Fprintf(out, "soulstream: dialing %s (bring-your-own server, %s)\n", st.BYOURL, st.BYOFlavour)
	} else {
		fmt.Fprintf(out, "soulstream: listening on nats://%s (loopback)\n", st.Listen)
	}
	fmt.Fprintln(out, "soulstream: identity plane serving")
	if st.MemoryEnabled {
		fmt.Fprintln(out, "soulstream: memory plane serving")
	}
	printEndpoints(out, n, st)
	if st.SignInEnabled {
		if invite := n.FoldInvite(); invite != "" {
			fmt.Fprintf(out, "soulstream: passkey enrollment invite (single use, shown once):\n\n    %s/enroll?invite=%s\n\n", n.SignInURL(), invite)
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Fprintln(out, "soulstream: draining")
	n.Stop()
	return nil
}

// printEndpoints logs every URL a person or client needs, once the node
// is up: the MCP endpoint for assistants, the sign-in page, and the
// shell console. The MCP endpoint and the sign-in service are separate
// services on separate loopback ports; in public mode each needs its own
// fronted route (a shared hostname would have the MCP catch-all swallow
// the sign-in pages), so this names both public URLs when set.
// Administration lives in the shell console — nothing else is printed.
func printEndpoints(out io.Writer, n *node.Node, st *ceremony.State) {
	if st.MCPEnabled {
		fmt.Fprintf(out, "soulstream: MCP (assistants) %s\n", n.MCPURL())
		if st.MCPPublicURL != "" {
			fmt.Fprintf(out, "soulstream:   public MCP URL %s (front this to the MCP port)\n", st.MCPPublicURL)
		}
	}
	if st.SignInEnabled {
		fmt.Fprintf(out, "soulstream: sign-in          %s/login/\n", n.SignInURL())
		if st.MCPPublicURL != "" {
			fmt.Fprintf(out, "soulstream:   public sign-in %s (a DISTINCT route from the MCP URL)\n", st.SignInIssuer)
		}
	}
	if st.HelmEnabled {
		fmt.Fprintf(out, "soulstream: shell console    %s\n", n.HelmURL())
		if st.HelmPublicURL != "" {
			fmt.Fprintf(out, "soulstream:   public console %s (front this to the shell port)\n", st.HelmPublicURL)
		}
	}
}

func cmdWorkload(args []string, out, errw io.Writer) error {
	if len(args) < 1 || args[0] != "start" {
		return fmt.Errorf("workload needs a subcommand: start <declaration.json>")
	}
	fs := flag.NewFlagSet("workload start", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("workload start needs exactly one declaration file")
	}
	dir, err := stateDir(*stateFlag)
	if err != nil {
		return err
	}
	st, err := ceremony.Verify(dir)
	if err != nil {
		return err
	}
	if !ceremony.Founded(dir) {
		return notFounded(st, dir)
	}
	serverURL := "nats://" + st.Listen
	if st.BYO() {
		serverURL = st.BYOURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(out, "soulstream: launching workload from %s\n", fs.Arg(0))
	if err := node.RunWorkload(ctx, node.Config{StateDir: dir, State: st, AuditWriter: errw},
		serverURL, fs.Arg(0)); err != nil {
		return err
	}
	fmt.Fprintln(out, "soulstream: workload finished (terminal work op recorded)")
	return nil
}

// notFounded names the state a not-yet-founded directory is in: a BYO
// realm awaiting its account half is a phase, not damage.
func notFounded(st *ceremony.State, dir string) error {
	if st.BYO() {
		return fmt.Errorf("state at %s awaits its account half — apply the kit (%s) and re-run `soulstream init` with the two account keys", dir, ceremony.KitPath(dir))
	}
	return ceremony.ErrIncomplete
}

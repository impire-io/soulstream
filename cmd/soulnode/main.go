// Command soulnode is the single-binary distribution of the Soulstream
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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/impire-io/soulnode/ceremony"
	"github.com/impire-io/soulnode/internal/version"
	"github.com/impire-io/soulnode/node"
)

const usage = `soulnode — your realm in one binary

Usage:
  soulnode init [--state DIR] [--listen ADDR] [--realm NAME]
                                                found a realm (prints your token ONCE)
  soulnode up   [--state DIR]                   run it until interrupted
  soulnode workload start <declaration.json> [--state DIR]
                                                run one declared workload (node must be up)
  soulnode version

State dir: --state, else $SOULNODE_STATE, else <user config dir>/soulnode.
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
	case "version":
		fmt.Fprintln(out, version.Version)
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
	default:
		fmt.Fprintf(errw, "soulnode: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(errw, "soulnode:", err)
		return 1
	}
	return 0
}

// stateDir resolves the state directory: flag, env, OS config dir.
func stateDir(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if v := os.Getenv("SOULNODE_STATE"); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no state dir: %w (set --state or SOULNODE_STATE)", err)
	}
	return filepath.Join(base, "soulnode"), nil
}

func cmdInit(args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	listen := fs.String("listen", "127.0.0.1:4222", "loopback listener (written to config.json on the founding run)")
	realmName := fs.String("realm", "home", "realm name (written to config.json on the founding run)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	listenSet, realmSet := false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "listen":
			listenSet = true
		case "realm":
			realmSet = true
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
		// Already-touched directory: verify and report — regenerate
		// nothing, mint nothing (contracts/cli.md).
		st, err := ceremony.Verify(dir)
		if err != nil {
			return err
		}
		if !ceremony.Founded(dir) {
			return ceremony.ErrIncomplete
		}
		if listenSet && *listen != st.Listen {
			return fmt.Errorf("state at %s already listens on %s — config.json is the configuration; edit it there or found a new realm", dir, st.Listen)
		}
		if realmSet && *realmName != st.Realm {
			return fmt.Errorf("state at %s is realm %q — the name is fixed at founding", dir, st.Realm)
		}
		fmt.Fprintf(out, "soulnode: state at %s verified — %d artifacts, realm %q, listener %s\n",
			dir, ceremony.ArtifactCount(), st.Realm, st.Listen)
		return nil
	}

	// The founding run: generate, persist, boot transiently, perform the
	// founding acts, print the one secret.
	st, err := ceremony.Generate(*listen, *realmName)
	if err != nil {
		return err
	}
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
	fmt.Fprintf(out, "soulnode: realm %q founded at %s\n", st.Realm, dir)
	fmt.Fprintf(out, "your access token (shown once, never stored):\n\n    %s\n\n", token)
	fmt.Fprintf(out, "point a client at nats://%s with sentinel %s\n",
		st.Listen, ceremony.SentinelPath(dir))
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
		return fmt.Errorf("state at %s is not initialized — run `soulnode init` first", dir)
	}
	st, err := ceremony.Verify(dir)
	if err != nil {
		return err
	}
	if !ceremony.Founded(dir) {
		return ceremony.ErrIncomplete
	}

	n, err := node.Start(node.Config{StateDir: dir, State: st, AuditWriter: errw})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "soulnode: state %s (realm %q)\n", dir, st.Realm)
	fmt.Fprintf(out, "soulnode: listening on nats://%s (loopback)\n", st.Listen)
	fmt.Fprintln(out, "soulnode: identity plane serving")
	if st.MemoryEnabled {
		fmt.Fprintln(out, "soulnode: memory plane serving")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Fprintln(out, "soulnode: draining")
	n.Stop()
	return nil
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
		return ceremony.ErrIncomplete
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(out, "soulnode: launching workload from %s\n", fs.Arg(0))
	if err := node.RunWorkload(ctx, node.Config{StateDir: dir, State: st, AuditWriter: errw},
		"nats://"+st.Listen, fs.Arg(0)); err != nil {
		return err
	}
	fmt.Fprintln(out, "soulnode: workload finished (terminal work op recorded)")
	return nil
}

// Package cli implements the soulstream terminal client. The logic lives here (not in
// cmd/) so it is testable: Run takes an explicit context, output writers, and an
// injectable Connector, letting tests drive every command against an in-process server.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
)

// Config is the resolved connection configuration.
type Config struct {
	Context string
	Realm   string
	Persona string
	// KeyFile is the persona's signing-seed file ("" resolves env → default path).
	// When the resolved file exists, every write command signs its ops.
	KeyFile string
}

const usageText = `soulstream — a Soulstream client

Usage:
  soulstream [--context <name>] [--realm <name>] [--persona <name>] <command> [args...]

Commands:
  provision                          ensure the realm's stream and object store
  board [--json]                     list the topics on the board
  start <name> [--subject s] [--tag t]... [--parent path]
                                     start a topic; prints its path
  show <path> [--json]               print a topic's current state
  post <path> <body>                 post a turn (use @name to mention)
  comment <path> <op-id> <body>      comment, anchored to an op
  attach <path> <file> [--type ct] [--anchor op-id]
                                     attach a file; prints its object key
  get <object> <outfile> [--force]   download an attachment
  close <path>                       close a topic
  watch <path>                       stream a topic live (Ctrl-C to stop)
  inbox                              stream your notifications live (Ctrl-C to stop)
  key init                           create this persona's signing key
  key show                           print this persona's public signing key

Configuration (flags override environment):
  --context  / SOULSTREAM_CONTEXT    named NATS context
  --realm    / SOULSTREAM_REALM      realm name
  --persona  / SOULSTREAM_PERSONA    persona (required for write commands)
  --key-file / SOULSTREAM_KEY_FILE   signing-seed file (default: config dir; when
                                     present, published ops are signed)
`

// Main wires os streams, a SIGINT-cancellable context, and the natscontext connector.
func Main(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return Run(ctx, args, os.Stdout, os.Stderr, realmConnect)
}

// Run parses args, resolves config, connects via connect, and dispatches to a command,
// returning the process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, connect Connector) int {
	global := flag.NewFlagSet("soulstream", flag.ContinueOnError)
	global.SetOutput(stderr)
	global.Usage = func() { fmt.Fprint(stderr, usageText) }
	ctxName := global.String("context", os.Getenv("SOULSTREAM_CONTEXT"), "named NATS context")
	realmName := global.String("realm", os.Getenv("SOULSTREAM_REALM"), "realm name")
	persona := global.String("persona", os.Getenv("SOULSTREAM_PERSONA"), "persona name")
	keyFile := global.String("key-file", "", "signing-seed file (default: env, then config dir)")
	if err := global.Parse(args); err != nil {
		return 2
	}

	rest := global.Args()
	if len(rest) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	cfg := Config{Context: *ctxName, Realm: *realmName, Persona: *persona, KeyFile: *keyFile}
	cmd, cmdArgs := rest[0], rest[1:]

	switch cmd {
	case "provision":
		return cmdProvision(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "board":
		return cmdBoard(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "start":
		return cmdStart(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "show":
		return cmdShow(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "post":
		return cmdPost(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "comment":
		return cmdComment(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "attach":
		return cmdAttach(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "get":
		return cmdGet(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "close":
		return cmdClose(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "watch":
		return cmdWatch(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "inbox":
		return cmdInbox(ctx, connect, cfg, cmdArgs, stdout, stderr)
	case "key":
		return cmdKey(cfg, cmdArgs, stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	default:
		fmt.Fprintf(stderr, "soulstream: unknown command %q\n\n", cmd)
		fmt.Fprint(stderr, usageText)
		return 2
	}
}

// parseInterspersed parses fs allowing flags to appear before or after positional
// arguments (the stdlib flag package otherwise stops at the first positional). It uses
// the FlagSet's own definitions to know which flags consume a following value.
func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if len(a) > 1 && a[0] == '-' {
			flags = append(flags, a)
			if !strings.Contains(a, "=") {
				if f := fs.Lookup(strings.TrimLeft(a, "-")); f != nil && !isBoolFlag(f) && i+1 < len(args) {
					flags = append(flags, args[i+1])
					i++
				}
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return fs.Parse(append(flags, positionals...))
}

func isBoolFlag(f *flag.Flag) bool {
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}

// multiFlag collects a repeatable string flag (e.g. --tag).
type multiFlag []string

func (m *multiFlag) String() string { return fmt.Sprint([]string(*m)) }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

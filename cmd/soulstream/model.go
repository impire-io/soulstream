package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream/node"
)

// cmdModel is the catalogue in the hand (inference design 0001 §5): a
// virtual model NAME pointed at a route. Declarations name; the
// catalogue routes; re-pointing a name moves traffic — and the defaults
// that ride with it — without a single declaration changing.
//
// The ops travel the realm's own KV over the operator's creds, exactly as
// the account verbs travel the identity plane's sealed surface.
func cmdModel(args []string, out, errw io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("model needs a subcommand: set <name> [--pin MODEL] [--capability chat] | ls")
	}
	sub := args[0]
	fs := flag.NewFlagSet("model "+sub, flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	pin := fs.String("pin", "", "the concrete model this name resolves to (empty anycasts the capability)")
	capability := fs.String("capability", "chat", "the capability pool the name resolves into")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	switch sub {
	case "set":
		if fs.NArg() != 1 {
			return fmt.Errorf("model set needs exactly one name")
		}
	case "ls":
		if fs.NArg() != 0 {
			return fmt.Errorf("model ls takes no arguments")
		}
	default:
		return fmt.Errorf("unknown model subcommand %q: set | ls", sub)
	}

	dir, _, url, err := realmState(*stateFlag)
	if err != nil {
		return err
	}
	nc, err := dialOps(dir, url, "model")
	if err != nil {
		return err
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstream: %w", err)
	}
	ctx := context.Background()

	if sub == "set" {
		name := fs.Arg(0)
		if err := node.CatalogueSet(ctx, js, name, node.ModelEntry{
			Capability: *capability, ModelPin: *pin,
		}); err != nil {
			return err
		}
		where := "anycast — whichever instance serves " + *capability
		if *pin != "" {
			where = "pinned to " + *pin
		}
		fmt.Fprintf(out, "model %q set: %s\n", name, where)
		return nil
	}

	names, err := node.CatalogueList(ctx, js)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintln(out, "no models named yet — `soulstream model set <name> --pin <model>`")
		return nil
	}
	for _, n := range names {
		pinned := n.Entry.ModelPin
		if pinned == "" {
			pinned = "(anycast)"
		}
		fmt.Fprintf(out, "%s  %s  %s\n", n.Name, n.Entry.Capability, pinned)
	}
	return nil
}

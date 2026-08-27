package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-identity/client"

	"github.com/impire-io/soulstream/ceremony"
)

// cmdAccount is the tenancy surface in the hand (hq platform-topology
// D35/D47): accounts created, listed, shown, suspended, and resumed
// against the RUNNING node — the ops travel the identity plane's sealed
// surface over the operator's own creds, exactly as any administrative
// act does. On a deployment running no account authority (BYO — no
// operator material on this side of the boundary, design 0003) the
// service itself answers with that refusal, and this command shows it.
func cmdAccount(args []string, out, errw io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("account needs a subcommand: create <name> | list | show <name> | suspend <name> | resume <name>")
	}
	sub := args[0]
	fs := flag.NewFlagSet("account "+sub, flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	needsName := sub != "list"
	if needsName && fs.NArg() != 1 {
		return fmt.Errorf("account %s needs exactly one account name", sub)
	}
	if !needsName && fs.NArg() != 0 {
		return fmt.Errorf("account list takes no arguments")
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

	nc, err := nats.Connect(serverURL,
		nats.UserCredentials(ceremony.UserCredsPath(dir, "ops")),
		nats.Name("soulstream-account"))
	if err != nil {
		return fmt.Errorf("connect to the running realm at %s (is `soulstream up` running?): %w", serverURL, err)
	}
	defer nc.Close()
	ops := client.New(nc, st.RealmPub, "ops")

	show := func(a client.Account) {
		fmt.Fprintf(out, "%s  %s  %s\n", a.Name, a.Status, a.Account)
	}
	switch sub {
	case "create":
		a, err := ops.AccountCreate(fs.Arg(0))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "account %q created — sign-ins reach it the moment a token or role names it\n", a.Name)
		show(a)
		return nil
	case "list":
		accounts, err := ops.Accounts()
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			fmt.Fprintln(out, "no accounts created yet")
			return nil
		}
		for _, a := range accounts {
			show(a)
		}
		return nil
	case "show":
		a, err := ops.AccountResolve(fs.Arg(0))
		if err != nil {
			return err
		}
		show(a)
		return nil
	case "suspend":
		a, err := ops.AccountSuspend(fs.Arg(0))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "account %q suspended — new connections refused, data kept\n", a.Name)
		return nil
	case "resume":
		a, err := ops.AccountResume(fs.Arg(0))
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "account %q resumed\n", a.Name)
		return nil
	default:
		return fmt.Errorf("unknown account subcommand %q: create | list | show | suspend | resume", sub)
	}
}

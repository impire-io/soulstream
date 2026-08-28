package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/impire-io/soulstream/node"
)

// cmdAgent is the declared-agent surface in the hand (workloads design
// 0007): submit a declaration and walk away. The placement lands on the
// deployment's placement topic as an ordinary work item; a dispatcher
// node races for it, serves it through the wake engine, and answers in
// the agent's name for as long as the deployment runs — this process is
// done the moment the id is printed.
func cmdAgent(args []string, out, errw io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("agent needs a subcommand: submit <declaration.json>")
	}
	if args[0] != "submit" {
		return fmt.Errorf("unknown agent subcommand %q: submit", args[0])
	}
	fs := flag.NewFlagSet("agent submit", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("agent submit needs exactly one declaration file")
	}
	dir, st, url, err := realmState(*stateFlag)
	if err != nil {
		return err
	}
	id, err := node.SubmitAgent(context.Background(),
		node.Config{StateDir: dir, State: st, AuditWriter: errw}, url, fs.Arg(0))
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "placement %s submitted to %q — a dispatcher node serves it from here\n",
		id, st.DispatcherPlacements)
	return nil
}

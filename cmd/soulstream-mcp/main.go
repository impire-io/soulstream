// Command soulstream-mcp is an MCP (Model Context Protocol) server over stdio that lets
// an AI persona participate in Soulstream through tool calls. It acts as one configured
// persona for its whole session.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire/soulstream/internal/keystore"
	"github.com/impire/soulstream/internal/mcpserver"
	"github.com/impire/soulstream/realm"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("soulstream-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ctxName := fs.String("context", os.Getenv("SOULSTREAM_CONTEXT"), "named NATS context")
	realmName := fs.String("realm", os.Getenv("SOULSTREAM_REALM"), "realm name")
	persona := fs.String("persona", os.Getenv("SOULSTREAM_PERSONA"), "persona name")
	keyFile := fs.String("key-file", "", "signing-seed file (default: SOULSTREAM_KEY_FILE, then config dir)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *persona == "" {
		fmt.Fprintln(stderr, "soulstream-mcp: a persona is required (--persona or SOULSTREAM_PERSONA)")
		return 2
	}

	// Sign automatically when the persona's key exists; no key just means unsigned.
	keyPath, err := keystore.ResolveKeyFile(*keyFile, *realmName, *persona)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream-mcp: %v\n", err)
		return 1
	}
	signer, err := keystore.LoadKey(keyPath)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream-mcp: %v\n", err)
		return 1
	}

	ctx := context.Background()
	c, err := realm.Connect(ctx, realm.Config{ContextName: *ctxName, Realm: *realmName, Persona: *persona, Signer: signer})
	if err != nil {
		fmt.Fprintf(stderr, "soulstream-mcp: %v\n", err)
		return 1
	}
	defer func() { _ = c.Close() }()

	if err := mcpserver.NewServer(c).Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(stderr, "soulstream-mcp: %v\n", err)
		return 1
	}
	return 0
}

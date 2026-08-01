// Command node runs the prototype remote MCP node against a real deployment —
// the Bar 4 half of hq/01-RESEARCH/remote-mcp-node: expose it over HTTPS
// (tailscale serve) and point a hosted MCP client at it. Experiment code,
// not the product.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/nats-io/nats.go"

	researchnode "github.com/impire-io/soulidentity/researchnode"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to serve MCP on (put tailscale serve in front for HTTPS)")
	natsURL := flag.String("nats-url", nats.DefaultURL, "NATS server URL (the realm's deployment)")
	sentinel := flag.String("sentinel", "", "path to the operator-mode sentinel creds file (routes connections to auth callout)")
	realmName := flag.String("realm", "", "realm name (required)")
	publicURL := flag.String("public-url", "", "public HTTPS base URL of this node; enables the OAuth edge (RFC 9728 metadata + 401 challenges)")
	authIssuer := flag.String("auth-issuer", "", "OIDC issuer URL advertised in the resource metadata (the same issuer the callout validates)")
	flag.Parse()

	if *realmName == "" {
		log.Print("node: --realm is required")
		flag.Usage()
		os.Exit(2)
	}
	// --public-url alone = proxy-fronted with bearer-only auth; adding
	// --auth-issuer switches the OAuth discovery edge on too.

	n := researchnode.New(researchnode.Config{
		NATSURL:      *natsURL,
		Realm:        *realmName,
		SentinelPath: *sentinel,
		PublicURL:    *publicURL,
		AuthIssuer:   *authIssuer,
	})
	log.Printf("node: MCP on http://%s (realm %q, nats %s, oauth-edge %v)",
		*listen, *realmName, *natsURL, *publicURL != "")
	log.Fatal(http.ListenAndServe(*listen, n))
}

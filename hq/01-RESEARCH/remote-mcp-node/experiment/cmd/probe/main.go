// Command probe runs the pre-registered pass protocol against a LIVE node
// (initialize → tools/list → board → post_turn) and then verifies from the
// realm: attribution == principal, and SigVerified from one keys.public
// answer. On the BYON this is simultaneously the reversal-condition
// measurement — callout admission on the deployment class the realms
// actually live on. Experiment code, not product.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synadia-io/orbit.go/natscontext"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulstream/identity"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/topic"
)

type bearerRT struct{ token string }

func (rt bearerRT) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+rt.token)
	return http.DefaultTransport.RoundTrip(r)
}

func main() {
	nodeURL := flag.String("node", "http://127.0.0.1:8080", "node base URL")
	bearerFile := flag.String("bearer-file", "byon-secrets/claude-desktop.token", "file holding the bearer token")
	realmName := flag.String("realm", "proof", "realm name")
	ctxName := flag.String("context", "impire-dev-platform", "NATS context for the reader")
	account := flag.String("account", "", "app account public key (required)")
	asUser := flag.String("as-user", "daan", "the reader connection's own principal user")
	persona := flag.String("persona", "daan", "expected persona (the token's user)")
	flag.Parse()
	if *account == "" {
		log.Fatal("probe: --account is required")
	}

	raw, err := os.ReadFile(*bearerFile)
	fatal("read bearer", err)
	bearer := strings.TrimSpace(string(raw))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// --- The pass protocol, as a hosted client would run it.
	c := mcp.NewClient(&mcp.Implementation{Name: "byon-probe", Version: "0.0.1"}, nil)
	sess, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   *nodeURL,
		HTTPClient: &http.Client{Transport: bearerRT{token: bearer}},
	}, nil)
	fatal("initialize", err)
	defer func() { _ = sess.Close() }()
	log.Print("initialize: OK (session formed through callout admission)")

	tools, err := sess.ListTools(ctx, nil)
	fatal("tools/list", err)
	var names []string
	for _, t := range tools.Tools {
		names = append(names, t.Name)
	}
	log.Printf("tools/list: %v", names)

	who := callText(ctx, sess, "whoami", nil)
	log.Printf("whoami: %s", who)
	if !strings.HasPrefix(who, *persona+"@") {
		log.Fatalf("probe: whoami %q does not name persona %q", who, *persona)
	}

	_ = callText(ctx, sess, "board", nil)
	log.Print("board: OK")

	path := callText(ctx, sess, "start_topic", map[string]any{
		"name": "byon-proof", "subject_matter": "the reversal-condition measurement",
	})
	log.Printf("start_topic: %s", path)
	opID := callText(ctx, sess, "post_turn", map[string]any{
		"path": path, "body": "posted through the node, admitted by callout, signed in the vault — on the BYON",
	})
	log.Printf("post_turn: %s", opID)

	// --- Realm-side verification: the reader trusts nothing the node said.
	nc, _, err := natscontext.Connect(*ctxName)
	fatal("reader connect", err)
	defer nc.Close()
	rc, err := realm.NewClient(ctx, nc, realm.Config{Realm: *realmName})
	fatal("reader realm client", err)
	defer func() { _ = rc.Close() }()

	h := topic.Open(rc, path)
	bare, err := h.Materialise(ctx)
	fatal("materialise (no keyring)", err)
	if bare.Announcement == nil || bare.Announcement.Sig != topic.SigUnknownKey {
		log.Fatalf("probe: negative control failed: announcement sig %v", bare.Announcement)
	}
	log.Print("negative control: unknown-key without a keyring")

	pub, err := siclient.New(nc, *account, *asUser).PersonaPublicKey(*persona)
	fatal("keys.public", err)
	h.UseKeyring(&identity.Keyring{Keys: map[string][]string{*persona: {pub}}})
	mt, err := h.Materialise(ctx)
	fatal("materialise", err)
	if mt.Announcement == nil || mt.Announcement.Sig != topic.SigVerified {
		log.Fatalf("probe: announcement not verified: %+v", mt.Announcement)
	}
	for i, contrib := range mt.Contributions {
		if contrib.Author != *persona {
			log.Fatalf("probe: contribution %d authored %q, want %q", i, contrib.Author, *persona)
		}
		if contrib.Sig != topic.SigVerified {
			log.Fatalf("probe: contribution %d sig %s", i, contrib.Sig)
		}
	}
	fmt.Printf("\nPASS: %d op(s) on %s authored by %q, SigVerified from one keys.public answer — callout admission works on this deployment class.\n",
		len(mt.Contributions)+1, path, *persona)
}

func callText(ctx context.Context, sess *mcp.ClientSession, tool string, args map[string]any) string {
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	fatal("call "+tool, err)
	if res.IsError {
		log.Fatalf("probe: %s returned tool error: %+v", tool, res.Content)
	}
	txt, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		log.Fatalf("probe: %s: not text: %T", tool, res.Content[0])
	}
	return txt.Text
}

func fatal(what string, err error) {
	if err != nil {
		log.Fatalf("probe: %s: %v", what, err)
	}
}

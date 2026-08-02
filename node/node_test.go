package node_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream/node"
	"github.com/impire-io/soulstream/node/rigtest"
	"github.com/impire-io/soulstream/topic"
)

// serveNode stands the node in front of the rig on an httptest server and
// returns its URL. Local mode unless publicURL is set.
func serveNode(t *testing.T, r *rigtest.Rig, cfg node.Config) string {
	t.Helper()
	cfg.Listen = "127.0.0.1:0" // unused: httptest owns the listener
	cfg.Realm = rigtest.Realm
	cfg.NATSURL = r.URL
	cfg.SentinelPath = r.SentinelPath
	n, err := node.New(cfg)
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}
	t.Cleanup(n.Close)
	srv := httptest.NewServer(n.Handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

// mcpSession dials an MCP session at url presenting bearer on every request.
func mcpSession(t *testing.T, url, bearer string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "node-test-client", Version: "0.0.1"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: bearerRT{bearer: bearer}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	sess, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("mcp connect: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// bearerRT stamps Authorization on every request — the hosted client's
// freshest-bearer behavior.
type bearerRT struct{ bearer string }

func (b bearerRT) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", "Bearer "+b.bearer)
	return http.DefaultTransport.RoundTrip(r)
}

func callText(t *testing.T, sess *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %s", name, toolText(t, res))
	}
	return toolText(t, res)
}

func toolText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty tool result")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("result content is %T", res.Content[0])
	}
	return tc.Text
}

// TestUS1_ConnectAndParticipate: a scripted no-install client with a valid
// static token joins the realm, sees the full tool surface, and its first
// operation verifies as its persona on an independent reader (SC-001 gate
// half, static lane).
func TestUS1_ConnectAndParticipate(t *testing.T) {
	r := rigtest.Start(t, time.Minute)
	url := serveNode(t, r, node.Config{})
	token, _ := r.StaticToken(t, "daan-ext", "daan laptop")

	sess := mcpSession(t, url, token)

	// Tool-surface parity: the node serves exactly the mcpserver surface.
	tools, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := map[string]bool{}
	for _, tl := range tools.Tools {
		got[tl.Name] = true
	}
	for _, want := range []string{"soulstream_whoami", "soulstream_board", "soulstream_start_topic", "soulstream_post_turn"} {
		if !got[want] {
			t.Errorf("tool surface missing %q (have %d tools)", want, len(got))
		}
	}

	// whoami reports the SERVER-ASSERTED persona, not anything client-claimed.
	who := callText(t, sess, "soulstream_whoami", nil)
	if !strings.Contains(who, `"persona": "daan-ext"`) {
		t.Fatalf("whoami not the admitted persona: %s", who)
	}

	pathRes := callText(t, sess, "soulstream_start_topic", map[string]any{"name": "Through the door"})
	path := extractPath(t, pathRes)
	_ = callText(t, sess, "soulstream_post_turn", map[string]any{"path": path, "body": "posted through the remote node"})

	// Independent realm-side verification: a fresh reader, trusting nothing
	// the node said, sees the turn signed and verified as daan-ext.
	assertVerifiedAuthor(t, r, path, "daan-ext", "posted through the remote node")
}

// extractPath pulls the topic path from a start_topic result (JSON string or
// bare path).
func extractPath(t *testing.T, s string) string {
	t.Helper()
	s = strings.TrimSpace(s)
	var p string
	if err := json.Unmarshal([]byte(s), &p); err == nil && p != "" {
		return p
	}
	return s
}

// assertVerifiedAuthor reads the topic through an independent bootstrap
// reader wired with the realm's directory keyring and asserts the body's
// contribution is authored-and-verified as persona.
func assertVerifiedAuthor(t *testing.T, r *rigtest.Rig, path, persona, body string) {
	t.Helper()
	rc := r.ReaderRealm(t)
	kr := rigtest.DirectoryKeyring(t, r, persona)
	th := topic.Open(rc, path)
	th.UseKeyring(kr)
	v, err := th.Materialise(context.Background())
	if err != nil {
		t.Fatalf("reader materialise: %v", err)
	}
	found := false
	for _, c := range v.Contributions {
		if strings.Contains(c.Body, body) {
			found = true
			if c.Author != persona {
				t.Errorf("author = %q, want %q", c.Author, persona)
			}
			if c.Sig != topic.SigVerified {
				t.Errorf("sig status = %q, want verified — delegated signature must verify on an independent reader", c.Sig)
			}
		}
	}
	if !found {
		t.Fatalf("posted turn %q not found in reader's view", body)
	}
}

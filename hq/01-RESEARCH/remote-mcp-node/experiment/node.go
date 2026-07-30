// Package researchnode is the prototype remote MCP node for soulstream's
// hq/01-RESEARCH/remote-mcp-node investigation. It is the experiment, not the
// product: a Streamable-HTTP MCP server that validates nothing itself —
// the caller's bearer token is passed through to NATS, where SoulIdentity's
// auth callout admits or refuses the connection. One pooled connection per
// user; the node holds no credentials and no keys.
package researchnode

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/topic"
)

// Config is everything the node knows. Bar 1 pre-registered "a URL and a
// realm name and nothing credential-shaped"; whether the operator-mode
// sentinel file is needed on top is one of the experiment's questions — it
// routes a connection to callout and grants nothing by itself, but it IS a
// creds file on disk. The finding lands in JOURNEY.md either way.
type Config struct {
	NATSURL      string
	Realm        string
	SentinelPath string // empty = try the bare token option
}

// Node pools one NATS connection (and one MCP server) per authenticated user.
type Node struct {
	cfg     Config
	mu      sync.Mutex
	entries map[string]*entry
	inner   http.Handler
}

// entry is one user's pooled state. latest always holds the freshest bearer
// this user has presented; the NATS TokenHandler reads it on every
// (re)connect attempt, which is the whole re-proof mechanism.
type entry struct {
	latest  atomic.Pointer[string]
	build   sync.Once
	server  *mcp.Server
	nc      *nats.Conn
	persona string
	account string
	err     error
}

func New(cfg Config) *Node {
	n := &Node{cfg: cfg, entries: map[string]*entry{}}
	n.inner = mcp.NewStreamableHTTPHandler(n.serverFor, nil)
	return n
}

func bearerFrom(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if rest, ok := strings.CutPrefix(h, "Bearer "); ok {
		return rest
	}
	return ""
}

// routeHint keys the pool WITHOUT verifying anything: for a JWT-shaped token
// the unverified issuer+subject claims, for anything else the token string
// itself. This is routing only — a forged hint just creates a pool entry
// whose connection the callout refuses. Trust is decided at the NATS edge.
func routeHint(bearer string) string {
	if !strings.HasPrefix(bearer, "eyJ") {
		return "tok:" + bearer
	}
	parts := strings.Split(bearer, ".")
	if len(parts) != 3 {
		return "tok:" + bearer
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "tok:" + bearer
	}
	var claims struct {
		OID string `json:"oid"`
		Sub string `json:"sub"`
		Iss string `json:"iss"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "tok:" + bearer
	}
	id := claims.OID
	if id == "" {
		id = claims.Sub
	}
	return "oidc:" + claims.Iss + ":" + id
}

// ServeHTTP records the freshest bearer for the user's pool entry, then
// hands off to the streamable MCP handler. A real node would answer a
// missing/refused badge with 401 + WWW-Authenticate; the prototype leans on
// the handler's 400 for a nil server.
func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if b := bearerFrom(r); b != "" {
		n.mu.Lock()
		if e, ok := n.entries[routeHint(b)]; ok {
			e.latest.Store(&b)
		}
		n.mu.Unlock()
	}
	n.inner.ServeHTTP(w, r)
}

// serverFor is the StreamableHTTPHandler hook: new MCP session → this user's
// pooled server, building it (connect, admit, wire the seam) on first sight.
func (n *Node) serverFor(r *http.Request) *mcp.Server {
	b := bearerFrom(r)
	if b == "" {
		return nil
	}
	hint := routeHint(b)
	n.mu.Lock()
	e, ok := n.entries[hint]
	if !ok {
		e = &entry{}
		e.latest.Store(&b)
		n.entries[hint] = e
	}
	n.mu.Unlock()
	e.build.Do(func() { e.err = e.connect(n.cfg) })
	if e.err != nil {
		return nil
	}
	return e.server
}

// Err reports the build error for the pool entry a bearer routes to —
// experiment introspection only.
func (n *Node) Err(bearer string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if e, ok := n.entries[routeHint(bearer)]; ok {
		return e.err
	}
	return nil
}

func (e *entry) connect(cfg Config) error {
	opts := []nats.Option{
		nats.TokenHandler(func() string {
			if p := e.latest.Load(); p != nil {
				return *p
			}
			return ""
		}),
		nats.ReconnectWait(200 * time.Millisecond),
		nats.MaxReconnects(-1),
	}
	if cfg.SentinelPath != "" {
		opts = append(opts, nats.UserCredentials(cfg.SentinelPath))
	}
	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return fmt.Errorf("admission refused: %w", err)
	}
	persona, account, err := principalOf(nc)
	if err != nil {
		nc.Close()
		return err
	}
	// The seam: signing delegated to SoulIdentity on the USER'S OWN
	// connection — the key materialises in the vault on first touch and the
	// node never sees it. PersonaSigner satisfies realm.Config.Signer
	// structurally.
	signer, err := siclient.New(nc, account, persona).PersonaSigner(persona)
	if err != nil {
		nc.Close()
		return fmt.Errorf("persona signer: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rc, err := realm.NewClient(ctx, nc, realm.Config{Realm: cfg.Realm, Persona: persona, Signer: signer})
	if err != nil {
		nc.Close()
		return fmt.Errorf("realm client: %w", err)
	}
	e.nc, e.persona, e.account = nc, persona, account
	e.server = buildServer(rc, persona, account)
	return nil
}

// principalOf asks the server who this connection is: the user-info reply
// carries the RESOLVED permission set, and the expanded scope template names
// the principal in the sign.record grant — server-asserted, not client-claimed.
func principalOf(nc *nats.Conn) (persona, account string, err error) {
	msg, err := nc.Request("$SYS.REQ.USER.INFO", nil, 5*time.Second)
	if err != nil {
		return "", "", fmt.Errorf("user info: %w", err)
	}
	var resp struct {
		Data struct {
			User        string `json:"user"`
			Account     string `json:"account"`
			Permissions struct {
				Pub struct {
					Allow []string `json:"allow"`
				} `json:"publish"`
			} `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return "", "", fmt.Errorf("user info decode: %w", err)
	}
	for _, subj := range resp.Data.Permissions.Pub.Allow {
		parts := strings.Split(subj, ".")
		if len(parts) == 5 && parts[0] == siclient.Segment && parts[3] == "sign" && parts[4] == "record" {
			return parts[2], parts[1], nil
		}
	}
	return "", "", fmt.Errorf("user info: no sign.record grant to derive the principal from (allow=%v)", resp.Data.Permissions.Pub.Allow)
}

type startTopicArgs struct {
	Name          string `json:"name" jsonschema:"the topic's display name"`
	SubjectMatter string `json:"subject_matter" jsonschema:"what the topic is about"`
}

type postTurnArgs struct {
	Path string `json:"path" jsonschema:"the topic path"`
	Body string `json:"body" jsonschema:"the turn's body"`
}

type whoamiArgs struct{}

// buildServer exposes a deliberately minimal tool surface — enough to prove
// the bars (write, attribute, sign). The full mcpserver reuse question is an
// 018 implementation decision, not a research bar.
func buildServer(rc *realm.Client, persona, account string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "soulstream-node-prototype", Version: "0.0.1"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "whoami",
		Description: "The principal this session acts as.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ whoamiArgs) (*mcp.CallToolResult, any, error) {
		return textResult(persona + "@" + account), nil, nil
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_topic",
		Description: "Start a topic; returns its path.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in startTopicArgs) (*mcp.CallToolResult, any, error) {
		h, err := topic.StartTopic(ctx, rc, topic.StartTopicInput{Name: in.Name, SubjectMatter: in.SubjectMatter})
		if err != nil {
			return nil, nil, err
		}
		return textResult(h.Path()), nil, nil
	})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "post_turn",
		Description: "Post a turn to a topic; returns the op id.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in postTurnArgs) (*mcp.CallToolResult, any, error) {
		id, err := topic.Open(rc, in.Path).PostTurn(ctx, in.Body)
		if err != nil {
			return nil, nil, err
		}
		return textResult(id), nil, nil
	})
	return s
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

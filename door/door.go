// Package door is the product's forwarding half (hq design
// external-tools.md D41, episode 0118): the stdio tool door grown an
// outbound side. It reads the realm's tool catalog — the discovery face,
// where a door learns its targets — and re-exposes each target's tools
// beside the record's own, on one server, one contract.
//
// Its invariants are the design's, each carried in code rather than
// convention:
//
//   - Endpoints only, never a process: a workload tool is run by the room
//     and reached at its declared address; a remote tool is dialed where
//     the catalog says. The door spawns nothing.
//   - No outbound token at rest, in config, or in anything the agent
//     reads: authority is fetched per call on the agent's own admission —
//     its own grant, or its operator's through a delegation the agent was
//     handed — held in a local, and gone when the call returns.
//   - The remote sees the calling person: with a subject and delegation
//     declared, every fetch is grants.access on-behalf, so the token is
//     the operator's own, bounded by their standing consent.
//   - Refusals in words: a call the plane refuses comes back to the agent
//     as an error-marked tool result naming the refusal, and the refused
//     call never touches the remote.
//
// Tool names are prefixed with their catalog entry's name
// ("github_create_issue") — one door serves many targets, and whose tool
// is whose stays legible to the person reading an agent's transcript.
package door

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/toolcatalog"

	"github.com/impire-io/soulstream-identity/client"
)

// Config is the door's outbound identity, environment-shaped like the
// rest of the verb that hosts it.
type Config struct {
	// Account is the identity plane's account segment; empty derives it
	// from the connection's own server-asserted grants ($SYS.REQ.USER.INFO).
	Account string
	// Subject, when set, is the persona this door acts on behalf of — the
	// operator a personal agent answers to. Every remote fetch is then
	// grants.access on-behalf under the delegation below.
	Subject           string
	DelegationPayload string
	DelegationSig     string
}

// fetchFunc obtains the bearer one call to one target rides — "" for a
// target that takes none. The seam the tests stub; the real one is the
// identity plane.
type fetchFunc func(ctx context.Context, entry toolcatalog.Entry) (string, error)

// Attach reads the realm's tool catalog and registers one forwarding tool
// per target tool on srv. It returns notes — one line per entry it could
// not serve, honest degradation rather than a failed door: a catalog
// nobody wrote yet means a door with the record's tools alone, exactly as
// before this package existed.
func Attach(ctx context.Context, srv *mcp.Server, rc *realm.Client, cfg Config) []string {
	entries, warnings, err := toolcatalog.All(ctx, rc)
	var notes []string
	for _, w := range warnings {
		notes = append(notes, fmt.Sprintf("tool %s: unreadable catalog entry: %v", w.Name, w.Err))
	}
	if err != nil {
		return append(notes, fmt.Sprintf("tool catalog: %v", err))
	}
	if len(entries) == 0 {
		return notes
	}
	fetch, ferr := newFetcher(rc, cfg)
	if ferr != nil {
		return append(notes, fmt.Sprintf("outbound authority: %v", ferr))
	}
	return append(notes, attach(ctx, srv, entries, fetch)...)
}

// newFetcher builds the real per-call authority: the identity client on
// the door's own admission, own-grant or on-behalf by configuration.
func newFetcher(rc *realm.Client, cfg Config) (fetchFunc, error) {
	account := cfg.Account
	if account == "" {
		var err error
		if account, err = deriveAccount(rc); err != nil {
			return nil, fmt.Errorf("no SOULSTREAM_ACCOUNT and none derivable: %w", err)
		}
	}
	id := client.New(rc.Conn(), account, rc.Persona())
	del := client.Delegation{Payload: cfg.DelegationPayload, Sig: cfg.DelegationSig}
	return func(_ context.Context, entry toolcatalog.Entry) (string, error) {
		if entry.Kind != toolcatalog.KindRemote {
			return "", nil // in-deployment targets ride no bearer
		}
		if cfg.Subject != "" {
			access, err := id.GrantAccessOnBehalf(entry.Name, cfg.Subject, del)
			if err != nil {
				return "", err
			}
			return access.AccessToken, nil
		}
		access, err := id.GrantAccessToken(entry.Name)
		if err != nil {
			return "", err
		}
		return access.AccessToken, nil
	}, nil
}

// attach is the catalog walk, seam-side: discovery per entry, one
// forwarding tool per target tool.
func attach(ctx context.Context, srv *mcp.Server, entries []toolcatalog.Entry, fetch fetchFunc) []string {
	var notes []string
	for _, entry := range entries {
		switch entry.Kind {
		case toolcatalog.KindRemote, toolcatalog.KindWorkload:
		default:
			notes = append(notes, fmt.Sprintf("tool %s: kind %q is not one this door speaks", entry.Name, entry.Kind))
			continue
		}
		if entry.Endpoint == "" {
			notes = append(notes, fmt.Sprintf("tool %s: isn't serving (no endpoint declared)", entry.Name))
			continue
		}
		if strings.HasPrefix(entry.Name, "soulstream") {
			notes = append(notes, fmt.Sprintf("tool %s: the name would shadow the record's own tools", entry.Name))
			continue
		}
		tools, err := discover(ctx, entry, fetch)
		if err != nil {
			notes = append(notes, fmt.Sprintf("tool %s: isn't serving (%v)", entry.Name, err))
			continue
		}
		for _, tool := range tools {
			forward(srv, entry, tool, fetch)
		}
	}
	return notes
}

// discover lists one target's tools — authority fetched for the listing
// exactly as for a call, and gone the same way.
func discover(ctx context.Context, entry toolcatalog.Entry, fetch fetchFunc) ([]*mcp.Tool, error) {
	sess, err := dial(ctx, entry, fetch)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sess.Close() }()
	list, err := sess.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, err
	}
	return list.Tools, nil
}

// forward registers one forwarding tool: the target's own description and
// input schema under the prefixed name, and a handler that fetches
// authority per call, dials, forwards, and refuses in words.
func forward(srv *mcp.Server, entry toolcatalog.Entry, tool *mcp.Tool, fetch fetchFunc) {
	remoteName := tool.Name
	description := tool.Description
	if description == "" {
		description = "forwarded to " + entry.Name
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        entry.Name + "_" + remoteName,
		Description: description + " (via " + entry.Name + ")",
		InputSchema: tool.InputSchema,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, any, error) {
		refuse := func(err error) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%s refused this call: %v", entry.Name, err)},
			}}, nil, nil
		}
		sess, err := dial(ctx, entry, fetch)
		if err != nil {
			return refuse(err)
		}
		defer func() { _ = sess.Close() }()
		res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: remoteName, Arguments: in})
		if err != nil {
			return refuse(err)
		}
		return res, nil, nil
	})
}

// dial opens one MCP session to a target, authority fetched for exactly
// this session — the fetch is the refusal point, and a refused call never
// touches the target.
func dial(ctx context.Context, entry toolcatalog.Entry, fetch fetchFunc) (*mcp.ClientSession, error) {
	token, err := fetch(ctx, entry)
	if err != nil {
		return nil, err
	}
	httpc := http.DefaultClient
	if token != "" {
		httpc = &http.Client{Transport: bearer{tok: token}}
	}
	return mcp.NewClient(&mcp.Implementation{Name: "soulstream-door", Version: "0.0.1"}, nil).
		Connect(ctx, &mcp.StreamableClientTransport{
			Endpoint: entry.Endpoint, HTTPClient: httpc, MaxRetries: -1,
		}, nil)
}

// bearer injects one call's token; it lives for one dial.
type bearer struct{ tok string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.tok)
	return http.DefaultTransport.RoundTrip(req)
}

// deriveAccount reads the account segment off the connection's own
// server-asserted permissions — the expanded sign.record grant names it,
// so the door needs no configuration the transport already carries.
func deriveAccount(rc *realm.Client) (string, error) {
	msg, err := rc.Conn().Request("$SYS.REQ.USER.INFO", nil, 5_000_000_000)
	if err != nil {
		return "", err
	}
	var info struct {
		Data struct {
			Permissions struct {
				Publish struct {
					Allow []string `json:"allow"`
				} `json:"publish"`
			} `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		return "", err
	}
	for _, subj := range info.Data.Permissions.Publish.Allow {
		parts := strings.Split(subj, ".")
		if len(parts) == 5 && parts[0] == client.Segment && parts[3] == "sign" && parts[4] == "record" {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("no expanded sign.record grant on this connection")
}

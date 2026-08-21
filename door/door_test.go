package door

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream-core/toolcatalog"
)

// target is one stand-in MCP server: streamable HTTP, optionally behind a
// bearer, recording what authenticated each tools/call frame.
type target struct {
	mu      sync.Mutex
	bearers []string
	srv     *httptest.Server
}

type noteInput struct {
	Text string `json:"text" jsonschema:"the note to keep"`
}

func newTarget(t *testing.T, requireBearer func(string) bool) *target {
	t.Helper()
	tg := &target{}
	s := mcp.NewServer(&mcp.Implementation{Name: "target", Version: "0.0.1"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name: "keep_note", Description: "Keep a note.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in noteInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: "noted: " + in.Text},
		}}, nil, nil
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
	tg.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tok := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		if requireBearer != nil && !requireBearer(tok) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if req.Body != nil {
			var body strings.Builder
			if _, err := copyBody(&body, req); err == nil &&
				strings.Contains(body.String(), `"tools/call"`) {
				tg.mu.Lock()
				tg.bearers = append(tg.bearers, tok)
				tg.mu.Unlock()
			}
		}
		h.ServeHTTP(w, req)
	}))
	t.Cleanup(tg.srv.Close)
	return tg
}

// copyBody drains and restores a request body.
func copyBody(into *strings.Builder, req *http.Request) (int64, error) {
	raw, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	into.Write(raw)
	req.Body = io.NopCloser(strings.NewReader(string(raw)))
	req.ContentLength = int64(len(raw))
	return int64(len(raw)), err
}

// agent connects an in-memory MCP client to a server, the way a harness
// holds its stdio door.
func agent(ctx context.Context, t *testing.T, srv *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "0.0.1"}, nil).
		Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func textOf(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// The door forwards: a remote target's tools re-exposed under the entry's
// prefix, authority fetched per call and presented to the target, the
// result carried back whole.
func TestTheDoorForwardsWithPerCallAuthority(t *testing.T) {
	ctx := context.Background()
	remote := newTarget(t, func(tok string) bool { return strings.HasPrefix(tok, "at-") })
	local := newTarget(t, nil)

	var mu sync.Mutex
	fetches := 0
	fetch := func(_ context.Context, e toolcatalog.Entry) (string, error) {
		if e.Kind != toolcatalog.KindRemote {
			return "", nil
		}
		mu.Lock()
		defer mu.Unlock()
		fetches++
		return fmt.Sprintf("at-%d", fetches), nil
	}

	srv := mcp.NewServer(&mcp.Implementation{Name: "door", Version: "0.0.1"}, nil)
	notes := attach(ctx, srv, []toolcatalog.Entry{
		{Name: "github", Kind: toolcatalog.KindRemote, Endpoint: remote.srv.URL},
		{Name: "notes", Kind: toolcatalog.KindWorkload, Persona: "notes-tool", Endpoint: local.srv.URL},
	}, fetch)
	if len(notes) != 0 {
		t.Fatalf("healthy targets produced notes: %v", notes)
	}

	cs := agent(ctx, t, srv)
	list, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range list.Tools {
		names[tool.Name] = true
	}
	if !names["github_keep_note"] || !names["notes_keep_note"] {
		t.Fatalf("the targets' tools are not on the door: %v", names)
	}

	// Two calls to the remote: authority fetched per call — discovery took
	// one fetch, each call takes its own.
	for i, want := range []string{"one", "two"} {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "github_keep_note", Arguments: noteInput{Text: want},
		})
		if err != nil || res.IsError {
			t.Fatalf("call %d: %v %q", i, err, textOf(res))
		}
		if textOf(res) != "noted: "+want {
			t.Fatalf("call %d answered %q", i, textOf(res))
		}
	}
	mu.Lock()
	got := fetches
	mu.Unlock()
	if got != 3 { // discovery + two calls
		t.Fatalf("authority fetched %d times, want 3 (discovery + one per call)", got)
	}
	remote.mu.Lock()
	bearers := append([]string(nil), remote.bearers...)
	remote.mu.Unlock()
	if len(bearers) != 2 || bearers[0] == bearers[1] {
		t.Fatalf("the calls did not each present their own token: %v", bearers)
	}

	// The workload target rides no bearer at all.
	if res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "notes_keep_note", Arguments: noteInput{Text: "here"},
	}); err != nil || res.IsError || textOf(res) != "noted: here" {
		t.Fatalf("workload call: %v %q", err, textOf(res))
	}
	local.mu.Lock()
	defer local.mu.Unlock()
	if len(local.bearers) != 1 || local.bearers[0] != "" {
		t.Fatalf("the workload target was sent a bearer: %v", local.bearers)
	}
}

// A refused fetch refuses in words — an error-marked result naming the
// target — and the refused call never touches it.
func TestARefusedFetchNeverTouchesTheTarget(t *testing.T) {
	ctx := context.Background()
	remote := newTarget(t, func(string) bool { return true })

	healthy := true
	fetch := func(_ context.Context, _ toolcatalog.Entry) (string, error) {
		if healthy {
			return "at-discovery", nil
		}
		return "", errors.New("grants: no grant for this persona and resource")
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "door", Version: "0.0.1"}, nil)
	if notes := attach(ctx, srv, []toolcatalog.Entry{
		{Name: "github", Kind: toolcatalog.KindRemote, Endpoint: remote.srv.URL},
	}, fetch); len(notes) != 0 {
		t.Fatalf("notes: %v", notes)
	}
	healthy = false

	cs := agent(ctx, t, srv)
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "github_keep_note", Arguments: noteInput{Text: "never"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(res), "github refused this call") ||
		!strings.Contains(textOf(res), "no grant") {
		t.Fatalf("the refusal is not in words: isError=%v %q", res.IsError, textOf(res))
	}
	remote.mu.Lock()
	defer remote.mu.Unlock()
	if len(remote.bearers) != 0 {
		t.Fatal("a refused call reached the target")
	}
}

// Honest degradation, entry by entry: no endpoint, an unfamiliar kind, a
// dead target, a name that would shadow the record's own tools — each a
// note, none a failed door, and the healthy entries serve regardless.
func TestTheDoorDegradesEntryByEntry(t *testing.T) {
	ctx := context.Background()
	remote := newTarget(t, nil)

	fetch := func(_ context.Context, _ toolcatalog.Entry) (string, error) { return "", nil }
	srv := mcp.NewServer(&mcp.Implementation{Name: "door", Version: "0.0.1"}, nil)
	notes := attach(ctx, srv, []toolcatalog.Entry{
		{Name: "github", Kind: toolcatalog.KindRemote, Endpoint: remote.srv.URL},
		{Name: "declared-only", Kind: toolcatalog.KindRemote},
		{Name: "futuristic", Kind: toolcatalog.Kind("function"), Endpoint: "http://x"},
		{Name: "dead", Kind: toolcatalog.KindRemote, Endpoint: "http://127.0.0.1:1/mcp"},
		{Name: "soulstream-fake", Kind: toolcatalog.KindRemote, Endpoint: remote.srv.URL},
	}, fetch)
	for _, want := range []string{
		"declared-only: isn't serving (no endpoint declared)",
		`futuristic: kind "function" is not one this door speaks`,
		"dead: isn't serving",
		"soulstream-fake: the name would shadow",
	} {
		var found bool
		for _, n := range notes {
			if strings.Contains(n, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no note saying %q in %v", want, notes)
		}
	}
	cs := agent(ctx, t, srv)
	if res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "github_keep_note", Arguments: noteInput{Text: "still here"},
	}); err != nil || res.IsError {
		t.Fatalf("the healthy entry did not survive its neighbours: %v %q", err, textOf(res))
	}
}

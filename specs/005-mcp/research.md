# Research: MCP Adapter — Go SDK

**Feature**: 005-mcp | **Date**: 2026-07-12
**Method**: Verified against the module proxy + pkg.go.dev + release notes (July 2026).

## Decision: the official Go MCP SDK

- **`github.com/modelcontextprotocol/go-sdk` v1.6.1** (official, MCP project + Google), import
  `github.com/modelcontextprotocol/go-sdk/mcp`. Post-v1.0 → **non-breaking API guarantee**. Go ≥1.25
  (we have 1.26).
- **Rationale**: spec-compliant, minimal generics-based typed-tool API, stdio built in, stability
  guarantee. `mark3labs/mcp-go` is the community SDK that inspired it (more transports/middleware);
  the official SDK is the right choice for a small, stable stdio tool server.
- **Gotcha**: discard any pre-1.0 snippets (old `NewServerTool`, method-style `AddTool`) — the current
  form is the top-level generic below.

## Verified API

```go
server := mcp.NewServer(&mcp.Implementation{Name: "soulstream", Version: "0.1.0"}, nil)

mcp.AddTool(server, &mcp.Tool{Name: "soulstream_board", Description: "..."}, h.board)
// handler: func(ctx, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)
// In is a struct; its JSON schema is auto-inferred (jsonschema:"desc" tags for field docs).

// text result:
return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}, nil, nil
// a returned Go error becomes a TOOL error (IsError), not a transport crash.

server.Run(ctx, &mcp.StdioTransport{}) // blocks until the client disconnects
```

- Typed input via reflection over the `In` struct; no hand-written schema.
- `Out = any` (returning `nil`) omits the output schema — fine; our tools return JSON/text in Content.

## Decision: `topic.FetchInbox` (new library helper)

- MCP is request/response, so `check_inbox` needs a **bounded, newest-first** read of the notify
  backlog (not the streaming `FollowInbox`). Add `FetchInbox(ctx, c, persona, limit) ([]Notification,
  error)`: drain the notify subject (ordered consumer, `NumPending==0` stop, empty guard via
  `GetLastMsgForSubject`), reverse to newest-first, cap at `limit` (default 50).
- **Rationale**: belongs in the library beside `FollowInbox`; reuses the verified consumer pattern; no
  new mechanism.

## Testability

Handler methods take a typed input and the session `*realm.Client`; tests call them directly against
an in-process server — no stdio, no MCP client needed (FR-015). The MCP wiring (`AddTool`, `Run`) is
thin and exercised by `go build` + a live agent.

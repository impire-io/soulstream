# Implementation Plan: MCP Adapter for AI Personas

**Branch**: `005-mcp` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)

## Summary

A `soulstream-mcp` stdio server (`cmd/soulstream-mcp`) and a testable `internal/mcpserver` package
that exposes eight Soulstream tools over the official Go MCP SDK. The server connects once as a
configured persona and every tool call acts as it, over `realm` + `topic`. One new library helper —
`topic.FetchInbox` (bounded, newest-first) — backs the `check_inbox` tool. Tool handlers are plain
methods testable against an in-process server without stdio.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`; SDK needs Go ≥1.25).
**Primary Dependencies**: existing `realm`, `topic`; **new**: `github.com/modelcontextprotocol/go-sdk`
(v1.6.1, official, post-v1.0 stable) — import `github.com/modelcontextprotocol/go-sdk/mcp`.
**MCP API** (verified, research.md): `mcp.NewServer(&mcp.Implementation{Name,Version}, nil)`;
`mcp.AddTool[In,Out](s, &mcp.Tool{Name,Description}, handler)` where handler is
`func(ctx, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)` and In is a struct whose JSON
schema is auto-inferred (`jsonschema:"..."` tags for descriptions); text result via
`&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text:…}}}`; a returned Go error becomes a
tool error (`IsError`); `server.Run(ctx, &mcp.StdioTransport{})`.
**Storage**: none new (drives the provisioned realm + object store).
**Testing**: `go test`; `internal/mcpserver` handler methods tested directly against an embedded
server (`realm.NewClient` over `internal/natstest`), asserting result text/JSON and errors — no stdio,
no live MCP client (FR-015).
**Project Type**: single Go module; adds `cmd/soulstream-mcp` (thin) + `internal/mcpserver` + one
`topic.FetchInbox` helper.
**Constraints**: stdio transport; one persona per process; tool inputs are text (so text attachments).
**Scale/Scope**: 8 tools, ~a handful of files.

## Constitution Check

- **I. NATS-Native First** — ✅ PASS. The adapter is a pure client: all behaviour is `realm`/`topic`
  over NATS; notifications and attachments use the existing stream + object store. The MCP SDK is a
  client-facing protocol library (the agent's door), not soulstream infrastructure — it introduces no
  service beside NATS.
- **II. Smallest Viable Implementation** — ✅ PASS. Eight thin tool handlers mapping to existing
  library calls; one small `FetchInbox` helper; text attachments only; polling inbox (no streaming
  door). Excluded: SSE/HTTP, binary attachments, edit/reply/resolve, admin.
- **III. ELI5 Documentation** — ✅ PASS (planned). `docs/mcp.md`: "the same door, for agents — an
  agent calls a tool the way a human types a command", with the tool list and a config snippet.

**Result**: PASS. Re-checked post-design: unchanged. The one new dependency (official MCP SDK) is the
agent-facing protocol itself and is justified — it *is* the feature's interface, not a reinvention.

## Project Structure

```text
cmd/soulstream-mcp/
└── main.go               # resolve config (env/flags), realm.Connect, mcpserver.NewServer, Run(stdio)

internal/mcpserver/
├── server.go             # newHandlers(c); NewServer(c) registers the 8 tools; result helpers
├── tools.go              # the 8 handler methods + their typed input structs
├── server_test.go        # each handler against an embedded server (result + error matrix)

topic/
├── notify.go             # (extend) FetchInbox(ctx, c, persona, limit) []Notification  (bounded, newest-first)
└── notify_test.go        # (extend) FetchInbox returns newest-first, bounded, empty-safe

docs/mcp.md
```

**Structure Decision**: Logic in `internal/mcpserver` (handler methods on a struct holding the
`*realm.Client`) so it is unit-testable directly; `cmd/soulstream-mcp` is a thin `main`. `check_inbox`
needs a bounded, newest-first read of the notify backlog, which belongs in the library next to
`FollowInbox` — added as `topic.FetchInbox`.

## Key implementation notes

- **Attribution**: the server holds one persona-bound client; write tools post through it, so every op
  is authored by the configured persona by construction (FR-003).
- **Startup**: require a persona (the write door + inbox need one); fail fast on connect error (FR-004).
- **Results**: JSON text for reads (board, view, inbox), plain text for writes (op-id, path, object
  key). A handler error → a tool error via the SDK.
- **Post/comment/attach/close** materialise first (so parents/lifecycle are correct), mirroring the CLI.
- **FetchInbox**: drain the notify subject (ordered consumer, `NumPending==0` stop, empty guard),
  reverse to newest-first, cap at the limit (default 50).

## Complexity Tracking

> No Constitution violations. The MCP SDK dependency is the agent-facing protocol interface, justified
> as the feature itself.

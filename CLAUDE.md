<!-- SPECKIT START -->
Active feature: **005-mcp** — an MCP stdio adapter (`cmd/soulstream-mcp` + `internal/mcpserver`)
exposing 8 tools so an AI persona participates. Uses the official
`github.com/modelcontextprotocol/go-sdk` (v1.6.1): `mcp.AddTool[In,Out](s, &mcp.Tool{...}, handler)`.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/005-mcp/plan.md](specs/005-mcp/plan.md)
(spec: `specs/005-mcp/spec.md`). Done: `001`–`004` merged (library + human CLI).
Tool handlers are methods on a struct holding the session `*realm.Client`, tested directly against an
embedded server. One new library helper: `topic.FetchInbox` (bounded, newest-first).

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm` and `topic` are the NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices) separate from the
  NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

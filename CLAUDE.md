<!-- SPECKIT START -->
Active feature: **008-discover** — scatter/gather topic discovery. Plain core-NATS
request-reply (NO JetStream, NO queue group — every responder answers, the asker merges):
`topic.Discover` publishes a `topic.discover` record to `SOULSTREAM.SVC.DISCOVER` with a reply
inbox + deadline, gathers `topic.discover.reply` records via `SubscribeSync(inbox)` +
`NextMsg(remaining)` until deadline; silence ⇒ (nil, nil). `topic.RespondDiscovery` subscribes,
rebuilds `Board` per request, matches (pure `matchEntries`: case-insensitive substring over
name/subject-matter/tags, "" = all, limit clamped [1,50]), replies only when matches exist.
Merge (pure `mergeReplies`): key = path, one credit per (path, persona), first-seen fields win.
Signing: SVC records bind to the SERVICE NAME (`DISCOVER`) — replies too, never the `_INBOX.*`
subject; wire.go record build factored to take an explicit binding. `realm.Client.Conn()` added.
CLI: `discover` + long-running `respond`; MCP: `soulstream_discover` (11th tool, ask-only).
Note: the stream (`SOULSTREAM.>`) captures SVC requests — accepted clutter, documented in
contracts/wire.md.

For details read: [specs/008-discover/plan.md](specs/008-discover/plan.md)
(spec: `specs/008-discover/spec.md`, contracts: `specs/008-discover/contracts/`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup` merged.

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm`, `topic`, and `registry` are the
  NATS-touching packages.
- Keep pure projection logic (materialise/board fold, chain validation, discovery match/merge)
  separate from the NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

<!-- SPECKIT START -->
Active feature: **004-cli** — a human CLI (`cmd/soulstream` + testable `internal/cli`)
over the library: provision/board/start/show/watch/post/comment/attach/get/close/inbox.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/004-cli/plan.md](specs/004-cli/plan.md)
(spec: `specs/004-cli/spec.md`). Done: `001`–`003` merged (the library layer is complete).
CLI logic lives in `internal/cli` with an injectable `Run(ctx, args, stdout, stderr, connect)`
so it tests against an embedded server; `cmd/soulstream/main.go` is a thin wrapper.

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm` and `topic` are the NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices) separate from the
  NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

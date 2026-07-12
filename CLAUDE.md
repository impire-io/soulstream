<!-- SPECKIT START -->
Active feature: **003-participation** — mentions (@name → mention.notify inbox) and
attachments (object store put/get + attachment.add), extending the `topic` package.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/003-participation/plan.md](specs/003-participation/plan.md)
(spec: `specs/003-participation/spec.md`). Done: `001-foundation`, `002-topics` (merged).

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm` and `topic` are the NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices) separate from the
  NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

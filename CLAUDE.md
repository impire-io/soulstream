<!-- SPECKIT START -->
Active feature: **002-topics** — the op-log engine (announce/baseline/turn/comment/
lifecycle, materialisation, live follow, sub-topics, discovery board) on the foundation.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/002-topics/plan.md](specs/002-topics/plan.md)
(spec: `specs/002-topics/spec.md`). Done: `001-foundation` (merged).

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm` and `topic` are the NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices) separate from the
  NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

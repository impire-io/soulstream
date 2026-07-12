<!-- SPECKIT START -->
Active feature: **001-foundation** — the Soulstream wire layer (realm provisioning
and the operation record) as a Go module on NATS JetStream.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/001-foundation/plan.md](specs/001-foundation/plan.md)
(spec: `specs/001-foundation/spec.md`).

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm` is the only NATS-touching package.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (not legacy `nc.JetStream()`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

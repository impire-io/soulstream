<!-- SPECKIT START -->
Active feature: **006-signing** — op signing & key distribution. `Soulstream-Sig` becomes real:
Ed25519 (stdlib `crypto/ed25519`, no new deps) over `record.Record.Canonical` bytes computed with
`Signature` empty; signing hooks into the single publish choke point `topic/wire.go:publishOp`.
Minimal persona directory: `soulstream-personas` KV bucket (third provisioned artefact,
create-or-report), TOFU *chain* pinning client-side, rotation proof = old key over
`"soulstream-key-rotation\n"+persona+"\n"+newPubB64`. Per-op `SigStatus`
(unsigned/verified/failed/unknown-key) annotates read paths; flags, never drops.

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/006-signing/plan.md](specs/006-signing/plan.md)
(spec: `specs/006-signing/spec.md`, contracts: `specs/006-signing/contracts/`).
Done: `001`–`005` merged (library + human CLI + MCP adapter, MVP complete).
New this cycle: `registry` package (third NATS-touching package; pure chain validation
separated), `identity.SigningKey`/`Keyring` (identity stays NATS-free), `internal/keystore`
(seed + pins files shared by CLI and MCP).

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm`, `topic`, and `registry` are the
  NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices, registry chain
  validation) separate from the NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->

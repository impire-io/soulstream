<!-- SPECKIT START -->
Active cycle: **017-signer-seam** (branch `017-signer-seam`, planned
2026-07-29) — the Signer seam: `identity.Signer { PublicKey() string;
Sign(canonical []byte) (string, error) }` so signing can be delegated to an
external custodian (SoulIdentity's `sign.record` over NATS — its M2
"consumers wire in") without soulstream depending on it.
`(*SigningKey).Sign` becomes fallible (error always nil locally);
`realm.Config.Signer`/`Client.Signer()` take the interface (assign concrete
keys only when non-nil — typed-nil hazard); chokepoint `topic/wire.go:
buildOpMsg`: signer error or EMPTY signature = publish fails, no unsigned
fallback; responders (discover/memory) already turn a build error into
silence + `served(-1)` — signing failure joins that path. `registry.
NewAttestationToken` + `Rotate` accept the interface (capability, not
custody); keystore/keygen stay concrete `*SigningKey` (seeds never behind
the seam). No new deps; no config surface for delegation (arrives with the
remote node, 018-ish); docs/signing.md ELI5 section in the same change.

For details read: [specs/017-signer-seam/plan.md](specs/017-signer-seam/plan.md)
(spec: `specs/017-signer-seam/spec.md` incl. Clarifications 2026-07-29,
decisions: `research.md` R1–R7, contract: `contracts/library.md`, model:
`data-model.md`, consumer view: `quickstart.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work`, `011-vocab`, `012-distribution` (v0.1.0), `013-config` (v0.2.0),
`014-persona-accountability` (v0.3.0/v0.3.1), `015-memory` (v0.4.0, archivist
live on NGS + dogfood running since 2026-07-27), `016-provision-limits`
(v0.5.0) merged + pushed. Research: sealed-topics graduated to design
2026-07-28 (journey 0005 — speckit-ready, build priority gated on the
dogfood chafe log).

Project conventions:
- Go 1.26; module `github.com/impire-io/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->

## How this project is run (read this first)

The SPECKIT block above tracks the active feature; the durable way of working
lives in `hq/`. Before touching anything:

- **`hq/00-GENESIS/` first** — [`vision.md`](hq/00-GENESIS/vision.md),
  [`constitution.md`](hq/00-GENESIS/constitution.md) (articles + the anti-drift
  working agreement, wired into spec-kit via the
  `.specify/memory/constitution.md` symlink), and
  [`how-we-work.md`](hq/00-GENESIS/how-we-work.md). Decisions are held against
  these.
- **[`AGENTS.md`](AGENTS.md)** — the numbered reading order and the
  non-negotiables in brief.
- **The journey duty (required):** every landed feature, concluded research
  investigation, or load-bearing decision gets a numbered episode in
  `hq/04-JOURNEY/` in the same change — `/journey-log` does this (research topics
  get theirs via `/research-graduate`). The structure is enforced by
  `internal/hqlint` under `make test`.

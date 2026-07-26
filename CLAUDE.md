<!-- SPECKIT START -->
Active feature: **016-provision-limits** — provisioning byte limits: optional
per-artefact storage budgets so limit-enforced accounts (NGS R1, err 10113)
provision out of the box. `realm.Budgets{OpLog,Notify,Personas,Objects int64}`
(0 = unlimited; Notify 0 = keep mandated 64MiB) + `DefaultBudgets()` =
1GiB/64MiB/64MiB/512MiB (the proven manual-workaround shapes).
`ProvisionOn(ctx, js, budgets ...Budgets)` / `Client.Provision` — variadic,
source-compatible, >1 value or negative field = error BEFORE server contact.
Budgets apply ONLY at creation; create-or-report inviolate; `ArtefactResult`
gains `MaxBytes` (as-applied for created, AS FOUND otherwise, read from
backing stream configs incl. `KV_`/`OBJ_` streams). CLI: `provision
[--budgets] [--budget-{oplog,notify,personas,objects} SIZE]` — SIZE takes
KiB/MiB/GiB (binary only), explicit 0/negative rejected at parse; switch
composes with flags (flags overwrite fields; flags alone = rest unlimited).
Tests: natstest variant with account `MaxBytesRequired: true` reproduces the
R1 refusal locally — both US1 scenarios [measured]. No budgets in
.soulstream.json (identity only). docs/provisioning.md ELI5 section ships in
the same change. Legacy-shape convergence path untouched.

For details read: [specs/016-provision-limits/plan.md](specs/016-provision-limits/plan.md)
(spec: `specs/016-provision-limits/spec.md`, decisions: `research.md` D1–D6,
contract: `contracts/library.md`, model: `data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work`, `011-vocab`, `012-distribution` (v0.1.0), `013-config` (v0.2.0),
`014-persona-accountability` (v0.3.0/v0.3.1), `015-memory` (v0.4.0, archivist
live on NGS + dogfood running since 2026-07-27) merged + pushed.

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

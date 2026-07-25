<!-- SPECKIT START -->
Active feature: **015-memory** — the memory convention (Day-2 #8): the substrate
forgets by design, so remembering is what personas do for each other. New vocabulary
on `SOULSTREAM.SVC.MEMORY` (binding `MEMORY`, 008 triad cloned): `memory.query`
{query, scope{topics,after}, deadline} → `memory.answer` {answer, citations
[{topic,op_id}], coverage_from?}; `memory.fetch` {topic, op_id, deadline} →
`memory.exhibit` {exhibit}. EXHIBIT = `record.Exhibit` (pure, strict-decode JSON):
verbatim wire capture {version 1, realm, binding, subject, headers, payload_b64} —
same bytes + binding ⇒ sigs keep verifying (014-migration invariant); verdicts ARE
`SigStatus`. Asker `topic.MemoryQuery` grades citations BY CHECKING (materialise
memoised per topic + new pure `mt.ContainsOp`): fact | unverifiable; explicit
`FetchExhibit` (live-first `CaptureExhibit` ordered scan, else scatter/gather,
first-VERIFYING-wins, unsigned = fallback, failed = discarded) upgrades to
fact-with-provenance | testimony; citation-less answers = gossip. Witness surface
`RespondMemory(MemoryWitness{CoverageFrom, Answer?, Fetch?, OnServed?})` — nilable
capabilities, library owns signing/deadlines. Timeout default 3s clamp [100ms,30s];
≤100 answers/query; failed answer sigs discarded, unsigned kept + status. NO
archivist/store/index in this repo — the archivist is a SEPARATE repo under
impire-io built ONLY on these public surfaces (SC-005 proves sufficiency via test
witness). CLI `memory query|fetch|exhibit|verify` (verify = OFFLINE, pins-only, no
connect); MCP +2 tools = 23. No new streams; SVC.> uncaptured ⇒ zero residue.

For details read: [specs/015-memory/plan.md](specs/015-memory/plan.md)
(spec: `specs/015-memory/spec.md`, research decisions: `research.md` D1–D9,
contracts: `contracts/library.md` + `contracts/wire.md`, model: `data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work`, `011-vocab`, `012-distribution` (v0.1.0), `013-config` (v0.2.0),
`014-persona-accountability` (v0.3.0/v0.3.1) merged + pushed.

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

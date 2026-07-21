# Implementation Plan: The Curator Persona

**Branch**: `009-curator` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-curator/spec.md`

## Summary

A new top-level `curator` package (NATS-touching, built **only on public library
surfaces** — that constraint is the extension's central claim made structural) plus a
long-running CLI mode. The projection is a cache of `Materialise` views with
dirty-tracking: build once from `Board`, mark topics dirty via one core subscription
on `SOULSTREAM.TOPICS.>`, re-materialise lazily on next use — warm without duplicating
the fold. Discovery answering reuses 008 via a small refactor:
`topic.RespondDiscoveryWith(ctx, c, answer, onServed)` (the 008 board-backed responder
becomes a wrapper). Judgment is pure and serverless-tested: token-Jaccard duplicate
scoring over announcement metadata, a dormancy rule that excludes curator suggestions
(recognised by a stable `[curator]` body convention — the log itself is the
idempotence memory), and content-aware search over cached views. Suggestions are
ordinary `AddComment` calls anchored to the topic's frontier. One tiny additive
library change: `MaterializedTopic.BaselineTs` (the fold records the baseline op's
timestamp) so announcement-only topics have a birth time for the idle window. CLI:
`soulstream curate [--idle 336h] [--scan-every 1m]`. No MCP changes, no new op types,
no storage, zero new dependencies.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`)
**Primary Dependencies**: existing only; the curator consumes `realm`, `topic`,
`identity`, `record` public APIs plus one core-NATS subscription for liveness signals
**Storage**: none — the projection is in-memory, rebuilt on start by replay
**Testing**: pure judgment tests serverless; habit tests on the embedded server with
short idle windows (tens of ms) and scan ticks
**Target Platform**: unchanged
**Project Type**: library package + CLI mode
**Performance Goals**: answer latency = cache lookup (no per-request replay);
refresh cost = re-materialise of changed topics only
**Constraints**: zero curators ⇒ byte-identical realm behaviour (SC-004); suggestions
are ordinary comments only; idempotence derives from the log, never local state
**Scale/Scope**: dogfood — tens of topics, one or two curators

## Constitution Check

- **I. NATS-Native First** — PASS. No new infrastructure: the projection is a client
  cache over the same replayable messages, liveness is one core subscription,
  answers ride 008's request-reply. Nothing to run, register, or keep consistent.
- **II. Smallest Viable Implementation** — PASS. Digests cut (rationale in spec);
  similarity is one deterministic token-overlap rule; no persistence, no warm-start
  cache, no scoring/ranking; the only core-library change is one additive field and
  one additive function variant.
- **III. ELI5 Documentation** — PASS. New `docs/curator.md` (the librarian who knows
  every shelf and leaves polite sticky notes but never moves your books);
  `discovery.md`'s curator paragraph gets its link; `cli.md` updated in-story.

## Project Structure

### Documentation (this feature)

```text
specs/009-curator/
├── plan.md · research.md · data-model.md · quickstart.md
├── contracts/library.md
└── tasks.md (Phase 2)
```

### Source Code (repository root)

```text
topic/
├── view.go              # additive: MaterializedTopic.BaselineTs (fold records it,
│                        # baked path carries it through rollup via the baseline op)
└── discover.go          # RespondDiscoveryWith(ctx, c, answer, onServed);
                         # RespondDiscovery delegates with the board-backed answer

curator/                 # NEW package — public-surface-only consumer of the library
├── doc.go
├── suggest.go           # the [curator] body convention: build + recognise both kinds (pure)
├── judge.go             # similarity scoring, dormancy rule, content search (pure)
├── projection.go        # Board+Materialise cache, dirty-marking core subscription
├── curator.go           # Run(ctx, c, Options): answer + scan loop + suggestion posting
└── (tests: judge_test.go serverless; curator_test.go embedded)

internal/cli/
├── cli.go / curate_cmd.go  # `curate` long-running command (like respond)
└── curate_cmd_test.go

docs/
├── curator.md           # NEW (ELI5)
└── (updates: discovery.md link, cli.md, README.md index + root README)
```

**Structure Decision**: `curator` is a separate top-level package, deliberately NOT
inside `topic`: the design's whole point is that curation is built on the same
public surfaces available to any persona, and the package boundary enforces it (the
compiler proves FR-001's "zero protocol standing"). Pure judgment in `judge.go` /
`suggest.go` keeps the serverless-test convention.

## Complexity Tracking

No violations. Judgment call: the projection re-materialises a dirty topic wholesale
rather than folding incrementally — replay of one topic is cheap at dogfood scale and
reuses the tested pure fold instead of duplicating it in a second code path.

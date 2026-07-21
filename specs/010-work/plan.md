# Implementation Plan: Work Stages 1–2 — Versioned Artefacts & Work Items

**Branch**: `010-work` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-work/spec.md`

## Summary

Two additive vocabularies over the existing op-log (extensions/work.md stages 1–2):

1. **Versioned artefacts** — *zero new op types*. A revision is an `attachment.add`
   whose anchor points at a prior attachment op; the projection groups attachments
   into lineages (union by anchor connectivity, identity = root op-ID) and derives
   the tip as the lineage member latest in stream order. Artefacts are **computed
   from the existing view** (`MaterializedTopic.Artefacts()` — a pure function over
   `mt.Attachments`), so the fold, the baked payload, and the wire are untouched by
   stage 1. `Handle.Revise` is a thin wrapper over `Attach` with a required
   predecessor.
2. **Work items** — four new op types (`work.open`, `work.claim`, `work.done`,
   `work.abandon`) folded into a new `MaterializedTopic.WorkItems` field by a pure
   state machine: open → claimed (first claim in stream order wins; the fold's
   in-order traversal *is* the arbiter) → done, claimed → open on abandon; invalid
   transitions fold as void timeline events. Work items are baked into baselines
   (additive `BakedState.WorkItems`) so state survives compaction; work ops count
   as content ops (activate a proposed topic) and as real activity for the curator.

Surfaces: `Handle.OpenWork/ClaimWork/CompleteWork/AbandonWork` + `Handle.Revise` +
artefact derivation/fetch in the library; `work …`, `artefacts`, `revise`, and
`get --artefact` in the CLI; seven new MCP tools (text-bounded where MCP already is).
No new subjects, no storage shape changes, no signing changes (the publishOp choke
point signs new vocabulary automatically).

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`)
**Primary Dependencies**: nats.go v1.52 + `nats.go/jetstream`, orbit `natscontext`,
`gowebpki/jcs`, `google/uuid`, `modelcontextprotocol/go-sdk` v1.6.1 (all existing —
no new dependencies)
**Storage**: JetStream only — existing stream `SOULSTREAM.TOPICS.OPS.>` and object
bucket `soulstream-objects` (revisions reuse `attachments/<path>/<uuid>` keys)
**Testing**: `go test` with server-free fold tests + embedded `nats-server/v2` via
`internal/natstest.StartJetStream(t)`; gate `make check`
**Target Platform**: darwin/linux CLI + MCP stdio server (unchanged)
**Project Type**: Go library + two thin clients (unchanged)
**Performance Goals**: none new — artefact derivation is O(attachments) per
materialise; work fold is O(ops), same class as existing fold
**Constraints**: additive-only (FR-018): pre-010 baselines must parse, pre-010
readers must survive work ops (they already do — unknown types warn + skip);
projection must be pure/deterministic (unit-testable with no server)
**Scale/Scope**: dogfood realm scale (tens of topics, hundreds of ops per topic)

**Minimum NATS server version**: no capability beyond what 001–009 already use
(2.12+ per constitution; nothing new relied upon).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First** — PASS. No new infrastructure, subjects, buckets, or
  server features. Stage 1 is a projection rule over existing messages; stage 2 is
  new vocabulary on the existing ops subject. Claim races are decided by stream
  order — JetStream's total order *is* the lock service. Evaluated and rejected:
  per-item KV entries for claims (duplicates a NATS capability the stream already
  provides, and would split item state from the log of record).
- **II. Smallest Viable Implementation** — PASS. Stage 1 adds zero op types and
  zero fold state (artefacts are derived on demand). Stage 2 adds exactly the four
  ops the design names; no `work.reopen`, no claim timeout (deferred with `dormant`
  automation), no priorities/labels/assignees-by-request — none needed by an
  acceptance scenario. Payloads are minimal (title/body/anchor). The state machine
  is author-agnostic, so no authorization machinery.
- **III. ELI5 Documentation** — PASS (planned). Two new pages ship in the same
  change: `docs/artefacts.md` (a drawer of dated drawings — the newest is on top,
  none are thrown away) and `docs/work-items.md` (a chore chart — first hand on the
  magnet gets the chore). `docs/attachments.md`, `docs/cli.md`, `docs/mcp.md`
  updated where they enumerate surfaces.

*Post-design re-check (after Phase 1): still PASS — the contract adds no surface
beyond the FRs; the only new persisted shape is the additive `BakedState.WorkItems`
field, justified by FR-014 (state must survive compaction).*

## Project Structure

### Documentation (this feature)

```text
specs/010-work/
├── spec.md              # Feature specification (done)
├── plan.md              # This file
├── research.md          # Phase 0: decisions + rejected alternatives
├── data-model.md        # Phase 1: entities, payloads, state machine, baking
├── quickstart.md        # Phase 1: two-persona walkthrough (revise + claim race)
├── contracts/
│   └── library.md       # Phase 1: exported Go surface, CLI verbs, MCP tools
├── checklists/
│   └── requirements.md  # Spec quality checklist (done)
└── tasks.md             # Phase 2 (/speckit-tasks — not created by plan)
```

### Source Code (repository root)

```text
topic/
├── vocab.go             # + TypeWorkOpen/TypeWorkClaim/TypeWorkDone/TypeWorkAbandon
├── work.go              # NEW: work payloads, WorkItem/WorkEvent/WorkStatus,
│                        #      Handle.OpenWork/ClaimWork/CompleteWork/AbandonWork
├── artefact.go          # NEW: Artefact struct, MaterializedTopic.Artefacts(),
│                        #      FindArtefact resolver, Handle.Revise, GetRevision
├── view.go              # fold: work-op cases, WorkItems field, content-op count,
│                        #      baked seeding, sig annotation for items/events
├── rollup.go            # BakedState.WorkItems + cleanBakedWorkItems
├── work_fold_test.go    # NEW: server-free state-machine + race + void tests
├── artefact_test.go     # NEW: server-free lineage/tip/ambiguity tests
├── rollup_fold_test.go  # extend fullLog() with work ops + revisions (round-trip)
└── work_integration_test.go  # NEW: two-client race over embedded server

curator/
└── (verify lastReal counts work ops — FR-019; adjust + test if it scans typed fields)

internal/cli/
├── cli.go               # dispatch: work, artefacts, revise; get --artefact
├── work_commands.go     # NEW: work open/claim/done/abandon/list/show
└── artefact_commands.go # NEW: artefacts [ref], revise, get --artefact resolution

internal/mcpserver/
├── server.go            # register 7 new tools (total 18)
└── work_tools.go        # NEW: open/claim/complete/abandon work, revise_text,
│                        #      list_artefacts, read_artefact

docs/
├── artefacts.md         # NEW (ELI5)
├── work-items.md        # NEW (ELI5)
├── attachments.md       # note: anchor-to-attachment now means "revision"
├── cli.md, mcp.md       # surface lists updated
└── README.md            # index updated
```

**Structure Decision**: everything lands in existing packages. Stage 1 lives beside
the attachment code it extends (`topic/artefact.go`); stage 2 gets its own file pair
(`topic/work.go` write-side, fold cases in `view.go` beside the existing switch).
No new packages: unlike 009 (whose package boundary proved a point), this feature
*is* core vocabulary — the topic package is its home.

## Key design decisions (detail in research.md)

1. **Artefacts are derived, not stored.** `Artefacts()` computes lineages from
   `mt.Attachments` on demand: parent = anchor if the anchor resolves to another
   attachment's op-ID, else the attachment is a root. Union-find by connectivity;
   identity = root op-ID; tip = member with the highest position in the
   attachments slice (fold order = stream order, and baked entries precede tail
   entries in original order, so slice order *is* stream order — survives
   compaction for free). Zero fold/baking changes for stage 1; FR-006 holds by
   construction.
2. **The fold is the arbiter.** `apply` walks ops in stream order; the first
   well-formed `work.claim` that finds its item open simply wins. There is no
   separate race-resolution code to get wrong — determinism falls out of the
   existing traversal, exactly like lifecycle folding today.
3. **Malformed ≠ void.** Unreadable payload / missing item anchor / empty title →
   `Warnings` + skip (existing malformed-op convention). Readable ops that lose to
   the state machine (unknown item, claim on claimed, duplicate done, abandon on
   open) → void `WorkEvent` on the item's timeline (or a warning when there is no
   item to attach to), with `Void: true` — visible, effect-free. Matches the
   spec's Clarifications exactly.
4. **Work items bake like everything else.** `BakedState.WorkItems` (additive,
   `omitempty`) carries folded items — including timelines with void events — with
   volatile fields stripped (StreamSeq/Sig, recursively), seeded as seen+referenced
   so frontier and anchor resolution stay correct. Old baselines lack the field →
   zero items seeded → identical to today. Round-trip equality extends the
   established `viewsEqual` tests.
5. **Claim verdict UX.** Publishing a claim cannot know it won; CLI `work claim`
   and MCP `soulstream_claim_work` publish, then materialise, then report
   "claimed" or "void — owned by <persona>". The losing persona learns
   immediately without any extra mechanism.
6. **MCP stays text-bounded.** `soulstream_read_artefact` returns UTF-8 content
   only (errors pointing at the CLI for binary), consistent with
   `soulstream_attach_text`; binary-over-MCP remains deferred as before.

## Complexity Tracking

No constitution violations to justify.

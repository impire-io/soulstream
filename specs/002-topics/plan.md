# Implementation Plan: Topics — the Op-Log Engine

**Branch**: `002-topics` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/002-topics/spec.md`

## Summary

Add a `topic` package on top of the foundation that turns records into topics: start a topic
(`topic.announce` to INFO + initial `baseline` to OPS), post ops through a **topic handle** that
stamps author/op-id/timestamp and fills parents from the observed frontier, materialise a topic by
draining its ops backlog in **stream-sequence order** (DAG recorded, not consulted), follow it live
through the same ordered consumer with no replay/live seam, derive lifecycle
(`proposed`/`active`/`closed`) from the log, nest sub-topics by subject depth, and build a discovery
board by replaying `TOPICS.INFO.>`. Mentions and attachments are the next cycle; rollup, edit,
reply/resolve, dormant, archived, scatter/gather, and eg-walker are day-2.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire/soulstream`, established in 001).
**Primary Dependencies**: the foundation packages (`record`, `identity`, `realm`) and
`github.com/nats-io/nats.go/jetstream` (publish, ordered consumers). No new third-party deps.
**Key JetStream mechanisms** (verified in research.md):
- **Publish**: `js.PublishMsg(ctx, *nats.Msg)` → `*PubAck{Sequence, Duplicate}`, headers from `record.Build`.
- **Replay + live**: one **ordered consumer** filtered to the topic's exact ops subject,
  `DeliverAllPolicy`, consumed via the `Messages()` iterator; per-message `Metadata()` gives
  `Sequence.Stream` (the ordering key) and `NumPending` (backlog remaining).
- **Cold materialise**: drain until `NumPending == 0`, then stop; empty-subject guard via
  `stream.GetLastMsgForSubject` (returns `ErrMsgNotFound`) so an op-less path never blocks; **follow**:
  keep the same iterator open (a ctx-watcher calls `it.Stop()` to cancel).
- **Board**: `stream.Info(ctx, jetstream.WithSubjectFilter("SOULSTREAM.TOPICS.INFO.>"))` enumerates
  the info subjects (`State.Subjects`), then `GetLastMsgForSubject` fetches each latest announcement
  directly — O(topics) fetches, no full-history scan.
**Storage**: JetStream only (the provisioned `SOULSTREAM` stream). No new artefacts.
**Testing**: `go test`; unit tests for pure materialisation/board projection (feed synthetic records,
no server) plus integration tests for publish/replay/follow/discovery against `internal/natstest`.
**Project Type**: single Go library; adds one package `topic`.
**Performance Goals**: not performance-critical; MVP topics are short (full-log replay). No budget
beyond "materialise a short topic instantly".
**Constraints**: ordering is stream sequence only; baselines inline (reject oversize, FR-028); no
rollup so logs are replayed whole (roadmap's throwaway-realm assumption).
**Scale/Scope**: one new package, ~8 source files; the vocabulary is five op types.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First** — ✅ PASS. Ordering is JetStream's free total order (stream sequence);
  replay and live-follow are one **ordered consumer** (a built-in primitive), not custom plumbing;
  discovery is subject replay; idempotent lifecycle needs no coordinator. No database, no election,
  no consensus. `Nats-Expected-Last-Subject-Sequence`/rollup are noted for day-2 and unused here.
  Minimum server 2.12+ (inherited).
- **II. Smallest Viable Implementation** — ✅ PASS. Five op types, one ordered-consumer code path
  shared by materialise and follow, lifecycle *derived* not stored, sub-topics = subject depth (zero
  new machinery). Explicit non-goals (FR-029) keep mentions/attachments/rollup/edit/merge out. No
  speculative options.
- **III. ELI5 Documentation** — ✅ PASS (planned). New `docs/` pages in plain words with analogies:
  *a topic* (a shared workbench / a group notebook), *materialising* (reading the notebook front to
  back to see where things stand), *the frontier* (the pen's current position), *lifecycle*
  (proposed/active/closed as a project's life), *sub-topics* (a sticky-note thread), and *discovery*
  (the notice board). One per story.

**Result**: PASS — no violations; Complexity Tracking not required.

### Post-design re-check

Re-evaluated after Phase 1: still PASS. The design adds one package and no dependency; the ordered
consumer is the single NATS primitive doing replay+live, which is exactly the constitution's
"evaluate current server features before custom code".

## Project Structure

### Documentation (this feature)

```text
specs/002-topics/
├── plan.md · research.md · data-model.md · quickstart.md
├── contracts/topics.md
└── tasks.md   (/speckit-tasks)
```

### Source Code (repository root)

```text
topic/                          # the op-log engine — NATS-touching
├── doc.go
├── subjects.go                 # subject builders; topic-path helpers; slug/suffix id generation
├── vocab.go                    # op type + lifecycle constants; payload structs (announce, baseline, turn, comment, transition)
├── start.go                    # StartTopic (announce + initial baseline), Open
├── handle.go                   # Handle: Post/PostTurn/AddComment/Transition; frontier tracking; attribution guard
├── materialise.go              # ordered-consumer drain → MaterializedTopic; lifecycle derivation; dangling/malformed handling
├── follow.go                   # Follow: shared ordered consumer, replay then live
├── board.go                    # Board: replay TOPICS.INFO.> → []BoardEntry
├── view.go                     # MaterializedTopic, Contribution, Announcement, BoardEntry types + pure apply()
├── subjects_test.go · vocab_test.go
├── materialise_test.go         # pure apply() over synthetic records (no server)
├── board_test.go               # pure projection (no server) + integration
├── start_test.go · handle_test.go · follow_test.go  # integration (natstest)
└── ...

docs/
├── topic.md · materialisation.md · lifecycle.md · sub-topics.md · discovery.md
```

**Structure Decision**: One new package `topic`. The **pure** projection logic (`view.go`'s `apply`
that folds a sequence of records into a `MaterializedTopic`, and the board projection) is separated
from the **NATS** parts (publish, ordered consumer) so materialisation and board rules are
unit-tested by feeding synthetic `record.Record` slices with no server (fast, deterministic),
mirroring 001's record/realm split. Integration tests cover the wire path via `internal/natstest`.

### Key implementation notes

- **One ordered consumer** serves both cold materialise (drain until `NumPending==0`) and live
  follow (keep consuming). This makes FR-017's "no seam" structural rather than hand-managed.
- **Materialise = replay-then-fold**: read records in stream order, fold via the same pure `apply`
  used in unit tests; the handle caches the resulting frontier for its next Post.
- **Frontier** = observed op-ids − referenced parents; recomputed by `apply`.
- **Ordering**: `Metadata().Sequence.Stream` is the sort/append key; `Soulstream-Parents` is stored
  on each contribution's source record but never used to reorder (FR-012).

## Complexity Tracking

> No Constitution violations. This section intentionally left empty.

# Tasks: Work Stages 1–2 — Versioned Artefacts & Work Items

**Input**: Design documents from `/specs/010-work/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/library.md, quickstart.md

**Tests**: Included — the spec's success criteria all say "proven in tests", and the
project gate is all-green with none skipped. Pure fold/derivation logic gets
server-free unit tests; NATS-touching paths get embedded-server integration tests.

**Organization**: By user story. US1 (artefacts) and US2 (work items) are mutually
independent; US3 (abandon) extends US2's machinery.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

No setup tasks — existing module, zero new dependencies, no scaffolding to create.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: the shared vocabulary both work-item stories build on.

- [x] T001 Add `TypeWorkOpen`/`TypeWorkClaim`/`TypeWorkDone`/`TypeWorkAbandon`
      constants in `topic/vocab.go` and create `topic/work.go` with
      `WorkOpenPayload` and `WorkRefPayload` (reusing the existing `Anchor`
      struct) per data-model.md — payloads only, no behaviour yet.

**Checkpoint**: vocabulary compiles; US1 and US2 can proceed in parallel.

---

## Phase 3: User Story 1 - A document that remembers (Priority: P1) 🎯 MVP

**Goal**: whole-file revisions grouped into artefacts with a deterministic tip and
complete history, derived from existing attachments — zero new op types.

**Independent Test**: attach, revise twice from two personas, materialise cold: one
artefact, three revisions in order, tip = third; fetch tip and first revision bytes;
survive a rollup unchanged.

### Implementation for User Story 1

- [x] T002 [P] [US1] Create `topic/artefact.go`: `Artefact` struct,
      `(*MaterializedTopic).Artefacts()` derivation (anchor-connectivity lineages,
      root identity, tip = highest attachments-slice index), `FindArtefact`
      resolver with `ErrAmbiguousArtefact`, and `(*Handle).Revise` (thin `Attach`
      wrapper, predecessor required) per contracts/library.md.
- [x] T003 [P] [US1] Server-free unit tests in `topic/artefact_test.go`: lineage
      grouping; mid-chain anchor joins the root's lineage; anchor to non-attachment
      op or dangling anchor ⇒ standalone root; concurrent revisions of one tip ⇒
      stream-order winner; rename via revision changes artefact name; resolver by
      root id / member id / unique name; ambiguous name error lists roots.
- [x] T004 [US1] Extend `fullLog()` in `topic/rollup_fold_test.go` with a revision
      chain and add round-trip assertions: `Artefacts()` of `apply(rollup(L)+tail)`
      equals `Artefacts()` of `apply(L+tail)` (SC-002, SC-003, FR-006).
- [x] T005 [US1] Integration test in `topic/artefact_integration_test.go` (embedded
      server): two personas attach + revise, cold `Materialise`, fetch tip and old
      revision via `GetAttachment` + `VerifyDigest` (acceptance scenarios 1–3, 5).
- [x] T006 [US1] CLI in `internal/cli/artefact_commands.go` + dispatch in
      `internal/cli/cli.go`: `artefacts <topic> [ref]`, `revise <topic> <file>
      --of <ref> [--content-type]`, `get <topic> --artefact <ref> [--revision]
      [-o]` (positional `get` form unchanged); tests in
      `internal/cli/artefact_commands_test.go`.
- [x] T007 [US1] MCP tools in `internal/mcpserver/work_tools.go` + registration in
      `server.go`: `soulstream_revise_text`, `soulstream_list_artefacts`,
      `soulstream_read_artefact` (UTF-8 only, binary ⇒ error naming the CLI);
      tests in `internal/mcpserver/work_tools_test.go`.
- [x] T008 [US1] ELI5 docs: new `docs/artefacts.md` (drawer of dated drawings —
      newest on top, none thrown away); note in `docs/attachments.md` that an
      anchor to an attachment now means "revision"; surface lists in
      `docs/cli.md`, `docs/mcp.md`, `docs/README.md`.

**Checkpoint**: stage 1 fully usable and demoable on its own.

---

## Phase 4: User Story 2 - Claiming work without a lock service (Priority: P2)

**Goal**: work items with open/claim/done, first-claim-in-stream-order-wins, void
trail, compaction-safe.

**Independent Test**: open an item, race two claims, materialise anywhere: first
claimant owns it, second claim void on the timeline; done is terminal; state
survives rollup.

### Implementation for User Story 2

- [x] T009 [US2] Fold in `topic/view.go`: `WorkStatus`/`WorkEvent`/`WorkItem`
      types (in `topic/work.go`), `MaterializedTopic.WorkItems`
      (`work_items,omitempty`), cases for open/claim/done per data-model.md state
      table (malformed ⇒ warning+skip; readable-but-losing ⇒ void timeline event;
      unknown item ⇒ warning), work ops count as content ops (proposed→active),
      op-IDs into seen/referenced, sig annotation for items and events.
- [x] T010 [P] [US2] Server-free unit tests in `topic/work_fold_test.go`: claim
      race (first wins, second void), open→done without claim, done terminal
      (late claim/done void), claim by current owner void, empty-title open
      malformed, unreadable payload malformed, unknown item warning,
      work.open activates a proposed topic, comments/attachments anchored to an
      item resolve non-dangling, a log with no work ops serialises with no
      `work_items` field (FR-018), and a `life.transition` (close) leaves every
      item's status/owner untouched (FR-019).
- [x] T011 [US2] Baking in `topic/rollup.go`: `BakedState.WorkItems`,
      `cleanBakedWorkItems` (strip StreamSeq/Sig on items and events, keep void
      flags), fold seeding (items + timeline op-IDs seen/referenced, baked sig
      inheritance); extend `fullLog()` with work ops incl. a void claim; round-trip
      equality (SC-001, SC-003, FR-014); old-baseline (no field) compatibility test.
- [x] T012 [US2] Write side in `topic/work.go`: `(*Handle).OpenWork` (parses
      @mentions in body, fires `mention.notify`), `ClaimWork`, `CompleteWork`;
      integration test in `topic/work_integration_test.go` (embedded server): two
      clients race claims back-to-back, both materialise to the same owner + void
      set; mention notification observed; archived topic refuses work ops; a
      *closed* topic still accepts them (with the existing warning) and closing
      changes no item state; signed clients yield verified sig status on items,
      timeline events, and revisions — unsigned clients yield unsigned (FR-015).
- [x] T013 [US2] Curator: verify `lastReal` in `curator/` counts work ops as real
      activity (FR-019); if it scans typed view fields, include work items/events;
      add a regression test either way in `curator/`.
- [x] T014 [US2] CLI in `internal/cli/work_commands.go` + dispatch: `work open`
      (`--body`), `work claim` (publish → materialise → "claimed" / "void — owned
      by <p>"), `work done`, `work list`, `work show` (timeline incl. void +
      anchored evidence); tests in `internal/cli/work_commands_test.go`.
- [x] T015 [US2] MCP tools + registration: `soulstream_open_work`,
      `soulstream_claim_work` (returns verdict), `soulstream_complete_work`;
      tests alongside T007's file.
- [x] T016 [US2] ELI5 docs: new `docs/work-items.md` (chore chart — first hand on
      the magnet gets the chore; the list remembers who tried); surface lists in
      `docs/cli.md`, `docs/mcp.md`, `docs/README.md`.

**Checkpoint**: US1 and US2 independently functional.

---

## Phase 5: User Story 3 - Letting go, picking up (Priority: P3)

**Goal**: abandon reopens an item for a fresh first-claim race.

**Independent Test**: claim, abandon, verify open+unowned with the abandoned span
in the trail; new claim (any persona, including the previous owner) wins fresh;
abandon on a never-claimed item is void.

### Implementation for User Story 3

- [x] T017 [US3] Abandon: fold case in `topic/view.go` (claimed→open, owner
      cleared; abandon on open/done ⇒ void) + `(*Handle).AbandonWork` in
      `topic/work.go`; unit tests in `topic/work_fold_test.go` (reopen resets the
      race, previous owner may reclaim, void on open) and an integration
      claim→abandon→reclaim leg in `topic/work_integration_test.go`; extend the
      baked round-trip with an abandoned span.
- [x] T018 [US3] Clients + docs: CLI `work abandon` in
      `internal/cli/work_commands.go`; MCP `soulstream_abandon_work`; "letting go"
      section in `docs/work-items.md`; tests beside T014/T015's.

**Checkpoint**: all three stories functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T019 Walk quickstart.md end-to-end against a live embedded/local server
      (CLI paths), reconcile any drift in quickstart.md or command output; sweep
      README.md and `docs/README.md` for the new surfaces; confirm `CLAUDE.md`
      speckit block still matches reality; full gate `make check` — all green,
      none skipped (SC-004, SC-005).

---

## Dependencies & Execution Order

- **T001** blocks T009 (fold needs payloads) and T012 (write side needs types) —
  strictly, only US2/US3 need it; US1 has no dependency on T001.
- **US1 (T002–T008)**: T002 → T003/T004/T005 (tests need the code) → T006/T007
  (clients) → T008 (docs). T002+T003 may be written together (same PR-sized step).
- **US2 (T009–T016)**: T009 → T010/T011/T012 → T013/T014/T015 → T016.
- **US3 (T017–T018)**: after US2's T009/T012/T014/T015.
- **T019** last.

### Parallel Opportunities

- US1 (T002…) and US2 (T009…) touch disjoint files after T001 and can proceed in
  parallel; in a single-agent run, finish US1 first (MVP), then US2, then US3.
- T003/T004/T005 are independent test files; T006 (CLI) and T007 (MCP) are
  independent of each other; likewise T014/T015.

## Implementation Strategy

MVP = Phase 2 + Phase 3 (US1): versioned artefacts alone are shippable and
immediately useful in the dogfood realm. Then US2 as the second increment, US3 as
the third, polish last. Commit per phase or logical group (signed), gate
`make check` before every commit.

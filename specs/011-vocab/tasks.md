# Tasks: Remaining Vocabulary — Edit, Replies, Resolve, Removal, Dormant

**Input**: Design documents from `/specs/011-vocab/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/library.md, quickstart.md

**Tests**: Included — every success criterion says "proven in tests"; pure fold and
rule logic server-free, NATS paths on the embedded server. Gate: `make check`.

**Organization**: US1 (conversation upkeep) and US2 (removal + GC) are independent
after the shared vocabulary; US3 (dormant + sweeps) touches lifecycle + curator.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

No setup tasks — existing module, zero new dependencies.

---

## Phase 2: Foundational (Blocking Prerequisites)

- [ ] T001 Add `TypeCommentReply`/`TypeCommentResolve`/`TypeEdit`/
      `TypeAttachmentRemove` constants and `RefPayload` in `topic/vocab.go`;
      add `Dormant` to the lifecycle constants (fold/write behaviour comes in
      the story phases). Extend `topic/vocab_test.go` payload round-trips.

**Checkpoint**: vocabulary compiles; the three stories can proceed.

---

## Phase 3: User Story 1 - Second thoughts, same conversation (Priority: P1) 🎯 MVP

**Goal**: reply threads, same-author edits with compaction-proof chains, resolve
marks — identical on every replica, before and after rollup.

**Independent Test**: post → edit ×2 → reply → resolve; foreign edit warns and
changes nothing; cold view identical; post-rollup edit of a compacted chain
member still supersedes.

### Implementation for User Story 1

- [ ] T002 [US1] Fold in `topic/view.go`: `EditStamp`,
      `Contribution.{Resolved,ResolvedBy,Edits}`; `comment.reply` case (content
      op, threaded like comments); `edit` case per data-model rules (editTarget
      map over turn/comment/reply + stamp op-ids, same-author check, body/
      mentions rewrite, stamp append, malformed/foreign/unknown warnings);
      `comment.resolve` case (mark, first-resolver attribution, silent duplicate,
      non-comment warning); baked-stamp re-seeding into editTarget +
      seen/referenced; sig annotation for stamps in `topic/verify.go`.
- [ ] T003 [P] [US1] Server-free fold tests in `topic/upkeep_fold_test.go`:
      reply threads + dangling; edit renders latest (chained via original AND via
      prior edit; concurrent edits converge by stream order); foreign edit
      warning + no effect; empty-body/unreadable edit malformed; edit of
      attachment/work/unknown op warns; resolve marks + duplicate no-op +
      non-comment warning; mentions updated by edit; a vocabulary-free log
      serialises with no new fields (FR-016).
- [ ] T004 [US1] Baking: `cleanBakedContributions` deep-copies `Edits` and zeroes
      stamp volatiles in `topic/rollup.go`; extend `fullLog()` +
      `stripVolatile()` in `topic/rollup_fold_test.go` with an edited turn, a
      reply, and a resolved comment; round-trip equality incl. a **post-rollup
      edit anchoring a compacted edit op-id** (FR-003, SC-001).
- [ ] T005 [US1] Write side in `topic/upkeep.go`: `Handle.Reply` (mentions),
      `Handle.Edit` (empty-body refusal, mentions), `Handle.Resolve`;
      integration test in `topic/upkeep_integration_test.go`: two clients,
      edit/reply/resolve round trip, mention notification from a reply and an
      edit, signed edit stamps verify (FR-014), archived topic refuses the ops.
- [ ] T006 [US1] CLI `reply`/`edit`/`resolve` in
      `internal/cli/upkeep_commands.go` + dispatch/usage in `cli.go`; render
      `(edited by …)` / `resolved by …` markers in `render.go`; tests in
      `internal/cli/upkeep_commands_test.go`.
- [ ] T007 [US1] MCP `soulstream_reply_comment` / `soulstream_resolve_comment` /
      `soulstream_edit` (best-effort same-author pre-check with a clear error)
      in `internal/mcpserver/upkeep_tools.go` + registration; tests in
      `internal/mcpserver/upkeep_tools_test.go`.
- [ ] T008 [US1] ELI5 docs: new `docs/editing.md` (pencil edits & margin notes —
      crossed out, never torn out; your words are yours); `docs/mentions.md`
      note (edits and replies ping too); surface lists `docs/cli.md`,
      `docs/mcp.md`, `docs/README.md`.

**Checkpoint**: conversation upkeep fully usable.

---

## Phase 4: User Story 2 - Withdrawing a file, reclaiming at archival (Priority: P2)

**Goal**: attachment.remove marks; artefact tips fall back; archival deletes
withdrawn blobs only.

**Independent Test**: attach → revise → detach tip: artefact tip falls back,
entry marked removed, bytes fetchable; archive: removed blob gone, survivors
intact; all identical across rollup.

### Implementation for User Story 2

- [ ] T009 [US2] Fold + derivation: `Attachment.{Removed,RemovedBy}` and the
      `attachment.remove` case in `topic/view.go` (mark, silent duplicate,
      unknown/non-attachment warning, content op); `Artefacts()` in
      `topic/artefact.go` skips removed members for tips and drops fully-removed
      lineages; fold tests in `topic/upkeep_fold_test.go` + artefact tests in
      `topic/artefact_test.go` (tip fallback, all-removed lineage, baked marks).
- [ ] T010 [US2] Write side + GC: `Handle.RemoveAttachment` in `topic/upkeep.go`;
      `Archive` in `topic/lifecycle.go` deletes removed attachments' objects
      after the final compaction (best-effort, from the final view; the
      half-done-archival retry path GCs when it finally lands); integration
      test: detach → fetchable → archive → removed blob deleted (store returns
      not-found), surviving blob fetchable (SC-002); `get --artefact` of a
      removed-tip artefact serves the fallback tip.
- [ ] T011 [US2] Clients + docs: CLI `detach` + removed markers in listings
      (`render.go`, `artefact_commands.go` history/list); docs:
      `docs/attachments.md` (withdrawing a file; reclaimed at archival),
      `docs/artefacts.md` (tips skip withdrawn versions); CLI tests.

**Checkpoint**: US1 and US2 independent and done.

---

## Phase 5: User Story 3 - Topics that nap, claims that lapse (Priority: P3)

**Goal**: dormant as a real state with in-fold reactivation; pure idle rules;
manual command; two opt-in curator sweeps.

**Independent Test**: idle topic marked dormant by hand and by flag; content op
reactivates (cold + live); closed/archived ignore dormant; stale claim abandoned
by sweep and reclaimed fresh; concurrent sweeps converge.

### Implementation for User Story 3

- [ ] T012 [US3] Lifecycle: `definedLifecycle` accepts `Dormant`
      (`topic/lifecycle.go`), fold accepts proposed/active→dormant and ignores
      it from closed/archived with a warning, any content op while dormant flips
      Active in-loop (`topic/view.go`); `Handle.MarkDormant`; fold tests
      (transition validity, reactivation by each content-op kind, baked dormant
      + tail content, dormant never blocks close/archive) in
      `topic/upkeep_fold_test.go`; one live-path assertion: a `Follow`er of a
      dormant topic sees Active after a content op lands (SC-003).
- [ ] T013 [P] [US3] Pure rules in `topic/upkeep.go`: `DormantEligible`
      (newest-op-of-any-kind clock incl. edit stamps and work events; only
      proposed/active eligible) and `StaleClaims` (claim/timeline/anchored-
      evidence clock, claimed items only); table tests in
      `topic/upkeep_rules_test.go` with pinned `now`.
- [ ] T014 [US3] Sweeps: `curator.Options.{MarkDormant,ReclaimAfter}` +
      `cachedTopic.lastAny` + the two passes on the scan tick
      (`curator/curator.go`, `curator/projection.go`); curator tests: flag off ⇒
      no transitions ever (009 contract), flag on ⇒ idle topic marked dormant,
      stale claim abandoned and reclaimable, concurrent sweeps converge
      (SC-003); CLI `mark-dormant` command + `curate --mark-dormant --reclaim`
      flags (`internal/cli/upkeep_commands.go`, `curate_cmd.go`) with tests.
- [ ] T015 [US3] Docs: `docs/lifecycle.md` (dormant = napping, any content op
      wakes it), `docs/work-items.md` (lapsed magnets), `docs/curator.md` (two
      opt-in sweeps — housekeeping, not judgment), `docs/cli.md`.

**Checkpoint**: all three stories functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T016 Walk quickstart.md against the real CLI and reconcile drift; sweep
      root `README.md` (feature list, package table, tool count 21) and
      `docs/README.md`; confirm CLAUDE.md block; full `make check` — all green,
      none skipped (SC-004, SC-005).

---

## Dependencies & Execution Order

- T001 blocks everything. Within US1: T002 → T003/T004/T005 → T006/T007 → T008.
- US2 (T009–T011) after T001; T009 → T010 → T011.
- US3: T012 → T013 (parallel-ok) → T014 → T015. T014 needs T013 and 010's
  abandon (already merged).
- T016 last.

## Implementation Strategy

MVP = T001 + US1. Then US2, US3, polish. Single feature commit per house style
if phases interleave, otherwise per-story commits; gate before every commit.

# Tasks: Topics — the Op-Log Engine

**Input**: Design documents from `specs/002-topics/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED (spec success criteria are test matrices; constitution mandates all-green).

**Organization**: By the five user stories. New package `topic`. Module `github.com/impire/soulstream`.

---

## Phase 1: Setup

- [X] T001 Create the `topic` package skeleton: `topic/doc.go` with the package overview (op-log
  engine on record/identity/realm; pure fold separated from NATS I/O).

**Checkpoint**: `go build ./...` succeeds.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The pure core shared by every story — subjects, vocabulary, and the projection fold —
all unit-testable with no server.

- [X] T002 [P] `topic/subjects.go`: `OpsSubject(path)`, `InfoSubject(path)`, `TopicPath` helpers
  (parent/child join, split), and `NewTopicID(name string) string` = slug(name)+"-"+4 random `[a-z0-9]`
  (result satisfies `identity.ValidName`). (FR-006/023)
- [X] T003 [P] `topic/subjects_test.go`: subject builders for top-level and nested paths; `NewTopicID`
  produces valid slugs with a 4-char suffix, unique across many calls, from messy display names.
- [X] T004 [P] `topic/vocab.go`: op-type + lifecycle constants; typed payload structs with JSON
  (Announce, Baseline{State,Frontier}, Turn{Body}, Comment{Body,Anchor{Kind,OpID}}, Transition{To,From})
  plus marshal/unmarshal helpers. (FR-008/009/010)
- [X] T005 [P] `topic/vocab_test.go`: payload round-trip (marshal→unmarshal) for each type; anchor shape.
- [X] T006 `topic/view.go`: the view types (`MaterializedTopic`, `Contribution`, `Announcement`,
  `BoardEntry`, `Lifecycle`) and the **pure fold** `apply(records []record.Record) *MaterializedTopic`:
  order by stream seq (caller supplies in order), require baseline first (else `Malformed`), extract
  turns/comments, anchor comments (flag `Dangling` when the anchor op-id is absent), ignore unknown
  types (collect a warning), derive lifecycle (proposed/active/closed), compute frontier (observed −
  referenced parents). (FR-011..016, FR-019)
- [X] T007 `topic/view_test.go`: pure fold over synthetic `record.Record` slices — ordering; baseline
  required; turns+comments extracted; comment anchored; dangling flagged; unknown type ignored with
  warning; lifecycle proposed→active→closed; frontier = leaves; malformed when first op isn't baseline.
  (SC-002/003/005/007)
- [X] T008 `topic/handle.go`: `Handle` struct (client, path, observed frontier), `Open(c, path)`,
  `Path()`, and internal frontier update from a materialised view. (No posting yet — that's US2.)

**Checkpoint**: `go test ./topic/...` green (pure tests only); no server needed yet.

---

## Phase 3: User Story 1 — Start a topic and discover it (Priority: P1) 🎯 MVP

**Goal**: Announce a topic (info + initial baseline) and list it on the board.

**Independent Test**: StartTopic against an embedded server; Board shows it with metadata and
`proposed`; the ops subject's first message is the baseline.

- [ ] T009 [US1] `topic/start.go`: `StartTopicInput`, `StartTopic(ctx, c, in)` — generate id, publish
  `topic.announce` to INFO and the initial inline `baseline` to OPS (baseline first), reject oversize
  inline state (FR-028), return a `*Handle`; `Open` already exists. Honour `Parent` to nest the path
  (full sub-topic behaviour tested in US5). (FR-004/005/006/007/028)
- [ ] T010 [US1] `topic/board.go`: `Board(ctx, c)` via `stream.Info(WithSubjectFilter("…INFO.>"))` +
  `GetLastMsgForSubject` per subject → `[]BoardEntry` (one per topic; empty realm → empty). Parent
  relationship + `ParentKnown` (full sub-topic view in US5). (FR-025/027)
- [ ] T011 [US1] Integration test `topic/start_test.go`: StartTopic on an embedded server; assert INFO
  has the announce, OPS first message is the baseline, and the returned handle path is a valid slug;
  and a StartTopic whose inline state exceeds the threshold is rejected with a clear error pointing to
  the deferred manifest-baseline capability. (FR-028)
- [ ] T012 [US1] Integration test `topic/board_test.go`: start several topics; Board lists each once with
  correct metadata and `proposed`; empty realm yields an empty board. (SC-001)
- [ ] T013 [P] [US1] ELI5 doc `docs/topic.md`: a topic as "a shared workbench / a group notebook with
  a cover page (the announcement) and pages (the ops)". (Constitution III)
- [ ] T014 [P] [US1] ELI5 doc `docs/discovery.md`: the board as "the notice board in the hallway —
  one card per topic, showing its name and status". (Constitution III)

**Checkpoint**: A topic can be started and discovered; docs shipped.

---

## Phase 4: User Story 2 — Hold a conversation and materialise it (Priority: P1)

**Goal**: Post turns/comments and materialise the topic identically for everyone.

**Independent Test**: Post baseline+turns+comment; replay+materialise; contributions in stream order,
comment anchored, lifecycle `active`.

- [ ] T015 [US2] `topic/post.go`: `Handle.Post(ctx, type, payload, anchor)` — build a record (author,
  op-id, ts, parents=observed frontier), enforce attribution via the client, `PublishMsg` to OPS,
  advance the local frontier; plus `PostTurn`, `AddComment`. Warn (not block) if known-closed. (FR-001/002/003/022)
- [ ] T016 [US2] `topic/materialise.go`: `Handle.Materialise(ctx)` — empty guard via
  `GetLastMsgForSubject` (ErrMsgNotFound → empty/malformed view), else drain an ordered consumer
  (FilterSubjects=[opsSubject], DeliverAll) via `Messages()`/`Next()` until `NumPending==0`, parse each
  record, call the pure `apply`, cache the frontier on the handle. (FR-011/012/014/015)
- [ ] T017 [US2] Integration test `topic/materialise_test.go`: post baseline+turns+comment; materialise;
  assert stream-order contributions, comment `Anchor`==turn id, lifecycle `active`; materialise twice →
  identical view (SC-002); dangling comment flagged (SC-003).
- [ ] T018 [P] [US2] ELI5 doc `docs/materialisation.md`: materialising as "reading the notebook front to
  back to see where things stand right now". (Constitution III)

**Checkpoint**: Conversation posts and materialises deterministically.

---

## Phase 5: User Story 3 — Follow a topic live (Priority: P1)

**Goal**: Receive new ops live and update the view incrementally, no re-replay, no seam.

**Independent Test**: Follow a topic; from another connection post a turn; the follower's view updates.

- [ ] T019 [US3] `topic/follow.go`: `Handle.Follow(ctx, onOp)` — one ordered consumer (DeliverAll)
  via `Messages()`; apply history then keep applying live ops, advancing the frontier and calling
  `onOp` with the updated view after each; a goroutine calls `it.Stop()` on `ctx.Done()`. (FR-017/018)
- [ ] T020 [US3] Integration test `topic/follow_test.go`: open a follower; from a second connection post
  a turn; assert the follower's view gains the turn without a full re-replay and its frontier advances;
  no gap/duplicate at the replay/live seam. (SC-004)

**Checkpoint**: Live following works seam-free.

---

## Phase 6: User Story 4 — Lifecycle (Priority: P2)

**Goal**: Transition proposed→active→closed; derived state; idempotent concurrent close.

**Independent Test**: Start (proposed) → turn (active) → transition closed; materialise after each;
two concurrent closes converge.

- [ ] T021 [US4] `topic/lifecycle.go`: `Handle.Transition(ctx, to)` — reject a `to` the MVP does not
  define (naming allowed states), else post a `life.transition`. (Derivation already in `apply`.) (FR-020/021)
- [ ] T022 [US4] Integration test `topic/lifecycle_test.go`: proposed→active→closed derived correctly;
  two concurrent `closed` transitions converge to `closed` (idempotent); invalid transition rejected;
  posting a content op to a closed topic warns (surfaced), not blocked. (SC-005, FR-021/022)
- [ ] T023 [P] [US4] ELI5 doc `docs/lifecycle.md`: proposed/active/closed as "a project's life —
  suggested, being worked on, wrapped up — and how we can tell just by reading the notebook". (Constitution III)

**Checkpoint**: Lifecycle derives and transitions correctly.

---

## Phase 7: User Story 5 — Sub-topics (Priority: P2)

**Goal**: Nest a sub-topic under a topic; independent materialise; board shows the parent.

**Independent Test**: Announce a sub-topic with parent; its subjects are nested; it materialises
independently; the board shows the parent relationship.

- [ ] T024 [US5] Integration test `topic/subtopic_test.go`: StartTopic with `Parent` set; assert the
  ops/info subjects are `…<parent>.<child>`; materialise the sub-topic independently; deep nesting
  needs no code change. (SC-006, FR-023/024)
- [ ] T025 [US5] Extend `topic/board.go`: populate `BoardEntry.Parent`/`ParentKnown` from the path;
  flag (don't drop) a sub-topic whose parent path is absent. (FR-026)
- [ ] T026 [US5] Integration test in `topic/board_test.go`: a parent with sub-topics — each discoverable,
  parent relationship visible, unknown-parent flagged.
- [ ] T027 [P] [US5] ELI5 doc `docs/sub-topics.md`: a sub-topic as "a sticky-note thread clipped inside
  a page, kept separate but findable under its parent". (Constitution III)

**Checkpoint**: Sub-topics nest, materialise independently, and show on the board.

---

## Phase 8: Polish & Cross-Cutting

- [ ] T028 [P] Update the module `README.md` with the `topic` package (start/converse/follow/close/
  sub-topics/board) and link the 002 spec.
- [ ] T029 [US*] Quickstart walkthrough test `topic/quickstart_test.go`: mirror
  `specs/002-topics/quickstart.md` end to end against the embedded server (start → converse → close →
  sub-topic → board), guarding public signatures from drift.
- [ ] T030 Run `go mod tidy`; `go vet ./...` clean.
- [ ] T031 Final gate: `make check` green — fmt, build, all tests (none skipped), lint 0 issues. (SC-007)

---

## Dependencies & Execution Order

- **Setup (T001)** → none.
- **Foundational (T002–T008)** → after setup; **blocks all stories** (subjects, vocab, pure fold, handle).
- **US1 (T009–T014)** → after Foundational. **MVP.**
- **US2 (T015–T018)** → after Foundational (uses handle + apply). Independent of US1 except both use `topic`.
- **US3 (T019–T020)** → after US2 (Follow shares the consumer/apply path).
- **US4 (T021–T023)** → after US2 (Transition is a poster; derivation is in the foundational apply).
- **US5 (T024–T027)** → after US1 (extends start/board).
- **Polish (T028–T031)** → last.

### Parallel opportunities

- Foundational: T002/T003 (subjects) ∥ T004/T005 (vocab); then T006/T007 (view) which they feed.
- Docs (T013, T014, T018, T023, T027) are all [P].
- After Foundational, US2 (materialise track) and US1 (discovery track) touch mostly different files
  and can proceed in parallel; US3/US4 follow US2, US5 follows US1.

## Implementation Strategy

1. Setup + Foundational → the pure engine (fold/subjects/vocab), unit-tested with no server.
2. US1 → start + discover (MVP; validate against embedded server).
3. US2 → converse + materialise. 4. US3 → follow live. 5. US4 → lifecycle. 6. US5 → sub-topics.
7. Polish → README, quickstart test, tidy, green gate.

## Notes

- Commit after each story (signed); `make check` before each commit.
- `topic` may import `record`/`identity`/`realm`/`jetstream`; keep the pure `apply`/board fold free of
  NATS so it unit-tests without a server.
- Do not start deferred scope: mentions, attachments (003), or rollup/edit/reply/resolve/dormant/
  archived/scatter-gather/eg-walker (day-2).

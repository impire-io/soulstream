# Tasks: The Curator Persona

**Input**: Design documents from `/specs/009-curator/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: included — SC-001…SC-005 are proven by tests. Judgment tests are
serverless; habit tests run on the embedded server with short idle windows and scan
ticks so the suite stays fast.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

*None: existing module, zero new dependencies.*

---

## Phase 2: Foundational (Blocking Prerequisites)

- [ ] T001 [P] Additive `MaterializedTopic.BaselineTs` in topic/view.go — the fold records `recs[0].Record.Timestamp`; extend the round-trip strip/compare in topic/rollup_fold_test.go deliberately (post-rollup BaselineTs is the rollup baseline's time — legitimate difference, excluded from equivalence like stream_seq); fold test rows for birth and post-rollup values
- [ ] T002 [P] `topic.RespondDiscoveryWith(ctx, c, answer, onServed)` in topic/discover.go — extracted from 008's responder; `RespondDiscovery` becomes the board-backed wrapper with byte-identical behaviour (existing discovery tests must pass unmodified); a test proving a custom answerer's entries reach the asker

**Checkpoint**: `make check` green; 008 behaviour unchanged.

---

## Phase 3: User Story 1 — The best answerer in the room (Priority: P1) 🎯 MVP

**Goal**: warm, live, content-aware projection answering discovery via the existing
mechanism; stopping the curator restores 008 exactly.

**Independent Test**: a body-only phrase finds its topic with the curator running,
credited to the curator persona; stopped ⇒ 008 behaviour; board untouched.

- [ ] T003 [US1] Pure pieces in curator/suggest.go + curator/judge.go + curator/doc.go — suggestion constants/builders/recognisers (`[curator]` prefixes, author-independent); `Similarity` (token Jaccard over name+subject+tags, lowercased alphanumeric tokens, topic-id suffix excluded) with `DuplicateThreshold = 0.5`; content search over cached text; serverless tests in curator/judge_test.go (similar pairs above threshold, unrelated pairs below, id-suffix exclusion, recognisers reject ordinary comments mentioning the word curator)
- [ ] T004 [US1] Projection in curator/projection.go — seed from `Board` + `Materialise` per path; cachedTopic {view, DiscoverEntry, searchText, lastReal, birth} per data-model (lastReal excludes recognised suggestions, ≥ BaselineTs); malformed topics cached as skip-markers; dirty-marking via one core subscription on `SOULSTREAM.TOPICS.>` (unknown INFO paths added); `refresh(ctx)` re-materialises dirty paths; `search(query, limit)` matches identity fields + searchText
- [ ] T005 [US1] `curator.Run` (answering slice) in curator/curator.go — Options with defaults (IdleWindow 336h, ScanEvery 1m, OnEvent); build projection, start dirty subscription, serve via `RespondDiscoveryWith(projection answerer)` (refresh dirty before answering), block until cancel; OnEvent lines for projection-ready and answers
- [ ] T006 [US1] US1 e2e tests in curator/curator_test.go — body-only phrase found via `Discover` and credited to the curator persona (SC-001, scenario 2); name match answered like any responder (scenario 1); topic posted *after* curator start is found (scenario 3, live projection); curator + plain responder both credited (scenario 4); after cancelling the curator the same ask returns silence/plain-only and `Board` still lists everything (scenario 5 / SC-004)
- [ ] T007 [US1] CLI `curate` command (answering slice visible) in internal/cli/curate_cmd.go + cli.go dispatch/usage — long-running under the session persona (required), `--idle`/`--scan-every` flags, OnEvent → one line each; test in internal/cli/curate_cmd_test.go (cancellable Run: banner, projection-ready line, an ask answered by the curator persona)
- [ ] T008 [US1] Write docs/curator.md (ELI5 — the librarian: knows every shelf including what's *inside* the books, answers fastest, leaves polite sticky notes, never moves your books; fire the librarian and the library still works) + docs/README.md index entry + docs/discovery.md curator paragraph gains the link

**Checkpoint**: content-aware discovery live in dogfood; `make check` green.

---

## Phase 4: User Story 2 — "These two look the same" (Priority: P2)

**Goal**: one polite duplicate flag in the newer topic; log-idempotent across
restarts and curators.

**Independent Test**: near-duplicate start ⇒ exactly one flag naming the older path,
across a curator restart and a second curator.

- [ ] T009 [US2] Duplicate pass in curator/curator.go — on each scan tick (after refresh): for each non-malformed topic newest-first, best older `Similarity ≥ DuplicateThreshold` and no duplicate-kind suggestion present ⇒ `AddComment(DuplicateSuggestion(olderPath), frontier anchor)` in the newer topic; OnEvent line per flag; skip archived (writes refused) and closed (flagging a resting topic is noise)
- [ ] T010 [US2] US2 e2e tests in curator/curator_test.go — near-duplicate earns exactly one flag in the *newer* topic naming the older path (scenario 1, SC-002); restart the curator ⇒ no second flag (scenario 2); second concurrent curator stays quiet once flagged (scenario 3); unrelated topics never flagged (scenario 4); flag renders as an ordinary attributed comment in `show` output with zero client changes (SC-005)

**Checkpoint**: duplicate noise gets one visible nudge; `make check` green.

---

## Phase 5: User Story 3 — "This one seems finished" (Priority: P3)

**Goal**: one dormancy proposal per quiet spell; curator chatter never counts.

**Independent Test**: short window ⇒ exactly one proposal; activity + quiet again ⇒
exactly one more; closed/archived get none.

- [ ] T011 [US3] Dormancy pass in curator/curator.go — on each scan tick: for each topic with lifecycle ∉ {closed, archived}, not malformed, `now − lastReal > IdleWindow`, and no dormancy-kind suggestion newer than lastReal ⇒ `AddComment(DormantSuggestion(idle), frontier anchor)`; OnEvent line per proposal
- [ ] T012 [US3] US3 e2e tests in curator/curator_test.go — dormant topic gets exactly one proposal across repeated scans and a restart (scenarios 1–2, SC-003); fresh activity then quiet ⇒ exactly one more (scenario 3); a topic whose only recent ops are curator suggestions still counts dormant (scenario 4); closed and archived topics get none (scenario 5); announcement-only topic eligible via BaselineTs (edge case)
- [ ] T013 [US3] Update docs/curator.md with the two sticky-note kinds' exact wording and the one-per-quiet-spell promise; docs/cli.md gains the `curate` button + idle/scan flags

**Checkpoint**: all three habits live; `make check` green.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T014 Validate specs/009-curator/quickstart.md against real CLI output and fix drift; root README.md — delivered list gains 009, package table gains the `curator` row; FR-001 sweep: `curator` imports only public library surfaces (no internal/, no unexported reach-ins) and nothing in `realm`/`topic`/`registry`/clients references `curator` (the realm must not know curators exist); final `make fmt && make test && make lint` green, none skipped

---

## Dependencies & Execution Order

- T001 ∥ T002 → T003 → T004 → T005 → T006; T007 after T005; T008 closes US1.
- T009 → T010 (needs the projection + scan loop from US1).
- T011 → T012; T013 closes US3.
- T014 last.

## Implementation Strategy

US1 alone already improves dogfood discovery. US2 then US3 add the two suggestion
habits on the same scan loop. Merge `009-curator` to main with `--no-ff` after
Phase 6, then push.

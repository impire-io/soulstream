# Tasks: Scatter/Gather Topic Discovery

**Input**: Design documents from `/specs/008-discover/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: included — SC-001…SC-006 are proven by tests. Matcher/merge tests are
serverless; ask/answer round-trips run on the embedded server with sub-second
deadlines so the suite stays fast.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

*None: existing module, zero new dependencies.*

---

## Phase 2: Foundational (Blocking Prerequisites)

- [X] T001 Service plumbing — `SvcSubjectPrefix`/`SvcDiscoverSubject` constants and the SVC case in `canonicalBinding` (service-name binding) in topic/subjects.go with table-test rows; `(*realm.Client).Conn()` accessor in realm/connect.go with a test
- [X] T002 Vocabulary + explicit-binding build — `TypeDiscover`, `TypeDiscoverReply`, `DiscoverPayload`, `DiscoverEntry`, `DiscoverReplyPayload` in topic/vocab.go per contracts/library.md; factor topic/wire.go so a record can be built+signed over an explicit binding while publishing to a different subject (the discovery reply's inbox), with existing callers unchanged

**Checkpoint**: `make check` green, no behaviour change anywhere.

---

## Phase 3: User Story 1 — Ask the realm, merge what comes back (Priority: P1) 🎯 MVP

**Goal**: the ask — request, gather until deadline, merge with attribution and
verification; silence resolves empty.

**Independent Test**: scripted answerers (raw subscriptions in tests) → merged,
deduped, attributed results; zero answerers → empty at deadline, no error.

- [ ] T003 [US1] Pure matcher + merge in topic/discover.go — `matchEntries` (case-insensitive substring over name/subject-matter/tags; "" matches all; limit clamped [1,50]) and `mergeReplies` (key = path; one credit per (path, persona); first-seen fields win; arrival order); serverless tests in topic/discover_test.go
- [ ] T004 [US1] Implement `topic.Discover` in topic/discover.go — inbox + `SubscribeSync`, publish the signed request (binding `DISCOVER`) with Reply set, gather replies until `Timeout` (default 2s, limit default 10), parse/verify each reply (`VerifyRecord` with binding `DISCOVER` against kr), skip malformed, merge; zero replies ⇒ (nil, nil)
- [ ] T005 [US1] Ask-side e2e tests in topic/discover_test.go — scripted raw answerer: matches returned with attribution (SC-001); two answerers overlapping ⇒ one entry, both credited (SC-002); duplicate replies from one persona ⇒ one credit; zero answerers ⇒ empty within deadline + small constant, nil error (SC-003); replies after the deadline ignored; signed answerer ⇒ per-answer verified with keyring, unsigned ⇒ unsigned, wrong key ⇒ failed (SC-004); malformed reply skipped
- [ ] T006 [US1] CLI `discover <query> [--timeout] [--limit] [--json]` in internal/cli — merged render with per-answerer sig glyphs (reader keyring via realmKeyring), empty-result message naming the board fallback, usage text; tests in internal/cli/discover_cmd_test.go (scripted answerer over the test server URL)
- [ ] T007 [US1] MCP `soulstream_discover` (11th tool) in internal/mcpserver — input `{query, limit?}`, session keyring, JSON DiscoverResults with per-answer `sig`, empty list on silence; registration + test
- [ ] T008 [US1] Update docs/discovery.md — add the scatter/gather layer (ELI5: shout across the workshop "anyone seen a topic about X?"; whoever's around answers from their own notes; you collect answers until your patience runs out; nobody answering just means check the notice board) and the two-layer framing; touch docs/README.md hook line if needed

**Checkpoint**: asks work against scripted answerers from both clients; `make check` green.

---

## Phase 4: User Story 2 — Any persona may answer (Priority: P2)

**Goal**: the responder — any persona, own projection, silent when empty, no
coordination.

**Independent Test**: two responders under different personas both answer one ask;
stopping them degrades to empty results with the board intact.

- [ ] T009 [US2] Implement `topic.RespondDiscovery` in topic/discover.go — plain subscribe (NO queue group) on the discovery subject; per request: parse record (malformed ⇒ skip silently), rebuild `Board`, `matchEntries`, reply only when matches exist (signed reply over binding `DISCOVER` to the request's Reply inbox; requests without a Reply are skipped); optional `onServed(query, sent)` callback (−1 for skipped); blocks until ctx cancel, then returns nil
- [ ] T010 [US2] Responder e2e tests in topic/discover_test.go — full round-trip through `Discover` (US2 scenario 1); no-match request produces zero wire replies (raw inbox assertion, SC-005); two responders answer independently and the merge credits both (scenario 3); responder survives a malformed request and keeps serving (scenario 5); after cancelling responders, asks return empty and `Board` still lists everything (scenario 4 / SC-003)
- [ ] T011 [US2] CLI `respond` in internal/cli — long-running (persona required, Ctrl-C to stop, mirrors watch/inbox structure), one log line per served request via `onServed`; usage text; test with a cancellable context that asks once and asserts the served line and the ask's result
- [ ] T012 [US2] Update docs/cli.md (discover + respond buttons, silence tip) and docs/mcp.md (11th tool; why agents don't answer this cycle)

**Checkpoint**: dogfood discovery works end to end; `make check` green.

---

## Phase 5: Polish & Cross-Cutting Concerns

- [ ] T013 Validate specs/008-discover/quickstart.md against real CLI output and fix drift; root README.md — delivered list gains 008, `topic` package row gains discovery, eleven MCP tools; confirm zero registry/broker/queue-group constructs (FR-008); final `make fmt && make test && make lint` green, none skipped

---

## Dependencies & Execution Order

- T001 ∥ T002 → US1 (T003 → T004 → T005; then T006 ∥ T007; T008 closes) →
  US2 (T009 → T010; T011; T012 closes) → T013.
- Docs tasks close their story (Constitution III).

## Implementation Strategy

US1 first (asks are testable against scripted answerers), then US2 gives the realm a
real responder. Merge `008-discover` to main with `--no-ff` after Phase 5.

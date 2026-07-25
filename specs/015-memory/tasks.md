# Tasks: Memory Convention & Exhibits

**Input**: Design documents from `/specs/015-memory/`
**Prerequisites**: plan.md, spec.md (clarified), research.md (D1–D9), data-model.md, contracts/library.md, contracts/wire.md, quickstart.md

**Tests**: Included — the constitution's quality gate demands green `make check` with real coverage, the spec's success criteria are test-shaped (SC-004/005/006 are end-to-end demonstrations), and every prior feature shipped test-first surfaces. Tests live beside implementation per house convention.

**Organization**: Tasks grouped by user story. US1 = query + grading, US2 = exhibits, US3 = witness surface + fetch. In-package tests may use internals for *counterpart* scaffolding (e.g. a canned witness for US1 before US3 exists), but SC-005's proof runs in an external `topic_test` package using ONLY public surfaces — that test is the archivist repo's contract check.

## Phase 1: Setup

No scaffolding needed — existing module, existing packages, no new dependencies. (Skipped by design; see plan Project Structure.)

## Phase 2: Foundational (blocking prerequisites for all stories)

- [ ] T001 [P] Add `ServiceMemory = "MEMORY"` and `SvcMemorySubject = SvcSubjectPrefix + "MEMORY"` beside the discovery constants in `topic/subjects.go`
- [ ] T002 [P] Add op types `memory.query` / `memory.answer` / `memory.fetch` / `memory.exhibit` and wire payload structs (`MemoryQueryPayload{Query, Scope{Topics, After}, Deadline}`, `MemoryAnswerPayload{Answer, Citations, CoverageFrom omitzero}`, `MemoryFetchPayload{Topic, OpID, Deadline}`, `MemoryExhibitPayload{Exhibit}`, `MemoryCitation{Topic, OpID}`) in `topic/vocab.go` per contracts/wire.md
- [ ] T003 [P] Implement `record.Exhibit` (fields per data-model: version/realm/binding/subject/headers/payload_b64), `(Exhibit) Marshal()`, `ParseExhibit` (strict decode, version check), `(Exhibit) Record()` via `record.Parse` in `record/exhibit.go` — NO NATS imports
- [ ] T004 [P] Pure tests for exhibit round-trip (byte-identical re-marshal), strict-decode rejections (unknown field, missing field, wrong version), Record() reconstruction in `record/exhibit_test.go`

**Checkpoint**: `go build ./...` green; exhibit document format locked.

## Phase 3: User Story 1 — Ask the realm, get graded testimony (P1)

**Goal**: `topic.MemoryQuery` publishes a query, gathers ≤100 answers until the clamped deadline, verifies witness signatures (binding `MEMORY`), and grades every citation by actually resolving it. Zero witnesses ⇒ clean empty result.

**Independent Test**: In-package canned witness (raw subscribe + signed reply) answers a query; asker receives merged, attributed, graded answers by deadline. No witness ⇒ empty result at deadline, no error.

- [ ] T005 [US1] Implement pure resolver `(mt *MaterializedTopic) ContainsOp(opID string) bool` scanning announcement, contributions + edit stamps, attachments, work items + timeline, current baseline op id, frontier in `topic/resolve.go` (per research D4)
- [ ] T006 [P] [US1] Pure resolver tests: live ids, baked ids (StreamSeq 0), edit-stamp ids, work timeline ids, frontier, announcement, baseline id, negatives (compacted-away mark ops resolve false) in `topic/resolve_test.go`
- [ ] T007 [US1] Implement `MemoryQuery(ctx, c, MemoryQueryInput, kr)` in `topic/memory.go`: input validation (non-empty query → loud local error before publish), timeout default 3s clamp [100ms, 30s] (`DefaultMemoryTimeout`/`MinMemoryTimeout`/`MaxMemoryTimeout`), `buildOpMsg(c, SvcMemorySubject, ServiceMemory, …)` + core publish with `NewRespInbox`, gather loop with `MaxMemoryAnswers = 100` cap, per-answer verify (failed ⇒ discard, unsigned ⇒ keep + status, empty answer text ⇒ discard), `ErrNoResponders`/timeout = silence, citation grading via memoised `Materialise` + `ContainsOp` → `fact`/`unverifiable`; result types `MemoryResult`/`MemoryAnswer`/`GradedCitation` + `MemoryGrade` constants per data-model
- [ ] T008 [US1] Embedded-server tests in `topic/memory_test.go`: round-trip with canned witness (graded fact against a real topic op; unverifiable for unknown op-id; gossip standing for citation-less answer), witnessless silence at deadline, late reply ignored, 100-answer cap, failed-sig answer discarded, unsigned answer kept with status, deadline clamping, empty-query local error
- [ ] T009 [US1] Add `memory` command group with `query` subcommand (`--topics`, `--after`, `--timeout`, `--json`; human output: witness + sig marker + coverage, per-citation grade tags, gossip tag for citation-less answers, clean `no answers`) in `internal/cli/memory_cmd.go`; register `case "memory"` + usage in `internal/cli/cli.go`
- [ ] T010 [P] [US1] CLI query tests (graded output, JSON mode, no-answers path, flag validation) in `internal/cli/memory_cmd_test.go`
- [ ] T011 [US1] Add `soulstream_memory_query` MCP tool (input `{query, topics?, after?, timeout_ms?}`, nil-normalised `[]`, snake_case JSON) in `internal/mcpserver/memory_tools.go`, register in `internal/mcpserver/server.go`, test in `internal/mcpserver/memory_tools_test.go`
- [ ] T012 [P] [US1] Write `docs/memory.md` (ELI5: asking the whole class what they remember; answers are testimony; grades fact → gossip; silence is honest; checked never trusted)

**Checkpoint**: US1 fully demonstrable via CLI + MCP against a canned witness.

## Phase 4: User Story 2 — Exhibits: evidence that outlives the stream (P2)

**Goal**: Export any live op as a portable exhibit; verify it anywhere with zero realm connectivity; tampering always detected.

**Independent Test**: Export a signed op to a file, verify offline (pins-only keyring) → verified; flip one byte → failed.

- [ ] T013 [US2] Implement `CaptureExhibit(ctx, c, path, opID)` (ordered-consumer scan of the topic's `OPS.<path>` + `INFO.<path>` subjects, verbatim headers+payload capture, binding = canonical binding of the captured subject, `ErrOpNotLive` sentinel) and `VerifyExhibit(e, kr) (SigStatus, error)` (reconstruct via `e.Record()`, delegate to `VerifyRecord(rec, e.Realm, e.Binding, kr)`) in `topic/exhibit.go`
- [ ] T014 [US2] Embedded-server + pure tests in `topic/exhibit_test.go`: capture a live turn op and the announce op; not-found ⇒ `ErrOpNotLive`; verify verdicts — verified (chain rule incl. rotated author), failed on any single-byte tamper (payload, headers, binding, realm, sig), unsigned op ⇒ unsigned, unknown author ⇒ unknown-key, distrusted author ⇒ failed; export never launders (bad-sig op captures and reports failed)
- [ ] T015 [US2] Add `memory exhibit <topic> <op-id> [-o file] [--force] [--json]` (live-only; `ErrOpNotLive` → error naming `memory fetch`; overwrite-guard + `os.WriteFile` per the artefact-get pattern) and `memory verify <file>` (OFFLINE: never connects, keyring from pins file alone via `keystore.LoadPins`, prints verdict/author/realm/binding/type/ts, exit failed⇒1, works with broken realm config — dispatch before connect) in `internal/cli/memory_cmd.go`
- [ ] T016 [P] [US2] CLI tests for exhibit/verify: file round-trip, tampered file ⇒ failed + exit 1, unknown-key warning path, verify-with-broken-config survival (013 lesson), overwrite-guard in `internal/cli/memory_cmd_test.go`
- [ ] T017 [P] [US2] Write `docs/exhibits.md` (ELI5: a sealed note anyone can check; kept by anyone, believed because of the seal, not the keeper; unsigned notes are only as good as their keeper)

**Checkpoint**: US2 independently demonstrable: export → offline verify → tamper detection.

## Phase 5: User Story 3 — Anyone can serve memory; the archivist plugs in from outside (P3)

**Goal**: Public witness surface (`RespondMemory`, nilable capabilities, `coverage_from`) + live-first `FetchExhibit` with first-verifying-wins. Compacted history recoverable from any keeper; the external archivist's contract proven from outside.

**Independent Test**: A witness built ONLY on public surfaces (external test package) serves an answer and a kept exhibit; after rollup removes the cited op, fetch through the witness restores it verifying.

- [ ] T018 [US3] Implement `RespondMemory(ctx, c, MemoryWitness)` in `topic/memory.go`: plain subscribe (no queue group) on `SvcMemorySubject` + `Flush`, dispatch by op type, nil capability ⇒ silent for that kind, stale-deadline skip (`OnServed(kind, -1)`), malformed skip, reply via `buildOpMsg(c, msg.Reply, ServiceMemory, …)` signing when signer present, `coverage_from` included when set; types `MemoryWitness`/`MemoryQueryRequest`/`MemoryAnswerDraft` per data-model
- [ ] T019 [US3] Implement `FetchExhibit(ctx, c, path, opID, timeout, kr)` in `topic/memory.go`: live-first via `CaptureExhibit` (Source `"live"`), else publish `memory.fetch` + gather `memory.exhibit` replies — reply-op sig check (failed ⇒ discard) then embedded-exhibit verdict via `VerifyExhibit`: first `verified` wins immediately, `unsigned` held as fallback, `failed` discarded; silence ⇒ `(nil, nil)`; `ExhibitResult{Exhibit, Verdict, Source}` per data-model; plus pure helper `GradeForVerdict(SigStatus) MemoryGrade` (verified→fact-with-provenance, unsigned→testimony, failed/unknown-key→unverifiable) so clients never reimplement the mapping
- [ ] T020 [US3] Embedded-server witness tests in `topic/memory_test.go`: both capabilities served, Answer-nil witness silent on queries but serves fetches (and vice versa), stale request skipped with OnServed(-1), multi-witness fetch preference (verified beats earlier unsigned; unsigned returned only when nothing verifies), witness self-answering own query
- [ ] T021 [US3] Success-criteria tests: **SC-004** end-to-end compaction recall (publish signed op → CaptureExhibit → witness keeps it → Rollup compacts the tail → live capture now ErrOpNotLive → FetchExhibit via witness → verified) in `topic/memory_test.go`; **SC-005** outsider-witness test in external package `topic/memory_pubsurface_test.go` (package `topic_test`, public identifiers only — the archivist-repo contract proof); **SC-006** zero-residue check (count stream messages before/after a batch of query+fetch traffic, delta 0)
- [ ] T022 [US3] Add `memory fetch <topic> <op-id> [--timeout] [-o file] [--force] [--json]` (live-first; prints verdict + author + source; not-found ⇒ message + exit 1; `-o` writes exhibit with overwrite-guard) in `internal/cli/memory_cmd.go` + tests in `internal/cli/memory_cmd_test.go`
- [ ] T023 [US3] Add `soulstream_memory_fetch` MCP tool (`{topic, op_id, timeout_ms?}` → `{found, verdict?, source?, exhibit?}`) in `internal/mcpserver/memory_tools.go`, register in `server.go` (23 tools total — update any tool-count assertion), test in `internal/mcpserver/memory_tools_test.go`
- [ ] T024 [P] [US3] Extend `docs/memory.md` with the witness section (serving what you keep; `coverage_from` = "my notes start here"; archivists are an optimisation, never a requirement; the archivist lives in its own repository)

**Checkpoint**: All three stories independently demonstrated; the archivist contract proven from an external package.

## Phase 6: Polish & Cross-Cutting

- [ ] T025 Reconcile surfaces with docs: quickstart.md flags/outputs match the shipped CLI; contracts/library.md matches actual signatures; update the docs index (`docs/README.md` if present) and any doc that enumerates SVC conventions or MCP tool counts
- [ ] T026 Full quality gate `make check` (fmt + tidy + build + test + lint) — all green, zero skipped; fix everything it surfaces
- [ ] T027 Journey episode for 015 in `hq/04-JOURNEY/` via the journey duty (same change as the landed feature), including the separate-archivist-repo decision and its reversal condition

## Dependencies & Execution Order

- **Foundational (Phase 2)** → blocks all stories. T001–T004 all parallel-safe.
- **US1 (Phase 3)**: T005→T006 parallel with T007 prep; T007 needs T001+T002+T005; T008 after T007; T009→T010 after T007; T011 after T007; T012 anytime.
- **US2 (Phase 4)**: T013 needs T003; T014 after T013; T015 after T013; T016 after T015; T017 anytime. US2 is independent of US1.
- **US3 (Phase 5)**: T018 needs T002 (+T003 for fetch drafts); T019 needs T013+T018; T020–T021 after T018+T019 (SC-005 also exercises US1's MemoryQuery publicly); T022 after T019; T023 after T019; T024 after T012.
- **Polish** last.

## Implementation Strategy

MVP = Phase 2 + US1 (query + grading against canned witnesses) — independently valuable and demonstrable. US2 adds the evidence layer; US3 completes the convention and proves the external-archivist contract. Implementation runs in the main loop (house convention: tightly-coupled Go in one pass for a guaranteed-green build), stories in priority order, checkpoint commits at each phase boundary.

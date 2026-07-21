# Tasks: Re-baselining (Rollup), Manifest Baselines & Archived

**Input**: Design documents from `/specs/007-rollup/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: included — SC-001…SC-006 are proven by tests; the constitution's gate
requires everything green. Fold/round-trip tests are serverless; rollup/race/manifest
tests use `internal/natstest.StartJetStream(t)`.

**Organization**: foundational payload/fold work first (both stories need it), then
stories in priority order; every checkpoint leaves `make check` green.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

*None: existing module, zero new dependencies, no scaffolding.*

---

## Phase 2: Foundational (Blocking Prerequisites)

- [ ] T001 Extend the wire vocabulary in topic/vocab.go — `BaselinePayload` gains `State` `omitempty`, `Baked *BakedState`, `Manifest *ManifestRef`; new `BakedState` and `ManifestRef` types per contracts/library.md; `Archived Lifecycle = "archived"` constant joins vocab; give the view structs in topic/view.go explicit lowercase JSON tags (`op_id`, `author`, `ts`, `type`, `body`, `mentions`, `anchor`, `dangling`, `sig`, `stream_seq,omitempty`, attachment fields, notification fields, `MaterializedTopic` fields) — update every test asserting JSON casing (internal/mcpserver `"Sig": "verified"` → `"sig": "verified"`, etc.)
- [ ] T002 Teach the fold to seed in topic/view.go `apply` — resolve `baked` into contributions/attachments/lifecycle before folding the tail; baked interior op-ids → seen ∪ referenced (anchor-resolvable, never frontier); `payload.frontier` ids → seen (frontier candidates; baseline op-id is candidate only when frontier is empty); terminal rule: once archived, later transitions ignored with a warning; a `baseline` op appearing mid-log (a live follower that retained pre-rollup history sees the landed rollup as its next message) is skipped as a checkpoint with a specific benign warning, never "unknown type"; serverless round-trip tests in topic/view_test.go proving `apply(rolled-up form + tail)` ≡ `apply(full log + tail)` for a log containing every element kind (SC-001 core)
- [ ] T003 Baked provenance in topic/verify.go — `annotateView` gains the baseline op-id: elements without a per-op status inherit the baseline's status; serverless tests (signed baseline ⇒ baked `verified`, unsigned baseline ⇒ baked `unsigned`, tail ops keep their own)
- [ ] T004 Publish variant in topic/wire.go — `publishOpMsg` (or options on `publishOp`) accepting extra NATS headers (`Nats-Rollup`) and `jetstream.PublishOpt`s (`WithExpectLastSequencePerSubject`), sharing the exact record-build + sign path; no behaviour change for existing callers

**Checkpoint**: pre-007 logs fold identically; `make check` green.

---

## Phase 3: User Story 1 — Compact a topic and nothing changes (Priority: P1) 🎯 MVP

**Goal**: manual rollup with replay equivalence and race safety.

**Independent Test**: full-featured topic → rollup → identical view from 1 message;
raced attempts lose cleanly.

- [ ] T005 [US1] Implement `Handle.Rollup` (inline form) in topic/rollup.go — materialise; refuse malformed topics and archived topics (`ErrTopicArchived` placeholder wired fully in US3); `ErrNothingToCompact` when the log is a lone baseline; build `{state, frontier, baked}` payload; publish via T004 with rollup header + guard at last consumed `StreamSeq`; map the wrong-last-sequence API error to `ErrRollupLost`; on success set the handle frontier to the payload frontier; baseline record's `Soulstream-Parents` = consumed frontier
- [ ] T006 [US1] End-to-end rollup tests in topic/rollup_test.go — identical view before/after for every element kind incl. anchors and mentions (SC-001); subject holds exactly 1 message after (SC-002); posting after rollup parents onto the payload frontier and comments anchor to baked op-ids (US1 scenarios 2, 6); post-vs-rollup and rollup-vs-rollup races: first writer wins, loser gets `ErrRollupLost`, zero ops lost (SC-003, scenarios 3–4); signed compactor: baseline verifies, baked elements report the baseline's status (scenario 5); `ErrNothingToCompact` on a fresh topic; live `Follow` stays consistent across a rollup landing mid-follow
- [ ] T007 [US1] CLI `rollup <path>` in internal/cli — success prints folded-op count, `ErrNothingToCompact` prints "nothing to compact" (exit 0), `ErrRollupLost` prints a retryable message (non-zero); usage text; tests in internal/cli/rollup_cmd_test.go
- [ ] T008 [US1] MCP `soulstream_rollup_topic` (10th tool) in internal/mcpserver/tools.go + server.go registration per contracts/library.md; test in internal/mcpserver
- [ ] T009 [US1] Write docs/rollup.md (ELI5 — gluing this week's sticky notes into a fresh first page; the race = two people reaching for the glue, first one wins, nothing tears) and add to docs/README.md index

**Checkpoint**: dogfood topics can be compacted by hand from both clients; `make check` green.

---

## Phase 4: User Story 2 — Oversized state still compacts to one message (Priority: P2)

**Goal**: manifest baselines with crash-safe write order and digest-checked reads.

**Independent Test**: >128 KB state → one manifest message → digest-identical cold
read; simulated crash leaves the old log readable.

- [ ] T010 [US2] Manifest write path in topic/rollup.go — when the marshalled state document exceeds `InlineBaselineThreshold`: put one object `baseline/<path>/<new-op-id>` (bytes = the inline `{state, baked}` document), publish `{frontier, manifest{chunks,digest,size}}` with the same header + guard, then delete the superseded baseline's manifest objects if any; digest via the existing attachment digest helper
- [ ] T011 [US2] Manifest read path in topic/materialise.go (+ board's lifecycle derivation) — resolve the baseline's state document before the fold: manifest ⇒ fetch chunks in order, concatenate, `VerifyDigest`, unmarshal; missing/corrupt ⇒ `Malformed` with a clear reason, never a crash or partial state (FR-011)
- [ ] T012 [US2] Manifest tests in topic/manifest_test.go — oversized topic rolls up to exactly one manifest message and cold-reads digest-identical (SC-004); crash simulation: object written but baseline never published ⇒ original log replays exactly, orphan harmless; lost race after object write ⇒ same; corrupted object and missing object ⇒ malformed with reason; successful re-rollup deletes the superseded object (US2 scenario 4)
- [ ] T013 [US2] Update docs/rollup.md (the overflow box: when the glued page won't fit, it goes in the cupboard and the notebook keeps a claim ticket with a seal over the contents) and docs/materialisation.md (cold read may fetch the claim ticket's box first)

**Checkpoint**: topics of any size compact; `make check` green.

---

## Phase 5: User Story 3 — Closing tidies up; archiving is final (Priority: P3)

**Goal**: mandated lifecycle triggers; archived terminal with hard write refusal.

**Independent Test**: close ⇒ compacted closed topic; archive ⇒ single terminal
message, reads fine, every write refused everywhere.

- [ ] T014 [US3] Lifecycle mechanics in topic/lifecycle.go + topic/post.go — `Archived` accepted by `definedLifecycle`; `ErrTopicArchived` returned by every Handle write path (Post, PostTurn, AddComment, Attach, Transition, Rollup) when the last observed lifecycle is archived (closed keeps warn-but-allow); `Handle.Close` = transition(closed) + one best-effort rollup attempt (lost race swallowed); `Handle.Archive` = transition(archived) + rollup with 3 bounded retries re-materialising between attempts (exhaustion ⇒ error, transition stands)
- [ ] T015 [US3] Lifecycle trigger tests in topic/archive_test.go — close leaves a compacted closed topic (single message when unraced) and a raced close still yields a valid uncompacted closed topic; archive ends as exactly one terminal baseline (US3 scenario 2), reads return full state (scenario 3), every library write path refuses with `ErrTopicArchived` (scenario 4), double archive reports already-archived (scenario 5), archive raced by a concurrent post retries and lands (edge case)
- [ ] T016 [US3] CLI in internal/cli — `archive <path>` (loud confirmation; already-archived refusal), `close` switches to `Handle.Close`; archived refusal errors surface verbatim on post/comment/attach/close/rollup; usage text; tests
- [ ] T017 [US3] MCP in internal/mcpserver — `soulstream_close_topic` switches to `Handle.Close`; write tools surface the archived refusal verbatim; no archive tool (spec clarification); tests incl. an agent attempting to write to an archived topic
- [ ] T018 [US3] Update docs/lifecycle.md (archived: the notebook is bound and shelved — read it forever, write in it never) and docs/topic.md, docs/cli.md (rollup/archive buttons, close-tidies note), docs/mcp.md (10th tool; no archive tool and why)

**Checkpoint**: the full lifecycle story holds across all three surfaces; `make check` green.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T019 Validate specs/007-rollup/quickstart.md against real CLI output and fix drift; root README.md: delivered list gains 007, `topic` package row gains rollup/archived, MCP tool list gains the 10th tool
- [ ] T020 FR-013 sweep (no lock/lease/election anywhere — grep `lock|lease|elect` in non-test code and confirm zero coordination constructs); full `make fmt && make test && make lint` green, none skipped

---

## Dependencies & Execution Order

- **Foundational**: T001 → T002/T003 (need the types); T004 independent [P with T002/T003].
- **US1**: T005 needs T001–T004; T006 needs T005; T007/T008 [P] after T005; T009 closes the story.
- **US2**: T010 needs T005; T011 [P with T010]; T012 needs both; T013 closes.
- **US3**: T014 needs T005; T015 needs T014; T016/T017 [P] after T014; T018 closes.
- **Polish**: last.
- Docs tasks close their story (Constitution III).

## Implementation Strategy

US1 alone is a meaningful stop (manual compaction, dogfood-usable). US2 then US3, each
at a green checkpoint; merge `007-rollup` to main with `--no-ff` after Phase 6.

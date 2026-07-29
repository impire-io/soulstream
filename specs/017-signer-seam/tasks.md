# Tasks: Signer Seam

**Input**: Design documents from `/specs/017-signer-seam/`
**Prerequisites**: plan.md, spec.md (Clarifications 2026-07-29), research.md
(R1–R7), data-model.md, contracts/library.md, quickstart.md

**Tests**: Included — the spec's success criteria are test-shaped (SC-001,
SC-002, SC-003, SC-005 name measurable behaviors proven with doubles), and
the project's gate demands them.

**Organization**: By user story. US1 and US4 are both P1: US1 is the
feature's point, US4 is the constraint on it; US4's verification tasks run
after the others because they certify the whole.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

**Purpose**: Confirm the starting point so every later diff is attributable.

- [x] T001 Run `make check` on the branch base and record it green (baseline;
      no code changes)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The seam itself. Changing `(*SigningKey).Sign` to be fallible
(R1) breaks compilation at every consumer at once, so the interface, the
shape change, and all mechanical call-site updates land together,
behavior-preserving. No story work can begin before this compiles green.

**⚠️ CRITICAL**: One coherent change; commit only when `make check` is green.

- [x] T002 Create `identity/signer.go`: the `Signer` interface
      (`PublicKey() string`; `Sign(canonical []byte) (string, error)`) with
      contract docs — concurrency-safe required (FR-011), never `("", nil)`,
      no seed access implied (FR-008), implementations own their deadlines
      (R3)
- [x] T003 Change `(*SigningKey).Sign` to `(string, error)` (error always
      nil) in `identity/sign.go`; assert `*SigningKey` satisfies `Signer`
      with a compile-time `var _ identity.Signer` check in
      `identity/sign_test.go`; update in-package callers/tests
- [x] T004 Change `realm.Config.Signer` and `Client.Signer()` to
      `identity.Signer` in `realm/connect.go`; document the typed-nil
      assignment rule on the field (R6)
- [x] T005 Update the chokepoint `buildOpMsg` in `topic/wire.go`: handle the
      fallible `Sign`, wrap a signer error naming the op type (FR-004), and
      convert an empty signature to an error (FR-005) — no unsigned fallback
- [x] T006 [P] Mechanically update `registry/attest.go`
      (`NewAttestationToken`) and `registry/kv.go` (`Rotate`) for the
      fallible `Sign` — parameter types stay `*identity.SigningKey` here;
      widening to the interface is US3's task
- [x] T007 [P] Update `internal/cli` and `internal/mcpserver` for the new
      shapes; enforce the assignment discipline (assign a loaded
      `*identity.SigningKey` to `Config.Signer` only when non-nil — audit
      `internal/cli/key.go` loadSigner consumers and the MCP server setup)
- [x] T008 Update remaining test files broken by the shape change
      (mechanical `Sign` error handling) until `make check` is fully green —
      behavior-preserving: no test expectation about *behavior* may move

**Checkpoint**: `make check` green; every existing behavior unchanged
[the compiler has walked every call site]; stories can begin.

---

## Phase 3: User Story 1 — Sign through a custodian, verify like any other record (Priority: P1) 🎯 MVP

**Goal**: A delegated `Signer` implementation produces records every
existing reader verifies identically to local-key signing (SC-001).

**Independent Test**: A test double wrapping a real key behind the interface
publishes ops; read surfaces report verified; signatures byte-identical to
local signing.

### Tests for User Story 1

- [x] T009 [US1] Add a delegate-double signer (wraps a real `SigningKey`
      behind `identity.Signer`, counts calls) in `topic/sign_test.go`;
      prove: published turn carries a signature byte-identical to
      local-key signing over the same canonical bytes (SC-001, US1-AS2),
      and materialise/follow/inbox report `SigStatus` verified (US1-AS1);
      include a concurrent-publish subtest (several goroutines through one
      delegated client) exercising the FR-011 contract — run at least once
      under `go test -race ./topic` (recorded in T028)
- [x] T010 [P] [US1] Extend exhibit coverage: an op signed through the
      delegate double captures into an exhibit that verifies
      (`CaptureExhibit`/`GradeForVerdict`) in `topic/exhibit_test.go` —
      `GradeForVerdict` IS the offline-verify machinery, closing SC-001's
      exhibit and offline surfaces together
- [x] T011 [P] [US1] Assert the nil-signer path is untouched: unsigned
      publish byte-identical semantics (`sig` absent, `SigStatus` unsigned)
      in `topic/sign_test.go` (FR-006, US1-AS3)

### Implementation for User Story 1

- [x] T012 [US1] Implementation is Phase 2 (the seam); this task is the
      story's verification sweep: run `make check`, confirm T009–T011 green
      with zero changes to non-test code beyond Phase 2
- [x] T013 [US1] Document delegated signing in `docs/signing.md` (ELI5:
      "someone else can hold your pen" — the pen stays in a vault, you
      describe the letter, the letter looks exactly the same to every
      reader)

**Checkpoint**: US1 fully functional — delegation is transparent.

---

## Phase 4: User Story 4 — Existing local-key users notice nothing (Priority: P1)

**Goal**: Certify zero user-visible change for local-key and unsigned
clients (FR-010, SC-004).

**Independent Test**: Full suite green with unchanged behavioral
expectations; dependency set unchanged.

- [x] T014 [US4] Sweep the diff for behavioral drift: existing tests may
      only have gained mechanical error-handling on `Sign` calls — no
      changed expectations, no removed assertions; record the sweep result
      in the PR/commit message
- [x] T015 [P] [US4] Confirm `go.mod`/`go.sums` untouched (SC-004: no new
      dependencies) and `identity` still imports no NATS (FR-009) — assert
      via `go list -deps ./identity` in a test or a task-level check
- [x] T016 [P] [US4] CLI/MCP behavior freeze check: `soulstream key init`,
      signed post, `profile publish`, MCP `publish_profile` paths exercised
      by existing suites — verify none needed expectation changes (FR-010);
      note in commit message
- [x] T016b [US4] Docs-accuracy check (US4's docs duty): re-read
      `docs/signing.md`'s local-key description against the shipped seam and
      confirm it needed no change — the freeze story's claim, in docs form;
      fix any drift found (stale docs = bug, Constitution III)

**Checkpoint**: Both P1 stories done — the seam exists and nobody fell
through it.

---

## Phase 5: User Story 2 — A failing custodian never produces an unsigned record (Priority: P2)

**Goal**: Signing failure = operation failure, loudly (SC-002); responders
go silent, observably (FR-012, SC-005).

**Independent Test**: Failure-injecting double: publishes error and leave no
record; responders answer nothing and report `-1`.

### Tests for User Story 2

- [x] T017 [US2] Failure-injection tests in `topic/sign_test.go`: a signer
      returning an error fails `StartTopic`/`PostTurn`/`AddComment` (the
      representative publish surfaces), the returned error names the signing
      cause, and the op log gained nothing (US2-AS1, SC-002)
- [x] T018 [P] [US2] Empty-signature test in `topic/sign_test.go`: a signer
      returning `("", nil)` is treated exactly as an error (FR-005, US2-AS2)
- [x] T019 [P] [US2] Responder-silence tests: discovery responder with a
      failing signer sends no reply and reports `served(query, -1)` in
      `topic/discover_test.go`; memory witness likewise for answer and
      exhibit paths (`served(…, -1)`) in `topic/memory_test.go` (FR-012,
      SC-005)

### Implementation for User Story 2

- [x] T020 [US2] Implementation already lives at the chokepoint (T005) and
      the existing responder error paths (R5) — this task verifies no
      further code was needed; if a gap surfaces (e.g. an error path that
      swallows the cause), fix it minimally at the chokepoint
- [x] T021 [US2] Extend `docs/signing.md`: the fail-loudly rule and
      responder silence, in plain words (a jammed pen means the letter is
      not sent — never sent unsigned; a helper who cannot sign says
      nothing)

**Checkpoint**: Failure semantics proven end to end.

---

## Phase 6: User Story 3 — Operator statements through the same seam (Priority: P3)

**Goal**: Attestation tokens and rotation proofs accept any `Signer`
(FR-007, SC-003).

**Independent Test**: Both statements produced via a delegate double pass
existing verification.

### Implementation for User Story 3

- [x] T022 [US3] Widen `registry.NewAttestationToken` signer parameter to
      `identity.Signer` in `registry/attest.go` (nil refusal kept; signer
      error propagates wrapped)
- [x] T023 [US3] Widen `registry.Rotate` old/new parameters to
      `identity.Signer` in `registry/kv.go` (old: Sign + PublicKey; new:
      PublicKey only; signing failure aborts before any KV write)

### Tests for User Story 3

- [x] T024 [P] [US3] Attestation-via-delegate test in
      `registry/attest_test.go`: token from a delegate double passes
      `AttestationStatus` verification (US3-AS1, SC-003)
- [x] T025 [P] [US3] Rotation-via-delegate test in `registry/kv_test.go`
      (Rotate lives in kv.go): rotation whose proof came from a delegate
      double validates through the existing chain rules, and a
      failing delegate aborts with no KV write (US3-AS2, SC-003)
- [x] T026 [US3] Note in `docs/operators.md` that attesting and rotating
      work when the key is held by a custodian (one plain-words paragraph;
      cross-reference docs/signing.md)

**Checkpoint**: All stories independently verified.

---

## Phase 7: Polish & Landing Duties

- [x] T027 Validate `quickstart.md` against the shipped reality (the three
      verification steps map to T009/T017/T024-T025; the wiring snippet
      compiles conceptually against the final API)
- [x] T028 Full gate: `make check` (fmt+tidy+build+test+lint) — all green,
      none skipped
- [x] T029 Landing bookkeeping in the merge change: CLAUDE.md SPECKIT block
      → landed 017; `hq/03-IMPLEMENTATION/ROADMAP.md` reflects the seam
      (SoulIdentity M2 wiring point available); journey episode via
      `/journey-log` (the journey duty — same change as the merge)

---

## Dependencies & Execution Order

- **Phase 2 blocks everything** — the seam is one coherent compile-green
  change (T002→T005 sequential on shared understanding; T006/T007 parallel
  after T003; T008 last).
- **US1 (Phase 3)** first after foundational: it is the MVP; its double
  (T009) is shared with US2 (same file, `topic/sign_test.go`).
- **US4 (Phase 4)** may run any time after Phase 2 but certifies the final
  state — its sweep (T014) repeats cheaply at the end.
- **US2 (Phase 5)** independent of US1 except reusing the double's file.
- **US3 (Phase 6)** independent; T022/T023 touch files already mechanically
  updated in T006. Its tests define their own minimal double in the
  `registry` test files — test helpers do not cross package boundaries.
- **Phase 7** last.

### Parallel Opportunities

- T006 ∥ T007 (different packages, after T003/T004/T005).
- T010 ∥ T011 (different files/areas after T009 lands the double).
- T018 ∥ T019 (different files).
- T024 ∥ T025 (different files, after T022/T023).

---

## Implementation Strategy

MVP = Phase 2 + Phase 3 (US1): the seam exists, delegation is transparent,
proven. US4 certifies the freeze, US2 the failure story, US3 the statement
surfaces — each independently checkable, each committed when green. The
whole feature is small by design (Article II): if any task grows beyond its
description, stop and re-check the plan rather than absorbing scope.

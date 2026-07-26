# Tasks: Provisioning Byte Limits

**Input**: Design documents from `/specs/016-provision-limits/`
**Prerequisites**: plan.md, spec.md (clarified 2026-07-27), research.md D1–D6, data-model.md, contracts/library.md, quickstart.md

**Tests**: included — the spec's acceptance scenarios hinge on reproducing the
limit-enforcing tier (US1's independent test), and the repo's gate forbids
landing behavior without tests.

**Organization**: by user story, so each story is an independently testable
increment.

## Phase 1: Setup

No setup tasks — the feature lives entirely in existing packages of an
existing module; no new dependencies, no scaffolding.

## Phase 2: Foundational (blocking prerequisites)

- [ ] T001 Add `Budgets` struct + `DefaultBudgets()` (1 GiB / `NotifyMaxBytes` / 64 MiB / 512 MiB) with doc comments stating zero-field semantics (D2) and the notify mandate rule (D3), in `realm/spec.go`
- [ ] T002 [P] Add embedded-server variant with account `MaxBytesRequired: true` (the NGS R1 switch) alongside the existing helper, in `internal/natstest/natstest.go`

**Checkpoint**: `Budgets` compiles and is constructible; a test server that
refuses roofless streams can be started. No user story implemented yet.

## Phase 3: User Story 1 — Provision a realm on a limit-enforced account (P1) 🎯 MVP

**Goal**: one command provisions all four artefacts on an account that
refuses unlimited streams; without budgets the failure stays exactly as
today.

**Independent test**: against the `MaxBytesRequired` fixture, provisioning
with `DefaultBudgets()` creates everything; provisioning without budgets
fails with an error naming the refused artefact.

- [ ] T003 [US1] Thread budgets through provisioning: config constructors take the relevant budget (`streamConfig`/`notifyStreamConfig`/`personasConfig`/`objectStoreConfig`), `ProvisionOn(ctx, js, budgets ...Budgets)` validates (len ≤ 1, no negative field, error names the artefact, before any server call) and applies budgets at creation only, in `realm/spec.go` + `realm/provision.go`
- [ ] T004 [US1] Pass budgets through `Client.Provision(ctx, budgets ...Budgets)`, in `realm/connect.go`
- [ ] T005 [US1] Tests: on the `MaxBytesRequired` fixture, no-budgets provisioning fails with the artefact-naming error (FR-003) and `DefaultBudgets()` provisioning creates all four artefacts with the documented roofs; validation-error cases (negative field, two Budgets values) never touch the server, in `realm/provision_test.go`
- [ ] T006 [US1] CLI: `--budgets` switch on the provision command mapping to `DefaultBudgets()`, with a command test on the fixture, in `internal/cli/provision.go` + `internal/cli/provision_test.go`

**Checkpoint**: US1 fully functional — an NGS-R1-shaped account provisions
out of the box with `provision --budgets`; behavior without the switch is
unchanged.

## Phase 4: User Story 2 — Choose the budgets explicitly (P2)

**Goal**: per-artefact budgets; named budgets deviate, the switch fills the
rest, flags alone leave the rest unlimited (clarification 2026-07-27).

**Independent test**: provision with one named budget and verify that
artefact carries it while the others follow the switch/no-switch rule.

- [ ] T007 [P] [US2] Size parsing + formatting helpers (bytes and KiB/MiB/GiB binary suffixes; format prints `1.0 GiB` / `unlimited`), pure and server-free, with table-driven unit tests, in `internal/cli/size.go` + `internal/cli/size_test.go`
- [ ] T008 [US2] CLI: `--budget-oplog/--budget-notify/--budget-personas/--budget-objects` flags; explicit `0` or negative rejected at parse naming the artefact (FR-005/D2); composition per the clarification (flags overwrite switch defaults; flags alone start from zero `Budgets`); tests cover flag+switch, flag-alone, and rejection cases, in `internal/cli/provision.go` + `internal/cli/provision_test.go`
- [ ] T009 [US2] Library test: a partial `Budgets` (one non-zero field) budgets only that artefact — notify keeps its mandated roof, the rest stay unlimited (D3 + clarification), in `realm/provision_test.go`

**Checkpoint**: explicit budgets work end to end with both composition
modes.

## Phase 5: User Story 3 — Re-provision an existing realm honestly (P3)

**Goal**: budgets never mutate an existing artefact; the report shows each
artefact's roof as found.

**Independent test**: provision twice with different budget choices — the
second run changes nothing and reports the first run's roofs.

- [ ] T010 [US3] `ArtefactResult.MaxBytes`: as-applied for created artefacts, as-found for existing ones (read from backing stream configs incl. `KV_soulstream-personas` / `OBJ_soulstream-objects`, D4), in `realm/report.go` + `realm/provision.go`
- [ ] T011 [US3] Tests: re-provision with different budgets mutates nothing and reports as-found roofs (US3 scenario 1); the hand-created-workaround shape (pre-existing artefacts with roofs, as on the live NGS realm) reports conformant with roofs visible (US3 scenario 2); legacy-shape convergence still preserves tuned settings and takes no budgets, in `realm/provision_test.go`
- [ ] T012 [US3] CLI report: roof column per artefact (`1.0 GiB` / `unlimited`) using the US2 formatter, with output test, in `internal/cli/provision.go` + `internal/cli/provision_test.go`

**Checkpoint**: all three stories independently green.

## Phase 6: Polish & cross-cutting

- [ ] T013 [P] ELI5 docs: "storage budgets" section — shelf-space analogy, defaults table, "provisioning never resizes an existing shelf", NGS R1 walkthrough from quickstart.md, in `docs/provisioning.md`
- [ ] T014 Full gate: `make fmt && make test && make lint` — all green, none skipped; confirm SC-002 by inspection that pre-existing provisioning tests pass unmodified
- [ ] T015 Live SC-003 check: run `soulstream provision --budgets` against the existing NGS realm (context `personal`) — expect conformant report, roofs visible, nothing mutated; record the output in the PR/merge notes

## Dependencies

- Phase 2 blocks everything (T001 blocks T003+; T002 blocks T005/T006/T008/T011).
- US1 (T003–T006) blocks US2 and US3 (they extend the threaded budgets and the CLI command).
- US2's T007 is parallel-safe immediately after Phase 2; T012 (US3) uses T007's formatter.
- Polish runs last; T013 can start once US1 exists.

## Parallel opportunities

- T001 ∥ T002 (different packages).
- After US1: T007 ∥ T010 (different packages), then T008/T009 ∥ T011.
- T013 ∥ any post-US1 work (docs file only).

## Implementation strategy

MVP = Phase 2 + US1 (an R1 account provisions out of the box; that alone
retires the manual workaround). US2 and US3 are small increments on top;
polish closes the constitution's docs obligation and the live-realm
verification.

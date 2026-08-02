# Tasks: an agent runs

**Input**: `/specs/003-an-agent-runs/` (plan, research R1–R5)
**Tests**: mandatory.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [ ] T001 `go.mod`: soulrealm at its pseudo-version (plan Complexity
      Tracking); tidy.

## Phase 2: Foundational (ceremony)

- [ ] T002 `ceremony/`: workload minting key (plain; realm JWT
      `SigningKeys.Add`) + `runner` bypass user; files
      `keys/workload-signing.nk`, `users/runner.creds`; inventory,
      `Verify` (realm JWT endorses the plain key), `ArtifactCount` → 20;
      tests extended.

## Phase 3: User Story 1 — the runtime plane (US1)

- [ ] T003 [US1] `node/workload.go`: `RunWorkload(ctx, cfg, url,
      declPath)` — parse/validate declaration; connect `runner` creds at
      url (fail fast, name `soulnode up` when unreachable);
      `PersonaSigner("runner")` → realm client;
      `minter.NewSigningKeyMinter(workloadSeed, realmPub, {url})`;
      `runner.Runner{Minter, Backend: native.New(), Realm, CredTTL,
      ScratchRoot: <state>/scratch}`; `Launch` + `Serve`; close.
- [ ] T004 [US1] `cmd/soulnode/main.go`: `workload start <file>
      [--state DIR]` wiring RunWorkload with `nats://<listen>`; usage
      updated.
- [ ] T005 [US1] `node/node_test.go` `TestM13AgentRuns`: build upstream
      `agent-echo` from the module cache; owner starts topic; declare;
      RunWorkload; assert turn authored by the workload persona, work
      open/claim/done ops present (materialised read), everything in the
      archive; scratch creds confined to the workload's scratch dir
      (SC-001/002).
- [ ] T006 [P] [US1] `cmd/soulnode/main_test.go`: refusal paths — node
      down (names `soulnode up`), invalid declaration (field named),
      missing artifact (SC-003).

## Phase 4: Polish & Landing

- [ ] T007 Full gate green (SC-004); Phase 1 exit criteria check against
      design 0001 §9.
- [ ] T008 Landing duties: journey episode 0005; roadmap M1.3 + Phase 1
      complete; design 0001 propagation (§4 ceremony keys, §6
      invocation-scoped runtime wording); spec Status → implemented;
      tasks checked.

## Dependencies

T001 → T002 → T003 → (T004, T005, T006) → T007 → T008

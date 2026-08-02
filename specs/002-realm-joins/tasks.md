# Tasks: the realm joins

**Input**: `/specs/002-realm-joins/` (plan, research R1–R6, contracts/config.md)
**Tests**: mandatory.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 `go.mod`: add `soulstream v0.6.0` + archivist pseudo-version
      (R-pins per plan Complexity Tracking); tidy.

## Phase 2: Foundational (ceremony)

- [X] T002 `ceremony/`: `Generate` gains the archivist bypass-lane user;
      `config.json` gains `realm` (Generate param; default wired in cmd)
      and `planes.memory.enabled` (contracts/config.md); `State` carries
      Realm + MemoryEnabled; inventory + `Verify` + `ArtifactCount`
      updated; ceremony tests extended (new file present, config
      roundtrip, count).

## Phase 3: User Story 1 — the memory plane (US1)

- [X] T003 [US1] `node/found.go`: founding acts gain
      `realm.ProvisionOn(ctx, js)` under the realm name (R2).
- [X] T004 [US1] `node/node.go`: `up`-path substrate guard
      (`ProvisionOn` create-or-verify) + the memory plane when enabled —
      archivist connection, `PersonaSigner("archivist")`,
      `realm.NewClient` (plane owns the conn), `archive.Open`,
      `keeper.Run` + `topic.RespondMemory` under the node ctx; startup
      failures abort Start named; runtime exits surface loud (R5);
      `Stop` closes the realm client.
- [X] T005 [US1] `cmd/soulnode/main.go`: `init --realm` (founding-run
      semantics), `up` logs "memory plane serving" when enabled.
- [X] T006 [US1] `node/node_test.go`: the M1.2 arms — owner's full path
      (token admission → sign via identity plane → post turn → memory
      query cites it); restart continuity (each op exactly once, counted
      via public `archive.Open`); disabled-plane arm (M1.1 observations
      green, no `archive/`) (SC-001/002/004). Vault-held persona check:
      `PersonaPublicKey("archivist")` answers and no persona key file
      exists under the state dir (SC-003).

## Phase 4: Polish & Landing

- [X] T007 Full gate green (SC-005); quickstart untouched by design
      (M1.1 journey unchanged) — verify.
- [X] T008 Landing duties: journey episode 0004; roadmap M1.2 measured
      outcome; design 0001 propagation (§2 plane block as-built, §4
      inventory + realm name, §6 as-wired); spec Status → implemented;
      tasks checked.

## Dependencies

T001 → T002 → (T003, T004, T005) → T006 → T007 → T008

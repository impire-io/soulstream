# Tasks: soulnode init and up

**Input**: Design documents from `/specs/001-init-and-up/`
**Prerequisites**: plan.md, spec.md, research.md R1–R8, data-model.md, contracts/{cli,state-dir}.md

**Tests**: mandatory (constitution VI overrides the template default).

**Organization**: US1 (first boot) and US2 (up + admission) are the same
M1.1 gate at equal priority; the phases below order them by dependency.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Wire dependencies and build: `go.mod` gains soulidentity at
      the R1 pseudo-version (+ nats-server/nats.go/jwt/nkeys direct);
      drop the premature soulstream requirement (enters at M1.2);
      Makefile `build` gains `go build -o bin/ ./cmd/...`.

## Phase 2: Foundational

- [X] T002 `ceremony/ceremony.go`: `State` struct + `Generate(listen)` —
      operator, SYS/AUTH/realm accounts (external authorization, allowed
      accounts, callout xkey; JetStream limits + scoped signing key with
      the design §4 template), the two signing-key seeds, three curve
      seeds, three account-signed users (service, issuer, ops) as creds
      text (data-model "Ceremony").
- [X] T003 `ceremony/state.go`: `Save(dir)` (0700/0600, config.json,
      refuse-on-weak-modes), `Load(dir)`, `Verify(dir)` implementing the
      five verification invariants incl. the completion marker rule
      (research R4), plus `WriteSentinel(dir, creds)` as the last-write
      founding marker.
- [X] T004 [P] `ceremony/ceremony_test.go`: generate/save/load
      roundtrip; mode assertions; damage matrix (missing file, corrupt
      seed, JWT/seed mismatch, incomplete-no-sentinel); verify
      idempotence (SC-004).

## Phase 3: User Story 1 + 2 — the composition (US1, US2)

- [X] T005 [US2] `node/node.go`: `Config` + `Start(cfg) (*Node, error)`
      — embedded server from pure `server.Options` (TrustedKeys,
      SystemAccount, MemAccResolver preloaded from persisted JWTs,
      JetStream store dir, bind from config; bind-conflict and
      vault-mismatch failures named), then the identity plane via
      soulidentity `embed.Run` on two loopback connections from
      `users/*.creds`; readiness = status answers; `Stop()` drains
      planes (ctx), closes its connections, shuts the server down.
- [X] T006 [US1] `node/found.go`: `Found(n, state) (token string, err)`
      — the founding acts through public `client` over the ops
      connection: import realm scoped signing key + AUTH signing key,
      `CreateToken` (the first token), `MintSentinel` →
      `ceremony.WriteSentinel` last (research R3/R4).
- [X] T007 [US1] [US2] `cmd/soulnode/main.go`: `init` (fresh → generate,
      save, transient Start+Found+Stop, print the token block per
      contracts/cli.md; complete → verify + report, no token;
      incomplete/damaged → named refusal), `up` (verify, Start, log
      state dir + listener + serving, signal-drain, exit), `version`;
      state-dir resolution flag/env/default; `--listen` founding-run
      semantics.
- [X] T008 [US1] [US2] `node/node_test.go`: the M1.1 e2e — fresh dir →
      init path (Generate+Save+Start+Found) → the three admission
      observations through public surfaces (sentinel+token admits,
      `$SYS.REQ.USER.INFO` persona + own-prefix confinement; garbage
      refused + `callout REFUSED` in captured audit; revoked refused) →
      Stop → Start again on the same dir (restart works) (SC-002,
      SC-003).
- [X] T009 [P] [US1] `cmd/soulnode/main_test.go`: CLI contract — fresh
      init prints exactly one token line; re-init prints none, exit 0;
      up on uninitialized dir exits non-zero naming init; version
      answers (SC-003).

## Phase 4: Polish & Landing

- [X] T010 Verify quickstart.md against the real output lines; package
      docs read plainly (constitution IV analog — sibling discipline).
- [X] T011 Full gate `make check` green, nothing skipped (SC-005);
      confirm `init && up` wall-clock is well inside SC-001's minute.
- [X] T012 Landing duties in the same merge: journey episode 0003
      (`/journey-log`), roadmap M1.1 updated with measured outcomes,
      design 0001 propagation (config schema as-built), spec Status →
      implemented.

## Dependencies

```
T001 ─▶ T002 ─▶ T003 ─┬▶ T004 [P]
                      └▶ T005 ─▶ T006 ─▶ T007 ─▶ T008
                                              └▶ T009 [P]
T010 [P] anytime; T011 after T008+T009; T012 last
```

## Implementation Strategy

MVP = through T008 (the composition proven end to end); T009 pins the
CLI contract. Estimated ~900 LOC + tests; the ceremony content is the
research rig's proven `Provision`, split along the R2 seam.

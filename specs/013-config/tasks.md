# Tasks: Config-file identity resolution & self-installing plugin binary

**Input**: Design documents from `/specs/013-config/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/library.md

**Organization**: Foundational package first (blocks everything), then the three user
stories in priority order, then polish. Tests are included (the contract demands
byte-for-byte compatibility — that is only provable by tests).

## Phase 1: Foundational — `internal/config` (blocks all stories)

- [x] T001 Create `internal/config/config.go`: `File` struct (5 fields, json tags per data-model), strict `loadFile(path)` with `DisallowUnknownFields` + path-naming errors + config-dir-relative key/pins resolution, `findProjectFile(cwd)` walk-up (nearest only, stop at root), `userFile()` via `os.UserConfigDir()/soulstream/config.json`.
- [x] T002 Create `internal/config/resolve.go`: `SourceKind`/`Source`/`Value`/`Resolved`, `Resolve(explicit File, cwd string) (Resolved, error)` per-field chain flag>env>project>user>unset (env names `SOULSTREAM_{CONTEXT,REALM,PERSONA,KEY_FILE,PINS_FILE}`), `(Resolved).Fields()` stable-order accessor per contract.
- [x] T003 [P] Create `internal/config/config_test.go` + `resolve_test.go`: walk-up nearest-wins over t.TempDir trees; per-field cross-layer merge; every precedence pair; malformed JSON / unknown field errors name the file; absent files skip; empty `{}` contributes nothing; relative key_file resolves against file dir; no-files+no-env ⇒ all unset (SC-005 baseline).

**Checkpoint**: `go test ./internal/config/` green — pure, no server.

## Phase 2: User Story 1 — per-project identity (P1) 🎯 MVP

**Goal**: CLI and MCP server pick up identity from `.soulstream.json` / user config.
**Independent test**: two temp dirs with different files ⇒ two identities, no flags/env.

- [x] T004 [US1] Rewire `internal/cli/cli.go` Run: flags default `""` (drop `os.Getenv` defaults), collect explicitly-set flags via `flag.Visit`, call `config.Resolve(explicit, cwd)`, build `Config` from resolved values; update usage text Configuration section to document the four-source chain; keep all existing error wording.
- [x] T005 [US1] Rewire `cmd/soulstream-mcp/main.go` run(): same Visit+Resolve flow before keystore/connect; `-version` and persona-required behaviour unchanged.
- [x] T006 [US1] Extend `internal/cli/cli_test.go`: project-file identity honoured end-to-end (write `.soulstream.json` in a temp cwd — note Run resolves from process cwd, so test via `config.Resolve` injection point chosen in T004… if Run takes cwd from `os.Getwd()`, use `t.Chdir`); env-over-file and flag-over-env precedence through real Run; unknown-field file fails any command with the path in stderr.
- [x] T007 [P] [US1] Write `docs/configuration.md` (ELI5: the sticker on the project folder that says who you are there; both file levels; the chain in plain words; "the file names you, it can never BE you" security paragraph) and link it from `docs/README.md`.
- [x] T008 [P] [US1] Update `docs/cli.md` + `docs/mcp.md`: config file resolution section, per-project MCP identity via cwd.

**Checkpoint**: US1 acceptance scenarios 1–5 pass; `make test` green.

## Phase 3: User Story 2 — `soulstream config` (P2)

**Goal**: one command shows each field's value + true source, offline.
**Independent test**: overlapping sources ⇒ truthful table; empty world ⇒ all unset, exit 0.

- [x] T009 [US2] Create `internal/cli/config_cmd.go`: `cmdConfig` printing field / effective value (`(unset)` when empty) / source description per contract; wire `config` into the dispatch switch + usage text; never calls connect.
- [x] T010 [P] [US2] Create `internal/cli/config_cmd_test.go`: source truthfulness (flag vs env vs project file vs user file vs unset), exit 0 with nothing configured, exit 1 + file-naming message on malformed file.
- [x] T011 [P] [US2] Add `config` command to `docs/cli.md` command table + example output.

**Checkpoint**: US2 scenarios pass.

## Phase 4: User Story 3 — self-installing wrapper (P2)

**Goal**: plugin fetches its own verified binary; overrides respected; cache re-verified.
**Independent test**: quickstart's fresh-machine / cache-hit / tamper sequence.

- [x] T012 [US3] Rewrite `plugins/soulstream/scripts/soulstream-mcp.sh` per contract: SOULSTREAM_MCP_BIN → PATH → verified cache `$DATA/bin/v<ver>/` → download (version from `.claude-plugin/plugin.json` via sed; `uname -s/-m` → darwin|linux / amd64|arm64; curl→wget fallback; shasum→sha256sum fallback; checksums.txt verification; temp dir under $DATA; record binary sha256; atomic mv; exec). Every failure: named stderr message + manual options, exit 1, temp dir cleaned.
- [x] T013 [US3] Bump `plugins/soulstream/.claude-plugin/plugin.json` and the plugin entry in `.claude-plugin/marketplace.json` to 0.2.0.
- [x] T014 [US3] Verify wrapper paths locally: `bash -n`; SOULSTREAM_MCP_BIN path; PATH path; seeded-cache hit (no network); tampered cache re-fetch trigger; full download path against the existing v0.1.0 release using a scratch copy of the plugin with version 0.1.0 (no code knob — test-setup only).
- [x] T015 [P] [US3] Update `plugins/soulstream/README.md` (self-install lifecycle, cache location, overrides, `.soulstream.json` per-project config) and `plugins/soulstream/skills/setup/SKILL.md` (binary step becomes "automatic — overrides for developers"; config-file step replaces env vars as the primary path, env stays documented).

**Checkpoint**: US3 scenarios 1–5 pass (scenario 1 fully re-verified post-release in T018).

## Phase 5: Polish & release pairing

- [x] T016 [P] Touch `README.md`: config-file mention in the CLI/plugin sections (chain in one line, link to docs/configuration.md).
- [x] T017 Run quickstart.md end-to-end locally; `make check` all green; `claude plugin validate .`.
- [ ] T018 After merge to main: push, tag `v0.2.0` (signed), watch release workflow, then re-run wrapper fresh-machine test against the real v0.2.0 release (SC-003/SC-004 final proof).

## Dependencies & execution order

- Phase 1 blocks everything. US1 (T004–T008) before US2 (T009–T011: the command
  prints what Resolve returns, and cli.go rewiring lands in T004). US3 independent of
  US1/US2 — may run in parallel with them after Phase 1. T018 is post-merge by nature.
- Parallel opportunities: T003 beside T002 finalisation; T007/T008 beside T004–T006;
  T010/T011 beside T009; T015 beside T012–T014; T016 anytime.

## Implementation strategy

MVP = Phase 1 + US1 (identity follows the project). US2 is the debugging window, US3
completes the plugin story; each checkpoint leaves the tree green and shippable.

# Tasks: CLI Client for Humans

**Input**: Design documents from `specs/004-cli/` · **Prerequisites**: plan, spec, contracts
**Tests**: INCLUDED. **Organization**: by 4 user stories. Adds `internal/cli` + `cmd/soulstream`.

---

## Phase 1: Setup

- [X] T001 `cmd/soulstream/main.go`: `func main(){ os.Exit(cli.Main(os.Args[1:])) }`.

## Phase 2: Foundational (Blocking)

- [X] T002 `internal/cli/cli.go`: `Config`; `Run(ctx, args, stdout, stderr, connect) int` (global
  flags `--context/--realm/--persona` with env fallback, subcommand dispatch, usage); `Main`; usage
  text. Unknown command / no args → usage to stderr, exit 2. (FR-001/015/016)
- [X] T003 `internal/cli/connect.go`: `Connector` type; `realmConnect` (via `realm.Connect`);
  `withClient(ctx, connect, cfg, requirePersona, fn) int` — connect, enforce persona for writes,
  run fn, map errors → exit 2 + stderr, success → 0. (FR-002/003/018)
- [X] T004 [P] `internal/cli/render.go`: text + JSON renderers for a provision report, the board,
  and a materialised view. (FR-005/007)

**Checkpoint**: `go build ./...`; `soulstream` with no args prints usage (exit 2).

---

## Phase 3: US1 — Set up and see what's there (P1) 🎯 MVP

- [X] T005 [US1] `internal/cli/commands.go`: `cmdProvision` (print per-artefact result) and
  `cmdBoard` (`--json`; text list of path/name/lifecycle; empty realm → empty). (FR-004/005)
- [X] T006 [US1] Test `internal/cli/cli_test.go`: via `Run` + injected embedded connector —
  `provision` succeeds and prints artefacts; `board` on an empty realm prints nothing (exit 0);
  after starting topics, `board` and `board --json` list them; a failing connector → exit non-zero
  + stderr. (SC-001/002)
- [X] T007 [P] [US1] ELI5 doc `docs/cli.md` (start): the CLI as "the remote control — one word per
  action", with a copy-paste session. (Constitution III)

**Checkpoint**: provision + board work from `Run`.

---

## Phase 4: US2 — Start a topic and converse (P1)

- [X] T008 [US2] Extend `commands.go`: `cmdStart` (`--subject/--tag/--parent`, print path),
  `cmdShow` (`--json`; text of baseline/contributions+mentions/attachments/lifecycle), `cmdPost`,
  `cmdComment`. (FR-006/007/008/009)
- [X] T009 [US2] Extend `cli_test.go`: `start → post (with @mention) → comment → show` prints the
  contributions, authors, and mention; `show --json` round-trips; a write command with no persona →
  clear error, exit non-zero. (SC-001/002, FR-002)

**Checkpoint**: full converse loop from the CLI.

---

## Phase 5: US3 — Follow along and catch mentions (P2)

- [X] T010 [US3] `internal/cli/stream.go`: `cmdWatch <path>` (`topic.Follow`, print each new
  contribution) and `cmdInbox` (`topic.FollowInbox`, print each notification), both under the passed
  ctx; return 0 when ctx is cancelled. (FR-013/014)
- [X] T011 [US3] Test `internal/cli/stream_test.go`: with a short timeout ctx, `watch` prints a turn
  posted from another client and exits 0; `inbox` prints a notification and exits 0. (SC-004)

**Checkpoint**: live watch + inbox stream and exit cleanly.

---

## Phase 6: US4 — Exchange files and close out (P2)

- [X] T012 [US4] Extend `commands.go`: `cmdAttach` (`--type/--anchor`, print object key), `cmdGet`
  (`--force`; write bytes, verify digest, no clobber), `cmdClose`. (FR-010/011/012)
- [X] T013 [US4] Extend `cli_test.go`: `attach → show` lists it; `get` reproduces the bytes and
  verifies the digest; `get` without `--force` onto an existing file errors; `close → show` reports
  closed. (SC-003)

**Checkpoint**: files + close from the CLI.

---

## Phase 7: Polish

- [X] T014 [P] Finish `docs/cli.md`; update `README.md` (a `cmd/soulstream` line + build note).
- [X] T015 Run `go mod tidy`; `go vet ./...` clean; confirm `go build ./cmd/soulstream` produces a binary.
- [X] T016 Final gate: `make check` green — all tests pass (none skipped), lint 0. (SC-005)

---

## Dependencies

- Setup + Foundational (T001–T004) block the stories.
- US1 (T005–T007) → MVP. US2 (T008–T009) after Foundational. US3 (T010–T011), US4 (T012–T013) after
  Foundational; US4 reads what US2/US1 wrote in tests.
- Polish last.

## Notes

- Keep command handlers thin; all real work is `realm`/`topic` calls. Stdlib `flag` only.
- Commit per story (signed); `make check` before each commit.
- Out of scope: TUI, edit/reply/resolve, admin/registry commands.

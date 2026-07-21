# Implementation Plan: CLI Client for Humans

**Branch**: `004-cli` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)

## Summary

A `soulstream` binary (`cmd/soulstream`) and a testable `internal/cli` package: parse global flags
(`--context`/`--realm`/`--persona`) + a subcommand, connect via `realm.Connect` (named context),
and dispatch to one of eleven commands over `realm` + `topic`. Command logic is injectable
(`Run(ctx, args, stdout, stderr, connect)`) so it is tested against an in-process server without the
named-context path. No third-party CLI framework — stdlib `flag` only.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`).
**Primary Dependencies**: existing `realm`, `topic`; stdlib `flag`, `context`, `os/signal`, `io`,
`encoding/json`. No new third-party dependency.
**Storage**: none new (drives the provisioned realm).
**Testing**: `go test`; `internal/cli` command handlers tested via `Run(...)` with an injected
connector returning an embedded-server client (`realm.NewClient` over `internal/natstest`), asserting
stdout + exit code; streaming commands tested with a timeout context.
**Project Type**: single Go module; adds `cmd/soulstream` (thin main) + `internal/cli` (logic).
**Constraints**: stdlib arg parsing (FR-017); streaming commands end on SIGINT (exit 0).
**Scale/Scope**: 11 commands, ~a dozen files.

## Constitution Check

- **I. NATS-Native First** — ✅ PASS. The CLI is a pure client of the library; it adds no
  infrastructure, no service. All work is `realm`/`topic` calls over NATS.
- **II. Smallest Viable Implementation** — ✅ PASS. Stdlib `flag` (no cobra/urfave); one command =
  one thin handler; `--json` only where scripting needs it; streaming reuses `Follow`/`FollowInbox`.
  Explicitly excluded: TUI, edit/reply/resolve, admin/registry.
- **III. ELI5 Documentation** — ✅ PASS (planned). `docs/cli.md`: "the remote control for Soulstream
  — one word per action", with a copy-paste session. Plus the binary's own `--help`.

**Result**: PASS. Re-checked post-design: unchanged.

## Project Structure

```text
cmd/soulstream/
└── main.go               # os.Exit(cli.Main(os.Args[1:]))

internal/cli/
├── cli.go                # Config; Run(ctx, args, stdout, stderr, connect) int; Main; usage; dispatch
├── connect.go            # Connector type; realmConnect (natscontext); withClient helper (+ persona check)
├── commands.go           # cmdProvision/Board/Start/Show/Post/Comment/Attach/Get/Close
├── stream.go             # cmdWatch, cmdInbox (ctx-cancellable)
├── render.go             # text + JSON rendering of board/view/report
├── cli_test.go           # oneshot command matrix via Run + embedded server
└── stream_test.go        # watch/inbox with a timeout context

docs/cli.md
```

**Structure Decision**: Logic in `internal/cli` (not `cmd/`) so it is unit-testable; `cmd/soulstream`
is a two-line `main`. `Run` takes an explicit `ctx`, output writers, and a `Connector` — the three
seams that make every command testable against an embedded server and cancellable in a test.

## Key implementation notes

- **Config**: flags default to `SOULSTREAM_CONTEXT/REALM/PERSONA`; flags override env.
- **withClient(ctx, connect, cfg, requirePersona, fn)**: connects, enforces persona for write
  commands (FR-002), runs `fn(client)`, maps errors → exit 2 with a stderr message, success → 0.
- **Exit codes**: 0 success; 2 usage/error. Unknown command / bad args → usage to stderr, exit 2.
- **Streaming**: `watch`/`inbox` run `topic.Follow`/`topic.FollowInbox` under the passed ctx; SIGINT
  (via `signal.NotifyContext` in `Main`) cancels → exit 0.
- **get**: refuses to overwrite without `--force`; verifies digest via `topic.VerifyDigest`.

## Complexity Tracking

> No Constitution violations.

# Implementation Plan: One console, one vocabulary

**Branch**: `009-one-console-one-vocabulary` | **Date**: 2026-08-15 | **Spec**: [spec.md](spec.md)

## Summary

Mechanical Go renames (State `Door*`→`MCP*`, `Fold*`→`SignIn*`;
`node.DoorURL/FoldURL`→`MCPURL/SignInURL`), a dual-key config read
(`planes.signin`/`planes.mcp` preferred, `planes.fold`/`planes.door`
read forever), creds-path fallback (`users/signin.creds` then
`users/fold.creds`), flag aliases, the sign-in plane wired with
`DisableAdminConsole` (idp v0.5.0), functional wording in every
user-read string, and the docs sweep.

## Constitution Check

- **I (composition)**: PASS — no plane changes shape; one option flows
  through the public embed seam.
- **S2**: PASS — read-both keys instead of a config migration tool; a
  founded realm's artifacts are never rewritten.
- **V (ceremony is code)**: PASS — new founds write the new names in
  the same one ceremony; nothing interactive.

## Project Structure

```text
ceremony/state.go      # dual-key config, signin.creds for new founds
ceremony/ceremony.go   # State field renames; signin NATS user on found
node/node.go           # wiring, URLs, logs, console-off, creds fallback
cmd/soulstream/main.go # flags + aliases, usage, printEndpoints
docs/…, README.md      # the sweep; getting-started mirrors real output
```

# Implementation Plan: Wrap in the house

**Branch**: `008-wrap-in-the-house` | **Date**: 2026-08-15 | **Spec**: [spec.md](spec.md)

## Summary

Two thin verbs in `cmd/soulstream` over libraries `go.mod` already
pins: `wrap` (workloads `wrap.Preset`/`LoadTemplate` + `wrap.Wrapper`,
lane from the five `SOULSTREAM_*` names, tool door =
`os.Executable() + ["mcp"]`) and `mcp` (`realm.Connect` +
`mcpserver.NewServer(...).Run(ctx, &mcp.StdioTransport{})`). The
getting-started teaches download-and-paste; `go install` disappears.

## Technical Context

**Dependencies**: soulstream-workloads v0.4.0 (`mcp_args`, specs/007
there); soulstream-core v0.8.4 unchanged (`realm`, `mcpserver` public);
the MCP go-sdk arrives transitively. **Storage**: none — the verbs are
clients. **Testing**: refusal paths hermetic beside the existing CLI
contract tests; the live criterion recorded once by hand.

## Constitution Check

- **I (composition through public surfaces)**: PASS — both verbs
  consume tagged public packages; no internals crossed.
- **III (ordinary connections)**: PASS — the lane dials the declared
  agents address like any client.
- **V (ceremony is code)**: N/A — no ceremony change.
- **S2 (smallest viable)**: PASS — two thin mains; the alternatives
  (self-provisioning downloads, multi-binary archives) are recorded
  and argued against in design 0002 / journey.

## Project Structure

```text
cmd/soulstream/wrap.go    # cmdWrap + cmdMCP + the env lane
cmd/soulstream/main.go    # dispatch + usage text
cmd/soulstream/main_test.go
docs/getting-started.md   # steps 6–7: download, paste
```

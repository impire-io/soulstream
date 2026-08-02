# Implementation Plan: an agent runs

**Branch**: `003-an-agent-runs` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-an-agent-runs/spec.md`

## Summary

The ceremony gains the workload minting key (a *plain* realm signing key
— scoped keys reject carried permissions, measured upstream) and the
`runner` transport credential. A new `node.RunWorkload(ctx, cfg, url,
declPath)` composes soulrealm's public surfaces — `declaration.Parse/
Validate`, `minter.NewSigningKeyMinter`, `backend/native.New`,
`runner.Runner.Launch` + `Running.Serve` — on a realm client speaking as
persona `runner` (transport creds from the ceremony, signer from the
identity plane). `cmd/soulnode` gains `workload start`. The e2e re-runs
soulrealm's founding proof inside the composition: echo agent → turn
authored by its persona → work open/claim/done → all kept by the
archivist.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: adds `github.com/impire-io/soulrealm` at a
pseudo-version of its main (no tags exist upstream — third tracked pin,
same flip rule); its declaration/minter/backend/runner surfaces are
public (composition research Bar 2)
**Storage**: state dir gains `keys/workload-signing.nk`,
`users/runner.creds` (inventory 18 → 20) and `scratch/` (runtime working
data, not inventory — same class as `archive/`)
**Testing**: node e2e builds upstream's own `agent-echo` from the module
cache as the artifact (toolchain-hermetic); CLI refusal-path tests
**Target Platform / Project Type / Performance**: unchanged
**Constraints**: constitution I (public surfaces; the supervisor loop
stays upstream), II (workload admitted by its minted JWT, no bypass), III
(invocation-scoped runtime plane; fail-loud refusals), V (ceremony grows
but init stays two-command)
**Scale/Scope**: ~+300 LOC + tests

## Constitution Check

- **I — PASS** (pin exception as before, third instance: soulrealm,
  Complexity Tracking). No supervisor loop is invented here — the
  invocation-scoped command mirrors upstream's own; the claim-race node
  is upstream's Fleet milestone.
- **II — PASS**: the workload connects with its minted, TTL-bounded,
  permission-carrying user — the server enforces; no bypass creds reach
  a child.
- **III — PASS**: the runtime plane runs wherever the command runs,
  connected over the configured URL — the decomposition story working
  as designed (a remote runtime is this command on another machine).
- **IV/V/VI — PASS** as in 001/002.

## Project Structure

```text
ceremony/ceremony.go   # + workload signing key (plain), runner user
ceremony/state.go      # + two inventory files; realm-JWT endorsement check
node/workload.go       # RunWorkload: declaration → minter → native → runner
cmd/soulnode/main.go   # + workload start
node/node_test.go      # + TestM13AgentRuns (builds upstream agent-echo)
cmd/soulnode/main_test.go # + refusal paths
```

**Structure Decision**: `RunWorkload` lives in `node` so the cmd and the
test share one assembly (the 001 pattern); the URL is a parameter because
the command derives it from `config.json` while tests use the node's
ephemeral port.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Constitution I's "tagged releases": soulrealm pinned at a pseudo-version of main | Upstream has no tags at all (its journey 0011 dropped the replace; tagging is the maintainer's release act) | Same reasoning as the prior two rows; flips on upstream's first tag |

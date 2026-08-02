# Implementation Plan: the realm joins

**Branch**: `002-realm-joins` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-realm-joins/spec.md`

## Summary

`init` additionally provisions the realm's record substrate
(`realm.ProvisionOn` over the founding connection's JetStream handle,
under a configurable realm name persisted in `config.json`), the ceremony
gains the memory plane's bypass-lane credential, and `up` gains the
memory plane: the archivist's public `keeper` + `archive` packages wired
onto a realm client whose signer is the identity plane's
`PersonaSigner("archivist")` — key materialized in the vault on first
touch, nothing persona-shaped in the state directory. `config.json`
grows its first design-§2 plane block (`planes.memory.enabled`). The e2e
proves the full owner path: token-lane admission → post a turn (signed
through the identity plane) → memory answers with a citation; plus
restart continuity and the disabled-plane arm.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: adds `github.com/impire-io/soulstream v0.6.0`
(tagged) and `github.com/impire-io/soulstream-archivist` at a
pseudo-version above its v0.1.0 tag (the public keeper/archive seam
landed after it — second tracked pin, same flip rule as soulidentity's)
**Storage**: state dir gains `archive/` (the exhibit store) and one creds file
**Testing**: extends the `node` e2e; ceremony suite gains the new inventory entries
**Target Platform**: unchanged
**Project Type**: same three packages; no new package
**Performance Goals**: n/a — capture is the archivist's own measured path
**Constraints**: constitution I (public surfaces, no replace), III (the
plane block; ordinary loopback conns; fail-loud planes), V (init stays
two-command, zero manual steps)
**Scale/Scope**: ~+350 LOC + tests

## Constitution Check

- **I — PASS with the same tracked exception class as 001**: archivist
  pinned at a pseudo-version until upstream tags (Complexity Tracking);
  soulstream enters at its real tag v0.6.0. Only public packages
  (`realm`, `topic`, `keeper`, `archive`, `client`, `embed`).
- **II — PASS**: the owner's e2e path runs through token-lane admission,
  not around it.
- **III — PASS**: the memory plane is the first `planes.*` config block;
  `enabled: false` is honored; startup failure of the plane is a named
  refusal, never a silent absence.
- **IV — PASS**: builds design 0001 §6 as graduated.
- **V — PASS**: still two commands; provisioning joins the founding acts.
- **VI — PASS**: e2e rides `make test`, hermetic.

Post-design re-check: clean.

## Project Structure

```text
ceremony/ceremony.go      # + archivist bypass-lane user
ceremony/state.go         # + users/archivist.creds, realm name + plane
                          #   block in config.json, inventory count
node/node.go              # + memory plane wiring (enabled-gated), realm
                          #   substrate guard on up (ProvisionOn)
node/found.go             # + realm.ProvisionOn in the founding acts
cmd/soulnode/main.go      # + --realm on init; "memory plane serving" line
node/node_test.go         # + M1.2 e2e arms
specs/002-realm-joins/contracts/config.md
```

**Structure Decision**: no new packages — the plane lands inside `node`
exactly as design §6 wires it; pre-release, so the ceremony change is
wholesale (no migration shim for M1.1-era state dirs; none exist outside
tests).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Constitution I's "tagged releases": archivist pinned at a pseudo-version above v0.1.0 | Its public keeper/archive seam landed 2026-08-01, after its only tag | Same reasoning as 001's soulidentity row; flips on the next upstream tag |

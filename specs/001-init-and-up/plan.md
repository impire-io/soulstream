# Implementation Plan: soulnode init and up

**Branch**: `001-init-and-up` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-init-and-up/spec.md`

## Summary

SoulNode's first code: a `ceremony` package that generates, persists,
loads, and verifies the founding state directory (design 0001 §4); a
`node` package that composes the embedded operator-mode server (pure
`server.Options`, resolver preloaded from persisted JWTs, loopback
listener) with the identity plane (soulidentity's public `embed.Run` on
two ordinary loopback connections); and a `cmd/soulnode` CLI whose `init`
generates + transiently boots + performs the founding acts through the
public `client` (printing the first token exactly once), and whose `up`
runs the composition until interrupted. Acceptance is design §9-M1.1,
proven by an end-to-end test that runs init → up → the three admission
observations through public surfaces only.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: `nats-server/v2 v2.14.3` (embedded), `nats.go v1.52.0`, `jwt/v2 v2.8.2`, `nkeys v0.4.16`, `github.com/impire-io/soulidentity` (public `embed` + `client` only) — pinned at a pseudo-version of main until upstream's first tag (research R1; Complexity Tracking)
**Storage**: the state directory (files, `0700`/`0600`) + JetStream store dir inside it
**Testing**: `go test` via `make check`; end-to-end in the `node` package + CLI-contract tests in `cmd/soulnode`
**Target Platform**: darwin/linux (any Go platform)
**Project Type**: single binary + two library packages in this module
**Performance Goals**: `init && up` to admission-ready in well under SC-001's minute (the research rig did the whole ceremony in ~60 ms)
**Constraints**: constitution I (no domain logic; public tagged surfaces; no `replace`), II (operator mode + callout, no dev lane), III (ordinary loopback connections, no in-process transport), V (zero manual steps; init idempotence)
**Scale/Scope**: ~3 packages, ~900 LOC + tests; no new upstream asks

## Constitution Check

*Evaluated against `hq/00-GENESIS/constitution.md` v1.0.0.*

- **I. Composition, Not Invention — PASS with one tracked exception.**
  Only public surfaces are imported (`soulidentity/embed`,
  `soulidentity/client`, nats libraries); no `internal/` reach, no
  `replace`. The exception: soulidentity has no tag yet, so the pin is a
  pseudo-version of its main — reproducible and public, but not the
  article's "tagged release". Recorded in Complexity Tracking; the
  roadmap already tracks tag-flipping as the standing external
  dependency.
- **II. Same Shape as Any Deployment — PASS.** Operator mode, auth
  callout, sentinel + token admission — the exact shape the research
  measured; no dev-only lane exists or is configurable.
- **III. One Process, Planes by Configuration — PASS.** Server and
  identity plane in one process; every connection is an ordinary NATS
  client connection to the loopback listener; the listener address is
  configuration (`config.json`), and no code path branches on
  loopback-ness.
- **IV. Research Gates Before Build Spends — PASS.** Phase 0 closed
  (episode 0002); this feature builds only what design 0001 §§3–6
  specify, against measured behavior.
- **V. First Boot Is the Product — PASS.** `init` is the ceremony:
  zero prompts, zero external binaries, idempotent re-run, refusal on
  damage, the one token printed once.
- **VI. All-Green Quality Gate — PASS by construction**: the e2e test
  rides `make test`; the suite needs no external NATS.

Post-Phase-1 re-check: no new violations introduced by the design below.

## Project Structure

### Documentation (this feature)

```text
specs/001-init-and-up/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md            # init/up: flags, output, exit codes
│   └── state-dir.md      # layout, modes, verification rules
├── checklists/requirements.md
└── tasks.md              # /speckit-tasks output
```

### Source Code (repository root)

```text
ceremony/
├── ceremony.go           # Generate: keys, account JWTs, users, curve seeds
├── state.go              # persist / Load / Verify the state directory
└── ceremony_test.go      # roundtrip, modes, damage detection, idempotence

node/
├── node.go               # Start(cfg) (*Node, error): server + identity plane
├── found.go              # the founding acts through the public client
└── node_test.go          # e2e: init → up → admission observations

cmd/soulnode/
├── main.go               # init / up / version; flag + env resolution
└── main_test.go          # CLI contract: token printed once, re-run no-op

Makefile                  # build gains bin/soulnode (cmd exists now)
go.mod                    # soulstream dep enters at M1.2, not here
```

**Structure Decision**: two public library packages + a thin cmd, the
sibling pattern. `ceremony` is pure (no server, no connections) so it
unit-tests instantly; `node` owns everything that talks; the cmd owns
flags, env, signals, and printing.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Constitution I's "tagged releases": soulidentity pinned at a pseudo-version of main | Upstream has no tag yet; the embed seam landed 2026-08-01 and SoulNode is its first consumer | Waiting for a tag gates Phase 1 on a release act (gates-not-calendars violation); a `replace` directive is flatly forbidden; the pseudo-version is reproducible, public, and flips to the tag the day it exists (roadmap-tracked) |

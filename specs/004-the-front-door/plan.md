# Implementation Plan: the front door

**Branch**: `004-the-front-door` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-the-front-door/spec.md`

## Summary

The door plane: upstream's `soulstream/node` library (`New(Config)` →
`Handler()`/`Close()`) runs inside `soulnode up` on its own loopback HTTP
listener — local mode, static bearers, bearer passthrough to the node's
own callout admission, sentinel from the state dir, audit into the
existing logger. `config.json` gains `planes.door {enabled, listen}`
(default enabled, `127.0.0.1:8080`). `init` and `up` print the door URL.
The e2e drives a real MCP client (the go-sdk streamable transport, the
upstream dial pattern): founding token → session + tools + `whoami` names
the owner; garbage bearer → no session; disabled arm byte-identical to
Phase 1.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: `soulstream v0.6.0 → v0.7.0` (tagged);
`github.com/impire-io/soulstream/node` at a pseudo-version (nested
module, untagged — fourth tracked pin; consumable since soulstream
journey 0010 dropped its replace); `modelcontextprotocol/go-sdk` (test
dial, already indirect)
**Storage**: none new — SC-003 pins the inventory unchanged
**Testing**: node e2e (MCP client against the composition) + disabled arm
**Constraints**: constitution I (the pool/admission logic stays
upstream — SoulNode only wires), II (door admission IS the realm's
callout), III (second plane block; loopback bind; fail-loud), V (init
still two commands; the printed block gains the door URL)
**Scale/Scope**: ~+120 LOC + tests

## Constitution Check

- **I — PASS** (fourth pin in the tracked class). The door's domain
  logic (pool, principal discovery, corpse eviction, custody rules) is
  upstream's landed 018; SoulNode adds configuration and lifecycle only.
- **II — PASS**: the bearer is passed through to the same callout that
  admits every client; `whoami` is server-asserted.
- **III — PASS**: `planes.door` block; loopback-only bind; named
  refusals; disabled arm supported.
- **IV — PASS**: the Phase 2 gate (upstream 018 + consumable surface)
  is met and cited.
- **V/VI — PASS** as before.

## Project Structure

```text
ceremony/state.go      # planes.door {enabled, listen}; State.DoorEnabled/DoorListen
ceremony/ceremony.go   # defaults (enabled, 127.0.0.1:8080)
node/node.go           # startDoor: pre-flight bind, door.New, http.Server on a
                       #   held listener, DoorURL(); Stop: Shutdown + Close
cmd/soulnode/main.go   # init prints door URL; up prints "front door serving"
node/node_test.go      # TestFrontDoor (MCP client e2e) + disabled arm
specs/004-the-front-door/contracts/config.md   # the grown block
```

**Structure Decision**: the door is a `node`-package plane like memory;
the HTTP listener is held by SoulNode (pre-bound, so tests get real
ports and conflicts refuse early) with upstream's handler mounted.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Constitution I's "tagged releases": `soulstream/node` pinned at a pseudo-version | The nested module has no `node/vX` tag yet (upstream release act) | Same class as the other three pins; flips on upstream's first node tag |

# Tasks: the front door

**Input**: `/specs/004-the-front-door/` (plan, contracts/config.md)
**Tests**: mandatory.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 `go.mod`: soulstream → v0.7.0; add `soulstream/node` at its
      pseudo-version; tidy (archivist-against-v0.7.0 compatibility rides
      the gate).

## Phase 2: Foundational (ceremony/config)

- [X] T002 `ceremony/`: `planes.door {enabled, listen}` per
      contracts/config.md; `State.DoorEnabled/DoorListen`; defaults
      (true, `127.0.0.1:8080`); loopback check on the door listen;
      roundtrip + disabled tests.

## Phase 3: User Story 1 — the door plane (US1)

- [X] T003 [US1] `node/node.go`: `startDoor` — pre-flight bind (held
      listener, so conflicts refuse named and tests get real ports),
      `door.New(door.Config{Listen, Realm, NATSURL: n.url,
      SentinelPath, Logger})`, `http.Server.Serve(listener)` goroutine,
      `DoorURL()`; `Stop`: HTTP shutdown + door Close, before the NATS
      teardown.
- [X] T004 [US1] `cmd/soulnode/main.go`: init's founding block and up's
      serving lines name the door URL when enabled.
- [X] T005 [US1] `node/node_test.go` `TestFrontDoor`: MCP client
      (go-sdk streamable transport, bearer round-tripper — the upstream
      dial pattern) with the founding token → session forms, tools
      non-empty, `soulstream_whoami` names the owner persona; garbage
      bearer → no session; state-dir inventory unchanged (SC-001/003).
      Disabled arm: no HTTP listener, Phase 1 observations intact
      (SC-002).

## Phase 4: Polish & Landing

- [X] T006 Full gate green (SC-004).
- [X] T007 Landing duties: journey 0006; roadmap Phase 2 (local mode
      done; public mode named, soulfold-gated); design 0001 §8 → as-built;
      README/CLAUDE status; quickstart line for pointing a client; spec
      Status → implemented; tasks checked.

## Dependencies

T001 → T002 → T003 → (T004, T005) → T006 → T007

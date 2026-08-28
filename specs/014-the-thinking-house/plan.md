# Implementation Plan: The thinking house

**Branch**: `014-the-thinking-house` | **Date**: 2026-08-28 | **Spec**: [spec.md](spec.md)

## Summary

Two opt-in plane blocks in `config.json`, two planes in `node`, one KV
catalogue, three verbs. The dispatcher plane runs soulstream-workloads'
`dispatcher.Dispatcher` over the house's own realm on a runtime-minted
identity, filling its two seams — `ConnectAgent` with a persona-scope
`mint.ephemeral`, `EngineFor` with the wrap-in-the-house preset plus the
inference lane. The inference plane runs soulstream-inference's `door.Door`
and its in-process instances, authorising exactly the keys the dispatcher
issued and routing through the catalogue. Everything composed is upstream
public surface.

## Constitution Check

- **Composition, not invention — PASS**: `dispatcher.Dispatcher`,
  `fleet.Submit`, `wrap.Preset`, `door.Door`, `instance.Serve`,
  `client.Descriptor`, `adapter/standin`, `adapter/anthropic`,
  `siclient.MintEphemeral`, `siclient.SecretGet` — all public packages. The
  house wires and refuses; it decides no inference policy.
- **One process, planes by configuration — PASS**: both planes are config
  blocks on ordinary connections to the same substrate; absent means absent.
- **Same shape as any deployment — PASS**: the served agent's credential is
  a real callout-shaped persona-scope mint, enforced by the server's own
  template expansion; there is no local-only lane.
- **Clean breaks pre-v1 — PASS**: nothing existing changes shape. Realms
  founded before this feature run both planes with no migration, because
  neither plane needs a founding artifact.
- **Gates all green — `make check`**, plus the new gates at `-race -count=3`.

## Project Structure

```text
specs/014-the-thinking-house/    # spec.md, plan.md, tasks.md
ceremony/
├── ceremony.go        # State fields for both planes; MintPlaneUser
└── state.go           # the two config blocks, load defaults, Verify refusals
node/
├── dispatcherplane.go # NEW: the plane — identity, placements topic,
│                      #   ConnectAgent, EngineFor, drain
├── inferenceplane.go  # NEW: the door, the key registry, the instances
├── catalogue.go       # NEW: the realm KV catalogue (ensure, put, list, get)
├── node.go            # start/stop wiring, the two accessors
└── thinkinghouse_test.go # NEW: TestM15ThinkingHouse and its arms
cmd/soulstream/
├── agent.go           # NEW: `agent submit`
├── model.go           # NEW: `model set|ls`
├── provider.go        # NEW: `provider set`
└── main.go            # usage + verb wiring + the two new endpoint lines
```

## Key decisions

- **The engine credential is the persona scope** (spec.md's finding). The
  narrow lanes cannot materialise a topic; the engine's every wake does.
- **The catalogue is a realm KV** read fresh per resolution. A watch is an
  [O]; a silent cache would make re-pointing a name a lie.
- **Door keys are per serve, keyed by persona.** Issuing replaces (and so
  revokes) the persona's previous key; stopping the plane revokes all.
- **`EngineFor` cannot see the declaration** — its seam carries the persona
  alone. The plane re-reads the placement topic to find the persona's
  declaration and its `inference` block. Recorded as a finding for design
  0007 §5, with `EngineFor(ctx, declaration.Declaration)` as the candidate.
- **The virtual name reaches the door through `ANTHROPIC_MODEL`**, not a
  `{{MODEL}}` template variable — closing design 0007 §3's last question the
  way the harness's own environment already works.

## Complexity Tracking

`soulstream provider set` is one verb beyond the brief. Without it nothing
can write the plane principal's D36 tree — the store is caller-own by
construction (identity D36), and the plane's identity is minted at runtime,
so no operator holds it. A configured anthropic instance would be
unloadable. The verb is the smallest thing that makes FR-008 reachable.
</content>

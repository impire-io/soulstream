# Implementation Plan: Re-baselining (Rollup), Manifest Baselines & Archived

**Branch**: `007-rollup` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-rollup/spec.md`

## Summary

Rollup folds a topic's baseline + tail into one new baseline published with
`Nats-Rollup: sub` (JetStream's atomic per-subject purge — permitted since 001, the
stream was provisioned `AllowRollup: true`) and guarded by
`jetstream.WithExpectLastSequencePerSubject(lastConsumedSeq)` so a concurrent write
rejects the attempt wholesale. The baseline payload grows two additive fields: `baked`
(the conversation folded in: contributions, attachments, lifecycle) beside the opaque
workbench `state`, and `manifest` (object-store reference + digest) replacing inline
state above the 128 KB threshold. The `apply` fold learns to seed from `baked` and to
start the frontier from the payload's frontier, so replay of new-baseline+tail equals
old-baseline+full-tail. `archived` joins the lifecycle as terminal (refuse writes,
final rollup with bounded retry). Baked view elements inherit the baseline op's
verification status. Clients: CLI `rollup` + `archive`, close compacts via a new
`Handle.Close`, MCP gains a `soulstream_rollup_topic` tool (10th). Zero new
dependencies; view structs gain proper lowercase JSON tags because baked state makes
their JSON a wire format.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire/soulstream`)
**Primary Dependencies**: existing only — `nats.go/jetstream` already exposes
`MsgRollup`/`MsgRollupSubject` and `WithExpectLastSequencePerSubject`; object store
already used by attachments; digest via the existing `topic.VerifyDigest` format
**Storage**: the existing `SOULSTREAM` stream (rollup is a publish, not a config
change) and `soulstream-objects` bucket for manifest state objects
(`baseline/<path>/<baseline-op-id>`)
**Testing**: `go test ./...`; fold/round-trip logic serverless; rollup/race/manifest
tests on the embedded server (`internal/natstest`)
**Target Platform**: unchanged (library + CLI + MCP stdio)
**Project Type**: existing library + two thin clients
**Performance Goals**: post-rollup cold read = 1 message + (tail since); rollup cost =
one materialise + one publish (+ one object put above threshold)
**Constraints**: baseline stays exactly one message; the race guard is the only
concurrency control (no locks/leases anywhere); un-compacted topics behave exactly as
today; wire changes are additive payload fields only
**Scale/Scope**: dogfood scale — topics of hundreds of ops, states up to a few MB

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First** — PASS. Compaction *is* a NATS primitive
  (`Nats-Rollup: sub`); race safety *is* a NATS primitive
  (`Nats-Expected-Last-Subject-Sequence`, explicitly named by the constitution as a
  capability to prefer); oversized state rides the existing object store. No janitor,
  no coordinator, no new infrastructure. Minimum server version: per-subject rollup
  and expected-last-subject-sequence are long-stable (< 2.12); target unchanged.
- **II. Smallest Viable Implementation** — PASS. Additive payload fields, no new op
  types (`baseline` and `life.transition` already exist; `archived` is a new *value*),
  one manifest object rather than speculative multi-chunk machinery (the schema keeps
  a list so growth is additive), no periodic triggers, no dormant automation, no blob
  GC beyond happy-path superseded-object deletion, one new MCP tool.
- **III. ELI5 Documentation** — PASS. New `docs/rollup.md` (gluing this week's sticky
  notes into a fresh first page), `docs/lifecycle.md` gains archived (the bound and
  shelved notebook), `docs/materialisation.md`/`docs/topic.md`/`docs/cli.md`/
  `docs/mcp.md` updated in-story.

## Project Structure

### Documentation (this feature)

```text
specs/007-rollup/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── library.md       # new/changed Go API surface
│   └── wire.md          # baseline payload forms, manifest object, race guard
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
topic/
├── vocab.go             # BaselinePayload gains Baked + Manifest; Archived lifecycle;
│                        # BakedState type
├── view.go              # lowercase json tags on view structs (baked state is wire now);
│                        # apply seeds from baked state + payload frontier; terminal rule
├── rollup.go            # NEW Handle.Rollup: materialise → build payload (inline or
│                        # manifest) → publish with rollup header + sequence guard;
│                        # ErrRollupLost, ErrNothingToCompact; superseded-object cleanup
├── lifecycle.go         # Archived defined; Handle.Close (transition + attempt),
│                        # Handle.Archive (transition + bounded-retry rollup)
├── post.go              # archived refusal on every write path (closed keeps warning)
├── wire.go              # publishOp variant taking extra NATS headers + publish opts
├── verify.go            # annotateView: baked elements inherit the baseline op's status
├── materialise.go / follow.go  # unchanged flow; fold changes live in apply
└── (tests: rollup_test.go, manifest_test.go, archive_test.go, + updates)

internal/cli/
├── cli.go / commands.go # rollup + archive commands; close uses Handle.Close
└── (tests)

internal/mcpserver/
├── server.go / tools.go # soulstream_rollup_topic (10th tool); closeTopic → Handle.Close
└── (tests)

docs/
├── rollup.md            # NEW (ELI5)
└── (updates: lifecycle.md, materialisation.md, topic.md, cli.md, mcp.md, README.md)
```

**Structure Decision**: everything lands in the existing `topic` package — rollup is
op-log mechanics, exactly its charter. No new packages: the manifest object I/O reuses
the attachment code's object-store access pattern, and clients only gain thin command/
tool wrappers.

## Complexity Tracking

No violations. One deliberate simplification to note: the design doc speaks of
"chunks" for manifest baselines; this cycle writes the state as **one** object (the
object store already chunks internally at its own granularity, so client-side
multi-chunk adds no crash-safety and no capability today). The manifest schema carries
a chunk-name *list* so multi-chunk becomes an additive change if a real need appears.

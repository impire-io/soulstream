# Implementation Plan: Memory Convention & Exhibits

**Branch**: `015-memory` | **Date**: 2026-07-25 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-memory/spec.md`

## Summary

The realm forgets by design (rollup compacts op tails); remembering becomes something
personas do for each other. This feature ships the convention and the library surface:
a scatter/gather memory channel (`memory.query` / `memory.answer` / `memory.fetch` /
`memory.exhibit` on `SOULSTREAM.SVC.MEMORY`, cloned from the proven 008 discovery triad),
portable self-authenticating **exhibits** (verbatim wire capture in a strict-decode JSON
document, verified offline against the author's validated chain), asker-side **citation
grading** by actually checking (materialise + `ContainsOp` resolver → fact / unverifiable;
explicit fetch upgrades to fact-with-provenance / testimony), and a public **witness
surface** (`RespondMemory` with independently optional query/fetch capabilities and a
declared `coverage_from`). The first archivist is deliberately NOT here: it lives in a
separate repository under impire-io, built only on these public surfaces — tests in this
repo play that role to prove the contract sufficient (SC-005). CLI gains a `memory`
command group (query / fetch / exhibit / verify — verify fully offline); MCP gains
`soulstream_memory_query` and `soulstream_memory_fetch` (23 tools total). No new streams,
no provisioning change, nothing retained: memory traffic rides the uncaptured `SVC.>`
space by construction.

All technical unknowns are resolved in [research.md](research.md) (D1–D9).

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`)
**Primary Dependencies**: `nats.go` v1.52 + `nats.go/jetstream`, `synadia-io/orbit.go/natscontext`, `gowebpki/jcs`, `google/uuid`; test-only embedded `nats-server/v2` via `internal/natstest`
**Storage**: none added — existing JetStream stream/KV/object store read-only for this feature; exhibits are files the *user* keeps
**Testing**: `go test ./...` (embedded JetStream for NATS-touching packages; pure packages server-free), quality gate `make check` (fmt+tidy+build+test+lint), zero skips
**Target Platform**: same as library — macOS/Linux/Windows, NGS R1-safe
**Project Type**: library + two thin clients (CLI, MCP) in one Go module
**Performance Goals**: query completes at deadline + negligible overhead (SC-001); grading ≤ one materialise per distinct cited topic per query; exhibit capture = one bounded scan of one topic's subjects
**Constraints**: `record`/`identity` import NO NATS (exhibit type + parsing live in `record`); pure logic (grading, exhibit verify, resolver) unit-tests with no server; canonical binding for all memory service traffic = `MEMORY`, never the inbox; deadline default 3s clamped [100ms, 30s]; ≤100 answers gathered per query; NATS payload ceiling 1MiB (NGS) respected by construction (ops rode the stream already)
**Scale/Scope**: single-realm scatter/gather; witnesses O(few); answers capped at 100/query

## Constitution Check

*GATE: passed pre-research; re-evaluated post-design — still passing.*

- **I. NATS-Native First** ✅ — Core request/reply over the uncaptured `SVC.>` subject
  space (transient by construction since 014); ordered-consumer reads of existing streams
  for capture/grading; no new streams, KV, or stores; no external infrastructure (no
  index, no database — explicitly excluded by the spec). Newer server features evaluated:
  direct/multi-get can't fetch by msg-id (last-per-subject semantics), so exhibit capture
  is a bounded ordered scan of one topic's subjects — a NATS primitive, not custom
  machinery (research D5). Minimum server version: unchanged standing target (2.12+);
  nothing new is relied upon.
- **II. Smallest Viable Implementation** ✅ — Growth is vocabulary over the existing
  wire (`memory.*` types), not machinery: the scatter/gather is the 008 triad reused; the
  exhibit is the wire form captured verbatim (no second serialization); the verdict enum
  is the existing `SigStatus`; the witness surface is two nilable funcs (no plugin layer);
  no store, no ranking, no reputation, no `memory serve` command, no archivist — cut or
  relocated to the separate repo. Every addition maps to an acceptance scenario.
- **III. ELI5 Documentation** ✅ — Two new pages ship in-change: `docs/memory.md`
  ("asking the whole class what they remember") and `docs/exhibits.md` ("a sealed note
  anyone can check"), plus touch-ups where discovery docs mention the SVC space. Each
  user story's task list carries its docs task.

## Project Structure

### Documentation (this feature)

```text
specs/015-memory/
├── plan.md              # This file
├── spec.md              # Feature specification (clarified)
├── research.md          # D1–D9 decisions
├── data-model.md        # Entities: wire payloads, exhibit doc, grades, witness
├── quickstart.md        # End-to-end walkthrough (query, exhibit, offline verify)
├── contracts/
│   ├── library.md       # Public Go API contract (asker, witness, exhibit, resolver)
│   └── wire.md          # Subjects, op types, payload schemas, exhibit format
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks output (not created by /speckit-plan)
```

### Source Code (repository root)

```text
record/
├── exhibit.go           # Exhibit type: strict-decode JSON, verbatim wire capture, Record()
└── exhibit_test.go      # pure — round-trip, strict decode, tamper detection inputs

topic/
├── subjects.go          # + SvcMemorySubject, ServiceMemory const
├── vocab.go             # + TypeMemoryQuery/Answer/Fetch/Exhibit + payload structs
├── memory.go            # MemoryQuery (asker), FetchExhibit (live-first), RespondMemory (witness)
├── memory_test.go       # embedded-server: query/answer/fetch round-trips, caps, deadlines, SC-004 compaction recall, SC-005 outsider witness, SC-006 zero residue
├── exhibit.go           # CaptureExhibit (ordered scan), VerifyExhibit (wraps VerifyRecord), ErrOpNotLive
├── exhibit_test.go      # embedded-server capture + pure verify paths
├── resolve.go           # (mt *MaterializedTopic) ContainsOp — pure
└── resolve_test.go      # pure — live ids, baked ids, edits/work/timeline ids, negatives

internal/cli/
├── cli.go               # + memory case in Run switch + usage
├── memory_cmd.go        # cmdMemory: query / fetch / exhibit / verify (verify = offline, no connect)
└── memory_cmd_test.go   # incl. offline-verify-with-broken-config test (013 lesson)

internal/mcpserver/
├── server.go            # + 2 mcp.AddTool registrations (23 total)
├── memory_tools.go      # soulstream_memory_query, soulstream_memory_fetch
└── memory_tools_test.go

docs/
├── memory.md            # NEW — collective memory, grades, coverage (ELI5)
└── exhibits.md          # NEW — self-authenticating evidence (ELI5)
```

**Structure Decision**: Same single-module layout as every prior feature. The exhibit
type goes in `record` (pure, NO NATS — it is a record concern); scatter/gather, capture,
and verification wrappers go in `topic` beside their discovery/verify precedents; the
resolver is a pure view method; clients follow the one-file-per-command-group /
one-file-per-tool-group conventions.

## Complexity Tracking

No Constitution Check violations — table intentionally empty.

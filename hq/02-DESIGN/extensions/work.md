# Extension: Work — Artefacts, Work Items, Execution, Sandboxes

*Optional vocabularies over ordinary topics. The design home for the ROADMAP's work stages: how "personas work ON something" grows from attachments to shared execution environments without ever adding privileged machinery.*

---

Core already makes a topic a workbench: state (the baseline) plus operations ([../core/03-topics.md](../core/03-topics.md)). This extension names the growth path from "topic with attachments" to "shared execution environment" — five stages, each an additive vocabulary over ordinary op-logs, each usable without the next, each gated by felt need rather than anticipation ([../ROADMAP.md](../ROADMAP.md)).

One principle governs every stage, and it is the reason sandboxes can be last without being lost:

> **The artefact's authoritative home is the topic** — object store for the bytes, ops for the history, baseline for the current state. Everything else, including a sandbox, is a *view* of that artefact or a *site* where operations on it are performed. Anything worth keeping flows back into the topic as ops. This is what keeps every runtime replaceable and every artefact's history complete.

## Stage 1 — Versioned artefacts

A document is a topic-anchored attachment revised whole-file: each revision is an `attachment.add` anchored to its predecessor's op-ID. An **artefact** is the resulting lineage — a named chain of revisions with full authorship history, current tip by projection rule (same supersession mechanic as `edit`). No new machinery; immediately useful.

## Stage 2 — Work items

A work-tracking vocabulary over ordinary ops: `work.open`, `work.claim`, `work.done`, `work.abandon` — tasks are conversations with status, attached evidence, and an owner. Claiming is the interesting one, and it follows the house coordination rule: any persona may publish `work.claim`; when two race, **the first claim in stream order wins** and later claims are void by projection — deterministic, no lock service, no arbiter. An abandoned or timed-out claim (deterministic idle rule, like `dormant`) reopens the item.

## Stage 3 — Live co-editing

Character/block-level operations on shared documents, merged by eg-walker — the machinery is already specified in core; this stage is where its cost is actually paid. Gate, stated hard: only when stage-1 whole-file versioning demonstrably chafes in real use. A team revising documents a few times an hour does not need character-level merge; a team pair-writing does.

## Stage 4 — Executable workloads

Long-running jobs personas start and observe. The shape is stage 2 plus a **runner**: a persona (ordinary credentials, typically `operated_by` someone) that claims execution-flavoured work items, runs them wherever it lives, streams progress as ops, and attaches results back into the topic. Execution is thus visible, attributable, and replayable like everything else. A runner is a persona that does things — not a service tier.

## Stage 5 — Sandboxes

The workbench made physical: a shared environment with a filesystem and processes, where artefacts can be touched by ordinary tools and personas can work side by side.

The substrate's stance is precise, not dismissive:

- **What a sandbox adds** is exactly two things: a *filesystem/process view* of topic artefacts (so existing tools — editors, compilers, shells — can operate on them) and an *execution site* with shared visible state. Both are runtime concerns.
- **What a sandbox does not own**: the artefact, its history, or its authoritative state. A sandbox is populated *from* the topic (artefacts checked out onto the bench) and its outputs flow *back* as ops (`attachment.add`, baselines, results). A sandbox that dies loses scratch state, never history.
- **What the substrate carries** is the coordination: a sandbox session is itself an op-log — `sandbox.open`, join/leave, intents and claims (stage-2 mechanics), notable state changes, `result.attach`. Who was on the bench, who did what, what came off it: all in the topic, all attributable.

The runtime itself — isolation, filesystems, process supervision — lives outside the substrate and is deliberately designed *last*, against a working stage 4, because a coordination vocabulary invented before a real runtime exists would be speculation. Deferred is not dismissed: stages 1–4 deliver the artefact-centred collaboration; stage 5 adds the room it happens in.

## Why this is an extension

Every stage is additive vocabulary plus, at most, personas with habits (a runner, a sandbox host). Nothing here touches the wire format, the stream, or the acid-test list in the [README](../README.md) — which is exactly the test: if a stage ever needs core changes beyond a vocabulary, the design is wrong.

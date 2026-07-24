<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.1.0 (MINOR — new section added)
Modified principles: none (I–III unchanged in substance)
Added sections:
  - The Working Agreement (Anti-Drift): teach-back gate, evidence-class tags,
    recorded reversal conditions, adversarial pass on direction changes
Relocation:
  - Canonical copy moved to hq/00-GENESIS/constitution.md; .specify/memory/
    constitution.md is now a relative symlink to it, so spec-kit's Constitution
    Check reads these articles directly (no forked second copy).
Templates:
  - ✅ .specify/templates/plan-template.md — Constitution Check gate still reads
    Principles I–III; no wording depended on the moved path
  - ✅ .specify/templates/tasks-template.md — unaffected
  - ✅ .specify/templates/spec-template.md — unaffected
Follow-up TODOs: none
-->

# Soulstream Constitution

The canonical copy of this file lives at `hq/00-GENESIS/constitution.md`;
`.specify/memory/constitution.md` is a symlink to it, so every spec-kit plan's
Constitution Check reads these articles. Decisions are held against this file
and [`vision.md`](vision.md) — see the decision test in [`README.md`](README.md).

## Core Principles

### I. NATS-Native First

NATS is the platform, not a dependency. A working soulstream is a NATS server with
JetStream, a stream, credentials, and the protocol — nothing else.

- Every capability MUST be implemented with built-in NATS and JetStream primitives
  (streams, consumers, KV, Object Store, subject hierarchies, headers) before any
  custom mechanism is considered.
- Designs MUST evaluate current NATS server features before proposing custom code.
  This explicitly includes the newer server capabilities: atomic batch publishing,
  batched direct get (multi-get), message scheduling, per-message TTLs, and
  optimistic concurrency via `Nats-Expected-Last-Subject-Sequence`.
- Infrastructure that duplicates a NATS capability — databases, coordinators, API
  tiers, external queues — is prohibited. If NATS can express it, NATS MUST express it.

**Rationale**: The protocol's premise is that the "what is needed" list stays short.
Every custom component added beside NATS is a component that can fail, drift, or
require operation independently of the stream that is the system of record.

### II. Smallest Viable Implementation

- Every feature MUST be the smallest implementation that satisfies its specification.
  Anything not required by an acceptance scenario is cut or deferred.
- Speculative generality is prohibited: no configuration options, abstraction layers,
  or plugin points added "for later". Add them when a concrete need exists.
- Growth MUST be expressed as new vocabulary over the existing log, never as new
  machinery. If a design addition does not survive the "what is needed for a working
  soulstream" list staying short, it goes in `extensions/` or it goes nowhere.
- Scope creep is a review blocker, not a style concern. Reviewers MUST reject
  additions that exceed the spec, however well-built.

**Rationale**: The original idea was once buried under its own elaborations. Keeping
each change minimal is how the core stays answerable to "what is needed" and how
extensions stay genuinely optional.

### III. Documentation Is a First-Class Citizen (ELI5)

- The `docs/` folder MUST explain every concept in the system simply enough for a
  five-year-old to follow: plain words, one concept per page, an everyday analogy
  before any technical detail.
- No feature is complete until its concepts are documented. Docs ship in the same
  change as the behavior they describe — documentation is a task inside every story,
  never a polish phase afterthought.
- Stale documentation is a bug with the same severity as a failing test and MUST be
  fixed before merge.
- Plain words beat invented terms (the new-term test: if the plain word works, use
  it). New terminology MUST carry meaning a plain word cannot.

**Rationale**: A protocol lives or dies by whether newcomers — human or AI — can
understand it. If a concept cannot be explained simply, that is a signal the concept
itself is too complicated.

## The Working Agreement (Anti-Drift)

Adopted 2026-07-24 with the hq adoption (journey
[0002](../04-JOURNEY/0002-adopting-the-hq-way.md)) to guard a specific failure
mode: a fluent counterpart steering the maintainer on a load-bearing design
call he cannot independently check in the moment, without either party
intending it. Applies to every load-bearing decision — a protocol change, a
scope call, a criterion, or a public claim.

1. **Teach-back as a gate.** No load-bearing direction decision is recorded
   until the maintainer can restate the argument for it in his own words. If
   he can't, the decision isn't ready — the deficit is in the explanation, not
   the listener.
2. **Claims carry their evidence class.** Every load-bearing claim is tagged
   **[measured]** (a reading in the repo: a test, a demonstrated NATS
   behavior, a benchmark), **[mechanism-argument]** (a reasoned case from how
   NATS or the protocol works, attackable by reasoning), or **[judgment]**
   (taste or preference). Only measured closes a debate.
3. **Decisions record the reversal condition.** Every direction decision gets a
   "what would change our minds" line written *when the decision is made* (the
   journey episode template requires it), so a future reversal is a clean,
   anticipated turn instead of drift.
4. **Adversarial pass on direction changes.** For decisions that change the
   protocol's shape or a core boundary, the other side is argued at full
   strength before the decision — the maintainer never sees only the most
   convincing case.

## Technology Constraints

- **NATS server**: target a modern release (2.12+ as of ratification) so batch
  publishing, multi-get, and message scheduling are available. Any feature relying on
  a specific server capability MUST state its minimum server version in its plan.
- **Persistence**: JetStream only (streams, KV, Object Store). No external databases.
- **Coordination**: deterministic rules, idempotent operations, and optimistic
  concurrency. Elections and consensus rounds are banned in the protocol.
- **Clients**: official, maintained NATS client libraries only. Deprecated clients
  (e.g., `nats.ws`) MUST NOT be used.

## Development Workflow & Quality Gates

- Work follows the spec-driven flow: specification → plan → tasks → implementation.
  No implementation begins without a spec and plan for the feature. Research (open
  questions that precede a spec) runs the `hq/01-RESEARCH/` lifecycle instead and
  never enters spec-kit — see [`how-we-work.md`](how-we-work.md).
- Every plan MUST pass the Constitution Check gate before research and again after
  design. Violations are either removed or justified in Complexity Tracking.
- Before merge, everything MUST be green: `make fmt && make test && make lint` — all
  tests pass (none skipped), linting clean, formatting applied, artifacts build. The
  test target includes the hq structural lint (`internal/hqlint`).
- Every user story's task list MUST include its `docs/` task (Principle III).
- Every landed feature, concluded research topic, or load-bearing decision gets a
  numbered episode in `hq/04-JOURNEY/` in the same change (the journey duty).
- Commits are signed.

## Governance

- This constitution supersedes all other practices for Soulstream. On conflict with
  README, CLAUDE.md, or any template, the constitution wins.
- **Amendments**: made by editing this file (typically via `/speckit-constitution`),
  and MUST include an updated Sync Impact Report, a version bump, and a journey
  episode recording the why and the reversal condition.
- **Versioning policy** (semantic versioning):
  - MAJOR — backward-incompatible governance changes: removing or redefining a principle.
  - MINOR — a new principle or section, or materially expanded guidance.
  - PATCH — clarifications, wording, and non-semantic refinements.
- **Compliance review**: every plan's Constitution Check enforces Principles I–III;
  every review verifies the change is NATS-native, minimal, and documented. Complexity
  MUST be justified or removed.

**Version**: 1.1.0 | **Ratified**: 2026-07-12 | **Last Amended**: 2026-07-24

*Amendment history:*
- *1.1.0 (2026-07-24)* — added The Working Agreement (Anti-Drift) and moved the
  canonical copy into `hq/00-GENESIS/`; recorded in journey
  [0002](../04-JOURNEY/0002-adopting-the-hq-way.md).
- *1.0.0 (2026-07-12)* — initial ratification (Principles I–III).

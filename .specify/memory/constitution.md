<!--
Sync Impact Report
==================
Version change: (template, unversioned) → 1.0.0 (initial ratification)
Modified principles: n/a — all template placeholders replaced with concrete text
Added sections:
  - Core Principles: I. NATS-Native First; II. Smallest Viable Implementation;
    III. Documentation Is a First-Class Citizen (ELI5)
  - Technology Constraints
  - Development Workflow & Quality Gates
  - Governance
Removed sections:
  - Two unused placeholder principle slots (template offered five; project defines three)
Templates:
  - ✅ .specify/templates/plan-template.md — Constitution Check gate filled with concrete
    principle-derived gates
  - ✅ .specify/templates/tasks-template.md — documentation made a per-story task type,
    not polish-only
  - ✅ .specify/templates/spec-template.md — no changes required (technology-agnostic,
    already aligned with minimal-scope and measurable-outcome requirements)
  - ⚠ .specify/templates/commands/ — directory does not exist; nothing to update
Follow-up TODOs: none
-->

# Soulstream Constitution

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
  No implementation begins without a spec and plan for the feature.
- Every plan MUST pass the Constitution Check gate before research and again after
  design. Violations are either removed or justified in Complexity Tracking.
- Before merge, everything MUST be green: all tests pass (none skipped), linting
  clean, formatting applied, artifacts build.
- Every user story's task list MUST include its `docs/` task (Principle III).
- Commits are signed.

## Governance

- This constitution supersedes all other practices for Soulstream. On conflict with
  README, CLAUDE.md, or any template, the constitution wins.
- **Amendments**: made by editing this file (typically via `/speckit-constitution`),
  and MUST include an updated Sync Impact Report and a version bump.
- **Versioning policy** (semantic versioning):
  - MAJOR — backward-incompatible governance changes: removing or redefining a principle.
  - MINOR — a new principle or section, or materially expanded guidance.
  - PATCH — clarifications, wording, and non-semantic refinements.
- **Compliance review**: every plan's Constitution Check enforces Principles I–III;
  every review verifies the change is NATS-native, minimal, and documented. Complexity
  MUST be justified or removed.

**Version**: 1.0.0 | **Ratified**: 2026-07-12 | **Last Amended**: 2026-07-12

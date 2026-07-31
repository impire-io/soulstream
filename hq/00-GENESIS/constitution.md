# SoulNode Constitution

The canonical copy of this file lives at `hq/00-GENESIS/constitution.md`; once
the project enters implementation, `.specify/memory/constitution.md` is a
symlink to it, so every spec-kit plan's Constitution Check reads these
articles. Decisions are held against this file and [`vision.md`](vision.md) —
see the decision test in [`README.md`](README.md).

## Core Principles

### I. Composition, Not Invention (NON-NEGOTIABLE)

SoulNode contains no domain logic. Identity behavior lives in SoulIdentity,
runtime behavior in soulrealm, record behavior in soulstream; SoulNode wires
their public surfaces together and adds only what composition itself requires
(embedding, provisioning, lifecycle, the front door). Components are consumed
as tagged releases through public packages — never `internal/` paths, never
`replace` directives on main — and a SoulNode release names the component
versions it bundles. If a feature cannot be built without new domain behavior,
that behavior lands upstream first. This article does not relax for
convenience.

### II. Same Shape as Any Deployment

The embedded NATS server runs operator mode with auth-callout admission,
exactly as a hosted deployment does. There is no dev-only auth lane, no
local-only bypass, and no admission shortcut the rest of the ecosystem lacks.
A persona on a SoulNode is admitted, scoped, and attributed identically to a
persona on hosted infrastructure — so what SoulNode proves locally holds
hosted, and vice versa. Divergence between the two shapes is a bug, never a
feature.

### III. One Process, Workloads Apart

Everything SoulNode runs lives in the one process — embedded server, identity
plane, memory, runtime, front door — connected in-process, with no sidecar
daemons required. Workloads are the exception and always run outside the
process, through soulrealm's isolation backends. A workload failure never
takes the node down; a node-component failure is surfaced and named, never
silent. Anything that cannot run in-process is a named limitation with its
reason recorded, not a quiet extra daemon.

### IV. Research Gates Before Build Spends

SoulNode is composition under constraint, and the constraints are measured
before they are built against. Every build milestone names the research gate
it depends on (a proven in-process behavior, a public surface that exists, a
ceremony enumerated end-to-end), and no machinery is built ahead of its gate.
Speculation about composition is research, recorded in `01-RESEARCH/`; it is
not code.

### V. First Boot Is the Product

`soulnode init` performs the entire ceremony — operator, accounts, signing
keys, sentinel, vault first key, buckets — with zero manual key steps, and
`soulnode up` reaches a connectable realm on a fresh machine in minutes. The
distance from download to working realm is a measured, guarded number.
First-boot regressions are release blockers; a manual step added to the
ceremony is a constitution violation, not a documentation task.

### VI. All-Green Quality Gate

Done means the full gate is green with nothing skipped: `make fmt && make test
&& make lint` (which includes the hq structural lint, `internal/hqlint`).
Tests that need no NATS server run without one; anything touching NATS uses an
in-process server or fake transport so the suite has no external dependency.
Sign every commit. Never commit `.claude/settings.local.json`. Hook or gate
failures are blocking — fixed before anything else continues.

## The Working Agreement (Anti-Drift)

Inherited from the sibling projects' hard-won practice, and applied here from
day one. Applies to every load-bearing decision.

1. **Teach-back as a gate.** No load-bearing direction decision is recorded
   until the maintainer can restate the argument for it in his own words. If
   he can't, the decision isn't ready — the deficit is in the explanation,
   not the listener.
2. **Claims carry their evidence class.** Every load-bearing claim is tagged
   **[measured]** (a reading in the repo), **[mechanism-argument]** (a
   reasoned case, attackable by reasoning), or **[judgment]**. Only measured
   closes a debate.
3. **Decisions record the reversal condition.** Every direction decision gets
   a "what would change our minds" line written *when the decision is made*
   (the journey episode template requires it), so a future reversal is a
   clean, anticipated turn instead of drift.
4. **Adversarial pass on direction changes.** For vision-level calls, the
   other side is argued at full strength before the decision — the maintainer
   never sees only the most convincing case.

## Development Workflow

Work flows through `hq/` as described in [`how-we-work.md`](how-we-work.md):
research (`01-RESEARCH/`, lifecycle active → graduated | abandoned) → design
(`02-DESIGN/`, functional specs explicit enough for `/speckit-specify`) →
implementation (the spec-kit flow specify → plan → tasks → implement on a
numbered feature branch, tracked in `03-IMPLEMENTATION/roadmap.md`) → journey
(`04-JOURNEY/`, one numbered episode per landed feature, concluded research
topic, or load-bearing decision). Research never goes through spec-kit;
designs always do. Every behavioral change propagates into the design docs it
touches.

## Governance

This constitution supersedes all other practices. An amendment requires: the
explicit textual change, a semantic version bump (MAJOR: article removed or
redefined; MINOR: article added or materially extended; PATCH: clarification),
a journey episode recording the why and the reversal condition, and
propagation into any spec-kit template that depends on the changed text.
Spec-kit plans verify compliance through the Constitution Check; reviews call
out violations rather than accommodate them.

**Version**: 0.1.0 (draft — ratifies when the first design graduates) |
**Drafted**: 2026-07-31

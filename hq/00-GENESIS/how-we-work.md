# How we work

The process companion to [`constitution.md`](constitution.md): the pipeline,
the lifecycles, the duties, and how all of it is enforced. [`../README.md`](../README.md)
holds the one-screen map.

## The pipeline

```
question ──/research-start──▶ 01-RESEARCH/<slug>/     (state: active)
                                   │
                     /research-graduate <slug>
                        │            │           │
                     design       artifact    abandoned
                        │            │           │
                        ▼            │           │
              02-DESIGN doc          │           │
                        │            ▼           ▼
               /speckit-specify   04-JOURNEY episode (always; folder removed)
                        │
                        ▼
        specs/NNN-*/ + code  ──landed──▶ /journey-log episode
                        │                      + ROADMAP.md updated
                        ▼
        design docs updated (behavioral changes propagate back)
```

Two hard boundaries:

- **Research never goes through spec-kit.** Spec-kit assumes you know what
  you're building; research exists to find out *whether* to build. Research uses
  the pre-registration method below, in `hq/01-RESEARCH/`.
- **Implementation always goes through spec-kit.** A design doc in
  `hq/02-DESIGN/` is written to be the argument to `/speckit-specify`; the
  generated plan's Constitution Check reads GENESIS through the
  `.specify/memory/constitution.md` symlink.

## Research (`01-RESEARCH/`)

One folder per topic, created with `/research-start <slug>`. The folder's
`README.md` (from [`../01-RESEARCH/TEMPLATE.md`](../01-RESEARCH/TEMPLATE.md))
carries: Title, State (`active | graduated | abandoned`), Abstract, the
Question, and **pre-registered bars** — the pass/fail criteria written *before*
any experiment runs. The folder's `JOURNEY.md` records the investigation as it
happens.

- **Method:** hypothesis → cheap discriminating experiment → verdict, one
  variable at a time. Experiment scripts live in the session scratchpad;
  conclusions, documents, and principled code changes land in git.
- **Always committed and pushed** — even work that will be abandoned. The point
  is a permanent trail; abandoned research keeps its full history in git after
  the folder is gone.
- **Ending:** `/research-graduate <slug> --to design|artifact|abandoned`
  composes the topic's journey into the next-numbered `04-JOURNEY/` episode
  (verdict, evidence tags, reversal condition included), creates or updates the
  design doc when the outcome is a design, and removes the topic folder in every
  case. An abandoned topic is a *result*, recorded with the same care as a
  success.

## Design (`02-DESIGN/`)

The normative design: [`core/`](../02-DESIGN/core/) is Soulstream itself
(protocol, identity, topics); [`extensions/`](../02-DESIGN/extensions/) are
optional conventions a realm may run none of and still be a working soulstream.
Documents are written functional-level — explicit enough that `/speckit-specify`
can turn one into a spec without guessing: the capability, its seams, its
configuration surface, its acceptance criteria. Every behavioral change made
during implementation propagates back into the design docs it touches — the docs
describe the system as it *is*. A new capability that isn't yet decided starts as
research, not as a design doc.

## Implementation (`03-IMPLEMENTATION/` + `specs/`)

[`ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md) is the live plan: what gets built
and in what order, behind which gate. Features run the spec-kit flow
(`/speckit-specify` → clarify → plan → tasks → implement) on a numbered feature
branch; `specs/NNN-*/` artifacts freeze when the feature lands. Landing a feature
means: gate green, roadmap updated, journey episode written, design docs
propagated — in the same merge. Frozen `specs/` bodies are point-in-time
artifacts and are not rewritten after the fact; their `Status` field is updated
to record shipping.

## Journey (`04-JOURNEY/`)

The append-only log: one numbered episode (`NNNN-slug.md`) per landed feature,
concluded research topic, or load-bearing decision — written with `/journey-log`
(or `/research-graduate`, which writes it for research). The
[`TEMPLATE.md`](../04-JOURNEY/TEMPLATE.md) requires: what happened with the
honest numbers, what was refuted or reversed, evidence-class tags on
load-bearing claims, and a **Reversal condition** line. `README.md` carries the
preamble, the episode index, and the "Where things stand" summary — both
refreshed with every episode.

## The working agreement (anti-drift)

The four correctives are constitution articles (see The Working Agreement
there); this is how they run day to day:

- **When to teach-back:** any decision that changes the protocol's shape, a
  scope, a criterion, or a public claim. The assistant asks for the
  restatement; the decision is recorded only after it survives.
- **Tagging:** write `[measured]` / `[mechanism-argument]` / `[judgment]` inline
  where the claim is made — in conversation, in episodes, in design docs. A
  demonstrated NATS behavior or a passing test is `[measured]`; a reasoned case
  from how the protocol works is `[mechanism-argument]`. If a debate is being
  closed by anything other than `[measured]`, stop and say so.
- **Reversal conditions:** phrased as observable evidence, not vibes. Written at
  decision time, never retrofitted.
- **Adversarial pass:** for calls that change the protocol's shape or a core
  boundary, the assistant argues the other side at full strength *before* the
  decision.

## Enforcement (how this stays true without willpower)

1. **The constitution symlink.** `.specify/memory/constitution.md` →
   `hq/00-GENESIS/constitution.md`, so every spec-kit plan is checked against
   GENESIS mechanically (a dangling link would let spec-kit re-copy a forked
   template over it).
2. **The structural lint.** `internal/hqlint` rides the standard gate under
   `go test ./...` (locally and in CI): hq layout, research-state legality,
   episode numbering and required fields, index completeness, symlink health,
   and that relative links inside `hq/` resolve.
3. **The skills.** `/research-start`, `/research-graduate`, `/journey-log` make
   the transitions one command each, so the right order is the easy order. They
   stage explicit paths, commit signed, and never push — pushing stays a human
   act.
4. **Orientation.** Root `CLAUDE.md` and `AGENTS.md` point every session here
   first.

## Quality gates (the non-negotiables, in one place)

- Gate: `make fmt && make test && make lint` — all green, nothing skipped,
  before any "done" and before every commit. `make test` (`go test ./...`)
  includes the hq structural lint (`internal/hqlint`).
- Keep pure logic (folds, chain validation, discovery match/merge, curator
  judgment) separate from NATS I/O so it unit-tests with no server.
- Sign every commit. Never commit `.claude/settings.local.json`.
- NATS-native first (constitution I) and smallest viable implementation
  (constitution II) apply to every change, product or research.

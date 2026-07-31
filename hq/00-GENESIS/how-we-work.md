# How we work

The process companion to [`constitution.md`](constitution.md): the pipeline,
the lifecycles, the duties, and how all of it is enforced. `hq/README.md`
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
              02-DESIGN/NNNN-*.md    │           │
                        │            ▼           ▼
               /speckit-specify   04-JOURNEY episode (always; folder removed)
                        │
                        ▼
        specs/NNN-*/ + code  ──landed──▶ /journey-log episode
                        │                      + roadmap.md updated
                        ▼
        design docs updated (behavioral changes propagate back)
```

Two hard boundaries:

- **Research never goes through spec-kit.** Spec-kit assumes you know what
  you're building; research exists to find out whether and what to build.
  Research uses the pre-registration method below.
- **Implementation always goes through spec-kit.** A design doc in
  `02-DESIGN/` is written to be the argument to `/speckit-specify`; the
  generated plan's Constitution Check reads GENESIS through the
  `.specify/memory/constitution.md` symlink.

## Research (`01-RESEARCH/`)

One folder per topic, created with `/research-start <slug>`. The folder's
`README.md` (from [`../01-RESEARCH/TEMPLATE.md`](../01-RESEARCH/TEMPLATE.md))
carries: Title, State (`active | graduated | abandoned`), Abstract, the
Question, and **pre-registered bars** — the pass/fail criteria written
*before* any experiment runs. The folder's `JOURNEY.md` records the
investigation as it happens.

- **Method:** hypothesis → cheap discriminating experiment → verdict, one
  variable at a time (constitution IV). For a composition project a
  "discriminating experiment" is usually a rig: a throwaway consumer-position
  module that wires real component releases to an embedded server and
  measures the thing in doubt. Rig code lives in the topic folder or the
  session scratchpad; conclusions, decision records, and principled code land
  in git.
- **Always committed and pushed** — even work that will be abandoned. The
  point is a permanent trail; abandoned research keeps its full history in
  git after the folder is gone.
- **Ending:** `/research-graduate <slug> --to design|artifact|abandoned`
  composes the topic's journey into the next-numbered `04-JOURNEY/` episode
  (verdict, evidence tags, reversal condition included), creates or updates
  the design doc when the outcome is a design, and removes the topic folder in
  every case. An abandoned topic is a *result*, recorded with the same care as
  a success.

## Design (`02-DESIGN/`)

Numbered documents (`0001-…` onward) describing architecture and features at
the functional level — explicit enough that `/speckit-specify` can turn one
into a spec without guessing: the capability, its seams, its configuration
surface, its acceptance criteria. Every behavioral change made during
implementation propagates back into the design docs it touches — the docs
describe the system as it *is*.

## Implementation (`03-IMPLEMENTATION/` + `specs/`)

`roadmap.md` is the live plan: phases, milestones, exit criteria, and the
research gate each milestone depends on. No dates — gates, not calendars.
Features run the spec-kit flow (`/speckit-specify` → clarify → plan → tasks →
implement) on a numbered feature branch; `specs/NNN-*/` artifacts freeze when
the feature lands. Landing a feature means: gate green (constitution VI),
roadmap updated, journey episode written, design docs propagated — in the same
merge.

## Journey (`04-JOURNEY/`)

The append-only log: one numbered episode (`NNNN-slug.md`) per landed feature,
concluded research topic, or load-bearing decision — written with
`/journey-log` (or `/research-graduate`, which writes it for research). The
`TEMPLATE.md` requires: what happened with the honest numbers, what was
refuted or reversed, evidence-class tags on load-bearing claims, and a
**Reversal condition** line. `README.md` carries the preamble, the episode
index, and the "Where things stand" summary — both refreshed with every
episode.

## The working agreement (anti-drift)

The four correctives are constitution articles (see The Working Agreement
there); this is how they run day to day:

- **When to teach-back:** any decision that changes direction, scope, a
  criterion, or a public claim. The assistant asks for the restatement; the
  decision is recorded only after it survives.
- **Tagging:** write `[measured]` / `[mechanism-argument]` / `[judgment]`
  inline where the claim is made — in conversation, in episodes, in design
  docs. If a debate is being closed by anything other than `[measured]`, stop
  and say so.
- **Reversal conditions:** phrased as observable evidence ("in-process
  admission diverges from the wire rig on X", "N minutes from download to a
  connectable realm"), not vibes. Written at decision time, never
  retrofitted.
- **Adversarial pass:** for vision-level calls the assistant argues the other
  side at full strength *before* the decision — or the question goes to an
  outside reader.

## Project conventions

- Go module `github.com/impire-io/soulnode`.
- Components — soulstream, SoulIdentity, soulrealm — are consumed as tagged
  releases through their public packages only (constitution I): no
  `internal/` imports, no `replace` directives on main.
- Connect to external NATS via `github.com/synadia-io/orbit.go/natscontext`;
  in-process connections use the embedded server's client API. Use the modern
  `nats.go/jetstream` API. Never use `nats.ws` (deprecated).
- Keep pure logic (ceremony planning, config resolution, wiring decisions)
  separate from NATS and process I/O so it unit-tests with no server.

## Quality gates (the non-negotiables, in one place)

- Gate: `make fmt && make test && make lint` — all green, nothing skipped,
  before any "done".
- Tests requiring NATS use an in-process server or fake transport; the suite
  has no external dependency.
- Sign every commit. Never commit `.claude/settings.local.json`.
- Composition-not-invention (constitution I), same-shape (constitution II),
  and honest measurement (the working agreement) apply to every change.

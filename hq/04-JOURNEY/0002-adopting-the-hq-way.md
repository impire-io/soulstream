# Episode 0002 — Adopting the hq way: the process gets a constitution (2026-07-24)

Not a build or a measurement — a working-structure decision, made at the owner's
call (wholesale adoption of the PRA "hq" way approved up front) and recorded here
with its reversal condition. Soulstream's process artifacts had grown organically:
a constitution living at `.specify/memory/constitution.md` (spec-kit's copy, three
principles, no anti-drift agreement), a `rationale.md`, a design set, and a
`ROADMAP.md` that had drifted behind reality — with no numbered journal and
nothing mechanically holding decisions against the vision.

## What happened

Everything about *how Soulstream is run* now lives under `hq/`, modeled on the PRA
layout. **GENESIS** is the fixed point: [`vision.md`](../00-GENESIS/vision.md)
(what Soulstream is and refuses to become — protocol not platform, no coordinator,
no bot API), [`constitution.md`](../00-GENESIS/constitution.md) **bumped 1.0.0 →
1.1.0** to add The Working Agreement (teach-back, evidence-class tags with only
`[measured]` closing a debate, recorded reversal conditions, adversarial pass on
protocol-shape changes), and [`how-we-work.md`](../00-GENESIS/how-we-work.md)
(pipeline, research lifecycle, the journey duty, the gate). The canonical
constitution moved into GENESIS and `.specify/memory/constitution.md` became a
relative symlink to it, so spec-kit's Constitution Check reads the real articles
rather than a forked copy [mechanism-argument].

Two areas that did not exist were added — **01-RESEARCH** (for the questions that
precede a spec; a `/research-start` → `/research-graduate` lifecycle that never
touches spec-kit) and **04-JOURNEY** (this numbered journal) — each with a README
and a TEMPLATE, plus a one-screen [`hq/README.md`](../README.md) map and READMEs
for the pre-existing 02-DESIGN and 03-IMPLEMENTATION areas. Staleness was
refreshed against git reality: the [`ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md)
gained a "Where we are" block mapping shipped features to versions, and the
shipped specs' `Status` fields moved from `Draft` to `Shipped (vX)` (bodies left
frozen). The three lifecycle skills (`/research-start`, `/research-graduate`,
`/journey-log`) were ported with paths and the gate adapted to this repo. The
structure is enforced by **`internal/hqlint`**, a test-only Go package that rides
`go test ./...` (so `make test`, `make check`, and CI pick it up unchanged):
required areas/READMEs/GENESIS files/templates, legal research states, contiguous
indexed episodes each carrying a `Reversal condition:` line, symlink health, and
resolving relative links inside `hq/`. It was verified to **fail on a planted
violation and pass once removed** [measured]. Along the way, ten broken relative
links in the living hq areas — files that moved in the 2026-07-11 restructure but
kept old link targets — were repaired; the link check skips the frozen
`99-ARCHIVE/` subtree by design.

The owner also settled the repo's license — **MIT** — closing a gap: the code had
shipped through v0.3.1 with no `LICENSE` file. A `LICENSE` (MIT, Copyright (c)
2026 Daan Gerits) was added and the README now states it, making concrete the OSS
stance the vision already assumes ("run it yourself; the substrate is the
product") [judgment].

## What was refuted or reversed

The task brief that launched this work carried a stale audit: it believed the repo
was at v0.2.0 with `014-persona-accountability` still in flight. Git said
otherwise — `014` is merged and tagged **`v0.3.0` (2026-07-23)**, with a **`v0.3.1`**
registry fix on top (2026-07-24) [measured]. The judgment call, made and flagged
rather than papered over: **build the journey and the spec statuses to reality,
not to the brief.** So the founding retrospective
([0001](0001-genesis-and-the-reference-library.md)) runs genesis → v0.3.1 (not → v0.2.0), and
`014`'s spec `Status` reads `Shipped (v0.3.0)` (not left as `Draft`). Reflecting
reality is the whole point of the honesty discipline being adopted; leaving a
shipped feature marked in-flight would have contradicted it on day one.

## What it taught / what it opened

The enforcement is mechanical, not aspirational [mechanism-argument]: the
constitution symlink wires GENESIS into every future spec-kit plan, and the
structural lint rides the same gate the code already runs. The lifecycle is now
one command per transition, so the right order is the easy order. The full gate
(`make fmt && make test && make lint`) is green with the lint included
[measured].

Reversal condition: if, two to three features from now, hq lags reality —
missing episodes, a ROADMAP that has drifted again, illegal research states
despite the lint, or the team routing research through spec-kit anyway — the
structure is failing its purpose and we fold back rather than maintain a facade.

Trail: the `hq-alignment` branch series — GENESIS (constitution v1.1.0 + symlink,
vision, how-we-work, decision test); the hq map, research + journey scaffolds,
and the design/implementation indexes; the ROADMAP refresh + spec `Status`
flips; the ported lifecycle skills; `internal/hqlint`; the orientation pointers
(AGENTS.md, CLAUDE.md, README); the MIT `LICENSE`; and this journey seed.
Reference: PRA's `hq/`, `LICENSE`, and `tests/test_hq_structure.py`.

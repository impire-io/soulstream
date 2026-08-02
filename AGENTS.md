# Agent guide for Soulstream

Durable instructions for any coding agent working in this repository. The full
rules live in `../soul-hq/00-GENESIS/`; this file is the orientation and the
non-negotiables.

## Orientation (read in this order)

1. `../soul-hq/00-GENESIS/` — [`vision.md`](../soul-hq/00-GENESIS/vision.md) (what Soulstream is
   and refuses to become — protocol not platform, no coordinator, no bot API),
   [`constitution.md`](../soul-hq/00-GENESIS/constitution.md) (the articles no change may
   violate, plus the anti-drift working agreement), and
   [`how-we-work.md`](../soul-hq/00-GENESIS/how-we-work.md) (pipeline, research
   lifecycle, the journey duty). Decisions are held against these.
2. `../soul-hq/04-JOURNEY/README.md` — where things stand + the episode index: what was
   built, what was refuted, and why things are the way they are.
3. The current feature plan — pointed to by the SPECKIT block in `CLAUDE.md`
   (tech stack, structure, commands).
4. `../soul-hq/02-DESIGN/soulstream/README.md` — the design map: `core/` (the protocol) and
   `extensions/` (optional conventions); `../soul-hq/00-GENESIS/rationale.md` for the
   reasons behind the non-obvious calls.

## Non-negotiables (constitution articles, in brief)

- **Quality gate before "done"** (all green, none skipped, before every commit):
  `make fmt && make test && make lint`; the hq structural lint rides the soul-hq gate (make test there).
- **NATS-native first** (I): every capability is built on NATS/JetStream
  primitives before any custom mechanism; no databases, coordinators, API tiers,
  or external queues that duplicate what NATS already does.
- **Smallest viable implementation** (II): the smallest thing that satisfies the
  spec; growth is new vocabulary over the log, never new machinery; scope creep
  is a review blocker.
- **Documentation is first-class, ELI5** (III): every concept explained plainly
  in `docs/`, shipped in the same change as the behavior; stale docs are bugs.
- **The working agreement** (anti-drift): load-bearing claims carry an evidence
  class (`[measured]` / `[mechanism-argument]` / `[judgment]`, only measured
  closes a debate); direction decisions record their reversal condition when
  made; sign every commit; never commit `.claude/settings.local.json`.

## The flow

- **Research** runs through `/research-start` → investigate →
  `/research-graduate` (`../soul-hq/01-RESEARCH/`; never through spec-kit).
- **Features** run the spec-kit flow (`/speckit-specify` → clarify → plan →
  tasks → implement) on a numbered branch, and land with the ROADMAP update, the
  journey episode, and design-doc propagation in the same merge. Frozen
  `specs/NNN-*/` bodies are not rewritten after shipping; their `Status` field
  records the version.
- **The journey duty (required):** every landed feature, concluded
  investigation, or load-bearing decision gets a numbered episode in
  `../soul-hq/04-JOURNEY/` — `/journey-log` does this (template, index,
  where-things-stand, ROADMAP). Never push; pushing stays a human act.

# Agent guide for SoulNode

Durable instructions for any coding agent working in this repository. The full
rules live in `../soul-hq/00-GENESIS/`; this file is the orientation and the
non-negotiables.

## Orientation (read in this order)

1. `../soul-hq/00-GENESIS/` — [`vision.md`](../soul-hq/00-GENESIS/vision.md) (what SoulNode
   is and refuses to become — a distribution that wires, never a fourth home
   for domain logic, never a dev-mode fork of the ecosystem's auth),
   [`constitution.md`](../soul-hq/00-GENESIS/constitution.md) (the articles no change
   may violate, plus the anti-drift working agreement), and
   [`how-we-work.md`](../soul-hq/00-GENESIS/how-we-work.md) (pipeline, research
   lifecycle, the journey duty). Decisions are held against these.
2. `../soul-hq/04-JOURNEY/README.md` — where things stand + the episode index: what
   was built, what was refuted, and why things are the way they are.
3. `../soul-../soul-hq/03-IMPLEMENTATION/ROADMAP.md` — the live plan: phases, milestones, and
   the research gate each depends on.
4. `../soul-hq/02-DESIGN/soulnode/README.md` — the design map; 0001 (the composition) is
   the one to read before touching code.

## Non-negotiables (constitution articles, in brief)

- **Quality gate before "done"** (all green, none skipped, before every
  commit): `make fmt && make test && make lint`; the hq structural lint rides the soul-hq gate (make test there).
- **Composition, not invention** (I): no domain logic here; components come
  in as tagged releases through public packages — no `internal/` imports, no
  `replace` on main; new behavior lands upstream first.
- **Same shape as any deployment** (II): embedded NATS runs operator mode
  with auth-callout admission exactly as hosted; no local-only auth lane.
- **One process, planes by configuration** (III): enabled planes in one
  process on ordinary loopback NATS connections; repointing or disabling
  a plane is configuration. Workloads outside via soulrealm's backends.
- **The working agreement** (anti-drift): load-bearing claims carry an
  evidence class (`[measured]` / `[mechanism-argument]` / `[judgment]`, only
  measured closes a debate); direction decisions record their reversal
  condition when made; sign every commit; never commit
  `.claude/settings.local.json`.

## The flow

- **Research** runs through `/research-start` → investigate →
  `/research-graduate` (`../soul-hq/01-RESEARCH/`; never through spec-kit).
- **Features** run the spec-kit flow (`/speckit-specify` → clarify → plan →
  tasks → implement) on a numbered branch, and land with the roadmap update,
  the journey episode, and design-doc propagation in the same merge. Frozen
  `specs/NNN-*/` bodies are not rewritten after shipping.
- **The journey duty (required):** every landed feature, concluded
  investigation, or load-bearing decision gets a numbered episode in
  `../soul-hq/04-JOURNEY/` — `/journey-log` does this (template, index,
  where-things-stand, roadmap). Never push; pushing stays a human act.

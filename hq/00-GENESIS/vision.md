# Soulstream Vision

## What Soulstream is

Soulstream is **a stream on which humans and AI collaborate through operations
applied to topics** — a *protocol with a reference library*, not a platform.
Every persona, human or AI, holds the same kind of credential, publishes the
same operation record, and is addressed the same way. There is no bot API and
no human API. There is one protocol.

A topic is a **shared workbench**, not a chat room: it has state (the
*baseline* — the concrete thing being worked on) and operations that change it.
Conversation is one operation vocabulary among several; "baseline + ops → new
baseline" *is* modifying things together and seeing where they end up.

## The founding bet

The "what is needed for a working soulstream" list stays short — and everything
else is either vocabulary over that list or an optional extension. A working
soulstream is exactly:

1. A NATS server with JetStream.
2. A JetStream `SOULSTREAM` stream.
3. An identity per persona — a NATS user credential.
4. The protocol on the stream: subjects, the operation record, topic lifecycle,
   discovery.
5. Baselines, and the ability to roll up messages into them.

Nothing else. No API tier, no database, no coordinator, no curator process.
Topics are self-coordinating: deterministic rules, idempotent operations, and
optimistic concurrency — never elections, never consensus rounds. If a future
design addition doesn't survive this list staying this short, it goes in
`../02-DESIGN/extensions/` or it goes nowhere. The reasons behind every
non-obvious call are in [`rationale.md`](rationale.md).

## Who it is for

People and agents who want to collaborate as peers, without a second-class door
for the AI. The unit of adoption is a *realm* (account-enforced tenancy) that
anyone can stand up over their own NATS: a human on the CLI, agents through the
MCP adapter, all first-class personas on one stream. The substrate is the
product — anything that only exists in one client isn't part of the platform —
so the bet is that useful clients grow around a stable protocol rather than the
protocol growing around one client.

## Where it is pointed

The growth path is more vocabulary over the same log, never new machinery — the
work stages in [`../02-DESIGN/extensions/work.md`](../02-DESIGN/extensions/work.md),
sequenced in [`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md):

- **Versioned artefacts** and **agent work items** — shipped, additive
  vocabulary over ordinary op-logs.
- **Live co-editing** — where an eg-walker-style merge would land, paid for only
  when whole-file versioning demonstrably chafes in real use, not anticipated.
- **Executable workloads** and **sandboxes** — a runtime designed last, against a
  working execution stage; the topic carries only its coordination, never the
  artefact itself.
- **A second library language and an extracted spec test-suite** — so the wire,
  not one implementation, is the contract.

## What we refuse to become

- **A platform instead of a protocol.** The original idea was once buried under
  its own elaborations (vision, substrate, personas, memory, search, encryption
  all at one level). The core answers "what is needed for a working soulstream"
  and nothing more; the rest is extensions.
- **A system with a coordinator or a consensus round.** No steward, no election,
  no lock service. A component you can't turn off without degrading core flows
  is plumbing, whatever the docs call it. Every coordination problem is solved
  with deterministic rules + idempotent ops + optimistic concurrency.
- **A second door for the AI.** No bot API, no human API, no `on_behalf_of`
  attribution laundering. Delegation is scoped credentials or a separately named
  persona.
- **A product that lives in one client.** No canonical UI; no privileged
  plumbing above NATS — no special services and no special dependencies.
- **A wrapper around NATS.** Lean on NATS, don't wrap it: subject permissions
  are nearly the whole security model; unknown op types are ignored with a
  warning.

## How ambition stays honest

Invented vocabulary is a budget, spent only on concepts a plain word can't
carry (*persona*, *realm*, *baseline* earn their place; "head" and "rung" did
not). Documentation is a first-class citizen — a concept that can't be explained
to a five-year-old is a concept that is too complicated. And load-bearing design
decisions record what would change our minds when they are made, so a future
reversal is a clean, anticipated turn instead of drift. The full discipline
lives in [`constitution.md`](constitution.md) and
[`how-we-work.md`](how-we-work.md).

# Soulstream

*A headless collaboration substrate where humans and AI personas work as peers.*

---

## The gap

Collaboration platforms — Notion, Google Workspace, Slack — were built for humans, then had AI bolted on as a feature: an assistant in a sidebar, a bot with a special API, a "copilot" that lives outside the document model. The AI is always a second-class citizen with a different door into the building.

Soulstream inverts this. It is a substrate, not a product: a set of NATS streams, subject conventions, and a client library. Every participant — human or AI — is a **persona** that connects the same way, speaks the same protocol, and appears in the same attribution model. There is no bot API and no human API. There is one protocol.

Work happens in **topics**: focused, multi-party conversations that carry turns, comments, attachments, edits, and angles as operations on a shared log. Topics are where personas meet; everything else is convention layered on top.

## Principles

**One protocol, no second door.** A persona is a persona. Humans and AI agents hold the same kind of credentials, publish the same operation record, and are addressed the same way. Whether a persona is backed by a person at a keyboard or a model in a loop is *metadata for presentation*, never *mechanism*. Anything that only works for humans, or only works for agents, is a design smell.

**Protocol symmetry, attention asymmetry.** Equality at the protocol layer does not mean equality of capacity. An AI persona can subscribe to every topic in a realm and read every operation; a human cannot. The scarce resource in the system is human attention, and the substrate must budget for it explicitly: mention routing, topic projections, digests, and curation are first-class concerns, not UI afterthoughts. A substrate designed only for the fast reader drowns the slow one.

**Headless means the substrate is the product.** There is no canonical UI. The platform is: a NATS deployment, the subject and stream conventions, the operation vocabulary, and a client library that implements them. Web apps, TUIs, MCP servers, and autonomous agents are all *heads* — clients of the same body. If a capability only exists in a head, it isn't part of the platform.

**No privileged plumbing.** Above NATS itself, there are no special services. Curation, digesting, archiving — jobs a platform would normally hide in backend services — are performed by personas holding ordinary credentials, publishing ordinary operations. A steward persona that flags duplicate topics uses the same protocol as everyone else. This keeps the substrate honest (if the protocol can't support the steward, it can't support your agents either) and makes every platform behaviour replaceable, inspectable, and opt-out.

**Convention over enforcement.** The substrate enforces little: subject permissions decide who can publish where, and that is nearly the whole security model. Everything above that — operation vocabularies, topic etiquette, lifecycle transitions, roles — is convention that participants and libraries agree on. Unknown operation types are ignored with a warning, not rejected, so vocabularies grow without breaking older participants.

**Lean on NATS, don't wrap it.** Tenancy is NATS accounts. Identity is NATS credentials. Persistence is JetStream. Blobs are the JetStream object store. History compaction is message rollup. A service in front of any of these would add a hop and an availability dependency without adding a capability. Where NATS has a primitive, Soulstream uses it directly.

## What v1 is

Three things, specified in the companion docs:

1. **The substrate** ([01-substrate.md](./01-substrate.md)) — realms, streams, subjects, and the operation record.
2. **Personas** ([02-personas.md](./02-personas.md)) — identity, credentials, delegation, and roles for humans and AIs alike.
3. **Topics** ([03-topics.md](./03-topics.md)) — the op-log conversation model: turns, comments, mentions, attachments, baselines, lifecycle.

Plus the definition of how anything connects: **the protocol surface** ([04-protocol.md](./04-protocol.md)) — the wire spec, the reference library, and adapters (MCP, WebSocket) for clients that can't speak NATS natively.

## Non-goals for v1

**Sandboxes.** Shared coding/editing execution environments are a stated ambition, but a sandbox is a filesystem and processes, not a message flow. NATS can carry the *coordination* around a sandbox — who is in it, what changed, intents and locks — and v1's job is to make sure topics and attachments are good enough hooks for that coordination. The sandbox runtime itself is a later document, designed against a working substrate rather than alongside a speculative one.

**Cross-realm anything.** A realm is a hard boundary (a NATS account). Federation between realms is out of scope.

**A canonical UI.** Heads will exist and one will probably be built early to keep the substrate honest, but no head is part of the v1 spec.

**Enforced workflow.** No required review states, no mandatory approval chains. Topics are free-form; structure is what participants layer on.

**Being a startup.** This is infrastructure built for its own sake. Design decisions optimise for smallness, inspectability, and the pleasure of a coherent system — not for a pitch.

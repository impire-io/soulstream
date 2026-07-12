# Extension: The Reference Library and Adapters

*The reference implementation and the doors for clients that can't hold NATS credentials. Nothing here may be required — any NATS client speaking the core spec participates fully.*

---

"Connecting to Soulstream" means one of three progressively thicker layers, every layer bottoming out in the same NATS subjects.

## Layer 0 — the wire spec

The core docs themselves ([../core/](../core/01-protocol.md)). Any NATS client in any language can participate with no Soulstream code at all — publish a well-formed `turn.post` and you are collaborating. This layer is the compatibility contract: everything above it is convenience.

## Layer 1 — the reference library

One library that clients and agents embed. It owns:

- record construction (headers ↔ canonical record), idempotent publish via `Nats-Msg-Id`, retry-with-same-id;
- topic materialisation: baseline first (inline or manifest), tail replay, live subscription, eg-walker merge;
- the mechanical routines: mention parsing and notify-publishing, idle detection, periodic re-baselining with the expected-sequence guard;
- the vocabulary: typed operation constructors and projection rules (edit supersession, comment threading, attachment resolution);
- the local topic projection (replay of `SOULSTREAM.TOPICS.INFO.>`) and the `topic.discover` scatter/gather client and responder.

**Language choice:** two targets sharing a spec test-suite — **Go** for infrastructure-adjacent processes (adapters, CLI) and **TypeScript** for browser clients and the JS agent ecosystem. The spec tests (record golden files, merge scenarios, baseline round-trips, rollup races) are the actual source of truth, so a third implementation is a porting exercise, not archaeology.

## Layer 2 — adapters

Adapters exist for clients that cannot or should not hold NATS credentials directly. **An adapter is a credential custodian, not a privileged service**: it holds the credentials of the personas it fronts and translates a foreign protocol onto Layer 1 calls. Nothing an adapter does is impossible for a direct NATS client; adapters add reach, never capability.

### The MCP adapter (agents' door)

An MCP server exposing a realm to any MCP-speaking agent:

- **Tools**: `list_topics` (local projection + discover), `open_topic`, `post_turn`, `comment`, `attach`, `announce_topic`.
- **Resources**: topics as subscribable resources, so frameworks that poll resources get materialised state without understanding op-logs.
- Each connected session is bound to **one persona's credentials**, supplied at setup. The adapter never multiplexes identities within a session — attribution stays honest.

A pragmatic bow to reality: most 2026 agents speak MCP, not NATS. The design stance is that a serious resident agent should eventually hold credentials and speak Layer 0/1 natively — the MCP adapter is a ramp, not the destination.

### The WebSocket door (browsers)

Browsers speak NATS over WebSocket natively (nats.ws), so the thinnest browser story is Layer 1 in TypeScript over a WebSocket-enabled NATS listener — no gateway at all. A realm wanting short-lived browser tokens or cookie auth runs a small gateway that exchanges a web login for scoped, expiring NATS credentials. The gateway authenticates humans; it does not proxy traffic.

### Bridges (later, but shaped now)

Email-in, Slack-in, webhook-in: each is an adapter persona (`slack-bridge`, `operated_by` set) posting into topics under its own name, carrying provenance in `data`. Bridged content is attributed to the bridge, never impersonated as the human — the no-attribution-laundering rule ([../core/02-identity.md](../core/02-identity.md)). If a bridged human becomes a regular persona, they get a real persona.

## Presence (thin convention)

A persona may publish ephemeral state — currently-open topic, focus/away — for "who's here" affordances and attention-aware agents (defer a non-urgent mention while the human is looking; don't defer when they're away). Two rules make it safe:

- **Advisory by definition.** Nothing may *depend* on presence.
- **Ephemeral by construction.** Presence must not accumulate in the `SOULSTREAM` stream: publish outside the captured prefix (e.g. `SOULSTREAM_PRESENCE.<persona-id>` as plain core NATS — a different first token, so the stream never sees it) or use a per-subject-limited side stream. Presence in the op-log is a bug.

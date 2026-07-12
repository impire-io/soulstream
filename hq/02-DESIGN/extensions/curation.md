# Extension: Curation

*Optional. Keeping an active realm liveable — as a persona habit, never a protocol role.*

---

An active realm accumulates near-duplicate topics, drift, and noise. The core protocol deliberately has no component responsible for this: lifecycle is deterministic ops any persona posts, rollup is leaderless, discovery is self-serve ([../core/03-topics.md](../core/03-topics.md)). What remains is *judgment* — "these two topics are about the same thing," "this digest is what a mention-only reader needs" — and judgment belongs to personas.

## Curator personas

A realm that wants dedicated curation runs one or more **curator personas**: ordinary credentials, ordinary operations, zero protocol standing. A curator typically:

- subscribes to `SOULSTREAM.TOPICS.INFO.>` and `SOULSTREAM.TOPICS.OPS.>`, maintaining a high-quality topic projection;
- answers `topic.discover` queries from that projection, making it the *best* responder in the scatter/gather — but never the only one;
- flags likely duplicates with a comment in the newer topic;
- proposes closing or archiving long-dormant topics with a comment in place;
- publishes digests as a regular (system-tagged) topic for mention-only readers.

A curator **suggests, never enforces**. Merging, closing, archiving are personas' decisions, posted as ordinary ops. And because a curator is just a persona: run none (the realm still works — this is the difference from the earlier "steward" design, which the protocol quietly depended on for discovery), run one, run two competing ones, or replace it any time.

## Why this is an extension, not core

Earlier drafts had a "steward" described as an ordinary persona but load-bearing in practice: discovery pointed at "the steward's projection," lifecycle transitions were "suggested by the steward." That made a supposedly optional component a de-facto moving part. The current core removes the dependency rather than the label: everything a curator does must be something the protocol already works without. If a proposed curator behaviour can't be turned off without breaking a core flow, it belongs in core as a deterministic rule — or it doesn't belong at all.

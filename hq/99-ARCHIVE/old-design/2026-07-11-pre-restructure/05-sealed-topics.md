# Sealed Topics

*End-to-end encrypted topics: content unreadable by anyone but participants — including the operator.*

---

A **sealed** topic is a topic whose operation payloads are encrypted client-side, decryptable only by its participants. Sealing protects content against every non-participant: other realm personas, the steward, and — the reason encryption exists at all rather than subject permissions — the NATS operator and anyone with access to the server's disks.

Sealing is a **mode chosen at announcement time**, not a flag toggled later. A topic is born sealed or born open; converting between them is a new topic. (Retroactively sealing an open topic protects nothing — the history was already visible — and unsealing a sealed one is equivalent to publishing a decrypted copy, which participants can do deliberately if they mean to.)

## Threat model — what sealing does and does not protect

Protected: the payload of every operation, baseline contents, attachment blobs, and the topic's display name and subject matter.

**Not protected, by design — sealed is not hidden:**

- The topic's existence, its `topic_id`, and its position in the subject taxonomy.
- The record headers: `Nats-Msg-Id`, `Ss-Author`, `Ss-Parents`, `Ss-Ts`. These must stay cleartext or DAG merge dies. So the operator sees *who* wrote, *when*, *how often*, and in reply to *what* — full traffic analysis.
- Membership. Key distribution names the key holders; joins and leaves are visible as epoch changes.
- Mention notifications: `soulstream.mention.<name>` messages reveal that a persona was pinged about a sealed topic, even though the pinging content is sealed.

If metadata privacy is ever required, that is a different and much harder design (padding, mixing, decoy traffic) and is explicitly out of scope.

**Two standing caveats that no cipher fixes:**

1. **Key authenticity.** Participants learn each other's public keys from the persona registry — which the operator controls. An operator willing to substitute keys can MITM a sealed topic at creation. E2EE against the operator therefore *requires* out-of-band key verification: fingerprint comparison, keys signed by an external identity, or TOFU with pinning in every library. The substrate can carry fingerprints; only humans (or an external PKI) can verify them.
2. **Where keys live.** An agent persona's private key sits in a long-running process. If that process runs on operator-controlled infrastructure, sealing against the operator is theater for that participant. Sealed topics whose members include operator-hosted agents should be understood as sealed against *other personas and casual operator access*, not against a determined host.

## Persona keys

Each persona that participates in sealed topics publishes a long-term **X25519 public key** in its registry profile:

```json
{ "name": "architect", "kind": "agent",
  "sealing_key": { "x25519": "<base64>", "since": "2026-07-10T09:00:00Z" } }
```

Libraries pin the first key they see per persona (TOFU) and hard-fail on unannounced changes. Key rotation is announced by publishing the new key signed by the old one (`sealing_key.rotates` proof in the profile); anything else is treated as a possible substitution attack and surfaced loudly.

## Epoch keys

Each sealed topic has a symmetric **topic key** per **epoch**. Epoch 1 is created by the announcer; every membership change creates a new epoch. The epoch key is distributed *through the op-log itself* as a key-management operation:

```
Ss-Type:   sealed.epoch
Ss-Author: daan

{ "epoch":   4,
  "members": ["daan", "architect", "counsel"],
  "wrapped": {
    "daan":      "<epoch key sealed to daan's X25519 key>",
    "architect": "<...>",
    "counsel":   "<...>" } }
```

- **Join:** any current member publishes a new epoch whose `members` includes the newcomer, and separately hands them enough prior epoch keys to read as much history as the group intends (all, or none — an explicit choice per invite, "history visibility on join").
- **Leave / eject:** a member publishes a new epoch excluding the leaver. The leaver keeps everything they already saw — there is no retroactive revocation anywhere in cryptography — but reads nothing after the epoch bump.
- The `sealed.epoch` op is cleartext apart from the wrapped keys: membership is metadata (see threat model).

This is the sender-key/age-recipients pattern: simple, auditable, adequate for collaboration. It deliberately does **not** provide forward secrecy or post-compromise security within an epoch — a stolen epoch key reads that epoch. The named upgrade path, if a hosted multi-tenant future demands FS/PCS, is to replace this section with an MLS (RFC 9420) group per topic; every op shape here survives that swap (`sealed.epoch` becomes a carrier for MLS commit/welcome messages).

## Sealed operations

All content operations on a sealed topic share one cleartext type, so operation kinds don't leak — and with the record in headers, the payload is simply the raw ciphertext:

```
Nats-Msg-Id: <op-id>
Ss-Author:   architect
Ss-Parents:  <op-id>
Ss-Type:     sealed.op
Ss-Epoch:    4
Ss-Nonce:    <nonce>

<XChaCha20-Poly1305 ciphertext, binary>
```

The ciphertext decrypts to the **canonical op record** of a normal inner operation ([01-substrate.md](./01-substrate.md)) — `{"type": "comment.add", "data": {…}, …}` — the entire open-topic vocabulary from [03-topics.md](./03-topics.md), unchanged, one layer down. Projection, threading, and edit supersession run client-side after decryption.

**AAD binds ciphertext to context.** The AEAD's associated data is the cleartext record: `(realm, topic-path, Nats-Msg-Id, Ss-Author, Ss-Parents, Ss-Epoch)`. An operator can therefore not splice a ciphertext into another topic, under another author, or at another DAG position without decryption failing. This is what keeps cleartext headers and encrypted payloads from being a re-attribution hazard.

**Merge is unchanged.** Eg-walker orders by `id`/`parents`/`author` — all cleartext. Sealed and open topics use the identical merge path; decryption happens after ordering, not before.

## Announcements, baselines, attachments, mentions

- **Announcement:** `topic.announce` carries `"sealed": true`, cleartext `topic_id`, the epoch-1 `sealed.epoch` material, and *encrypted* `name` and `subject_matter` (sealed to epoch 1). The steward sees that a sealed topic exists and who is expected — not what it is about.
- **Baselines:** the writer materialises client-side and encrypts the baseline state with the current epoch key — inline ciphertext if it fits the threshold, encrypted chunks plus a manifest if not. The rollup message is a normal `baseline` op whose content is opaque; `frontier` stays cleartext (op-IDs are opaque). New members without full history keys materialise from the first baseline of an epoch they hold — the baseline is the natural history-visibility boundary on join.
- **Attachments:** encrypted client-side before `put`; `digest` covers the ciphertext; content type and size are inside the sealed inner op, with only a size *class* observable from the blob itself.
- **Mentions:** the inner op carries `mentions` as usual; the library still fires `mention.notify` so sealed topics don't silently fail to reach people, but the notification body is sealed to the mentioned persona's key. The *existence* of the notification leaks (threat model).

## Interaction with the rest of the design

- **Membership becomes real.** Open topics keep `expected` as a hint; sealed topics have an enforced member set — the key holders. This is the one sanctioned exception to "membership is not a gate," and it is enforced by mathematics rather than by a service, so the no-privileged-plumbing principle survives.
- **The steward goes blind, on purpose.** No duplicate detection, no digest content, no drift-flagging for sealed topics; it can still track lifecycle and staleness from metadata and propose archival. A sealed topic's participants are their own curators.
- **Defense in depth is still free.** A realm may *additionally* restrict subscribe permissions on sealed topics' subjects. Encryption protects content; permissions reduce metadata exposure to other personas. They compose; neither replaces the other.
- **Search cannot index sealed content** server-side. If sealed-topic search matters, it is a participant-side index held by a member (possibly an agent member — with the key-custody caveat above).

## Cost, stated plainly

Sealing turns key management into a participant responsibility, makes the steward useless for content-level curation, complicates joining (history-visibility decisions), and rests on out-of-band key verification that tooling can encourage but not perform. It should be the exception, not the default: an open realm with sealed exceptions, never a sealed-by-default realm — at that point the substrate's collaboration model is fighting its own privacy model.

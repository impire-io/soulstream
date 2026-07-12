# Research: Participation — Object Store & Notify

**Feature**: 003-participation | **Date**: 2026-07-12
**Method**: Object store API verified against `nats.go@v1.52.0/jetstream/object.go` source.

## 1. JetStream object store (attachments)

- **Decision**: bind with `js.ObjectStore(ctx, realm.ObjectBucket)`, store with
  `os.PutBytes(ctx, name, data) (*ObjectInfo, error)`, fetch with `os.GetBytes(ctx, name) ([]byte,
  error)`.
- **ObjectInfo**: embeds `ObjectMeta` (so `.Name` is promoted), plus `Size uint64` and `Digest
  string`. **Digest format is `"SHA-256=<base64url(sha256(content))>"`** (URL encoding, padded) —
  recorded on the `attachment.add` op and independently recomputable for verification (FR-009/015).
- **Names**: object names may contain slashes/dots (base64-encoded into the subject internally), so
  `attachments/<topic-path>/<uuid>` is valid (FR-007). Only the *bucket* name is restricted.
- **Missing object**: `jetstream.ErrObjectNotFound` (checked with errors.Is) → mapped to a clear
  not-found error (FR-016). Missing bucket: `ErrBucketNotFound` (should not occur — provisioned).
- **Rationale**: the object store is JetStream's own — single dependency (Constitution I). PutBytes/
  GetBytes cover the whole add/retrieve slice; streaming `Put`/`Get` are available but unneeded for
  MVP-size files.

## 2. Notifications on the stream

- **Decision**: publish `mention.notify` records (via the same `publishOp` used for topic ops, with
  `parents=nil`) to `SOULSTREAM.PERSONA.NOTIFY.<persona>`. The stream captures `SOULSTREAM.>`, so
  notifications persist and are consumable; `FollowInbox` reads them with an ordered consumer
  (DeliverAll) filtered to the persona's notify subject — the same pattern as topic `Follow`.
- **Rationale**: reuses the wire record and the ordered-consumer replay/live path; no new mechanism.
  A notification is just a record on a different subject.
- **Alternative rejected**: core NATS (non-JetStream) pub/sub for notifications — would drop
  notifications for an offline persona; JetStream persistence lets an agent catch up on wake.

## 3. Mention parsing (pure)

- **Decision**: `regexp` `@([a-z0-9]+(-[a-z0-9]+)*)` finds `@name` tokens matching the slug grammar
  by construction (no trailing hyphen, no uppercase); de-duplicate and length-check via
  `identity.ValidName`.
- **Rationale**: the regex encodes the grammar exactly, so `@Daan`/`@@`/`@ x` yield nothing and
  `@bookkeeper-agent!` yields `bookkeeper-agent` (SC-002). No persona-existence check (FR-006).

## 4. Digest verification (pure)

- **Decision**: `"SHA-256=" + base64.URLEncoding.EncodeToString(sha256.Sum256(data)[:])`, compared to
  the recorded digest. Matches the object store's own computation exactly.

No new third-party dependency is introduced by this cycle.

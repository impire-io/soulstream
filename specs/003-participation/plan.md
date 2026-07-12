# Implementation Plan: Participation — Mentions & Attachments

**Branch**: `003-participation` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)

## Summary

Extend the `topic` package with two additive conventions: **mentions** (parse `@name` from turn/
comment bodies, record them on the op, and publish a `mention.notify` to each mentioned persona's
`SOULSTREAM.PERSONA.NOTIFY.<persona>` inbox, which a persona can follow) and **attachments** (put a
blob in the realm's `soulstream-objects` store, publish an `attachment.add` referencing it by name +
digest + size + content type, surface it on the materialised view, and retrieve/verify it). No new
package, no new dependency, no wire-record change.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire/soulstream`).
**Primary Dependencies**: existing (`record`, `identity`, `realm`, `topic`, `nats.go/jetstream`,
`google/uuid`). Standard library `regexp`, `crypto/sha256`, `encoding/base64`.
**Object store API** (verified, research.md): `js.ObjectStore(ctx, realm.ObjectBucket)`,
`os.PutBytes(ctx, name, data) (*ObjectInfo, error)`, `os.GetBytes(ctx, name) ([]byte, error)`;
`ObjectInfo` embeds `ObjectMeta` and carries `Size uint64` and `Digest` = `"SHA-256=<base64url>"`;
missing object → `jetstream.ErrObjectNotFound`; object names may contain slashes.
**Storage**: JetStream stream (notify + attachment.add ops) and the object store (blobs) — both
already provisioned.
**Testing**: `go test`; unit tests for mention parsing and digest verification (pure, no server);
integration tests for notify delivery and attachment put/materialise/get against `internal/natstest`.
**Project Type**: single Go library; extends `topic`, adds `TypeMentionNotify`/`TypeAttachmentAdd`
vocabulary and an `Attachments` field on the view.
**Constraints**: no `attachment.remove` (day-2); no encryption; no mention digests/presence.

## Constitution Check

- **I. NATS-Native First** — ✅ PASS. Notifications ride the existing stream (`PERSONA.NOTIFY.>`);
  blobs use JetStream's own object store; no external service, no new infra. Digest is the object
  store's own SHA-256.
- **II. Smallest Viable Implementation** — ✅ PASS. Mentions/attachments are additive vocabulary +
  parsing; the notify inbox reuses the ordered-consumer pattern from `Follow`; no speculative
  options; deferrals (FR-017) explicit.
- **III. ELI5 Documentation** — ✅ PASS (planned). New `docs/`: *mentions & notifications* (tapping
  someone on the shoulder / a pigeonhole) and *attachments* (the shared filing cabinet the notebook
  points to). One per story cluster.

**Result**: PASS. Re-checked post-design: unchanged — no new dependency beyond stdlib helpers.

## Project Structure

```text
topic/
├── mention.go            # ParseMentions; PostTurn/AddComment fill mentions + fire notify
├── mention_test.go       # pure parse matrix (no server)
├── notify.go             # NotifySubject; Notification; publishNotify; FollowInbox (ordered consumer)
├── notify_test.go        # integration: mention → inbox
├── attachment.go         # Attach (PutBytes + attachment.add), GetAttachment, VerifyDigest
├── attachment_test.go    # integration: attach → materialise → get + verify
├── vocab.go              # (extend) TurnPayload/CommentPayload +Mentions; AttachmentPayload; NotifyPayload; new type consts
└── view.go               # (extend) MaterializedTopic +Attachments; apply() handles attachment.add + dangling

docs/
├── mentions.md
└── attachments.md
```

**Structure Decision**: Stay inside `topic` — participation is vocabulary over the same op-log, so a
new package would violate "smallest viable". The pure parts (mention parsing, digest verification)
are separate functions with no NATS import, unit-tested without a server; the wire parts (notify,
object store) are integration-tested.

## Key implementation notes

- **PostTurn/AddComment** now parse mentions into the payload, post, then publish notifies — existing
  002 tests still pass (payload gains an optional `mentions`, omitted when empty).
- **apply()** gains an `attachment.add` case (append to `Attachments`, count as content op); the
  dangling check extends to attachments; unknown types still warn.
- **FollowInbox** mirrors `Follow`: one ordered consumer (DeliverAll) on the notify subject, parse
  each `mention.notify`, call `onNotify`; cancellable via ctx.
- **Digest verification**: recompute `"SHA-256="+base64url(sha256(data))` and compare.

## Complexity Tracking

> No Constitution violations.

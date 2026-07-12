# Data Model: Participation

**Feature**: 003-participation | **Source**: [spec.md](./spec.md)

Additive vocabulary + two view fields. No database.

## New / extended op types

| Type | Subject | Payload |
|---|---|---|
| `mention.notify` | `SOULSTREAM.PERSONA.NOTIFY.<persona>` | `{topic, op_id, author}` |
| `attachment.add` | topic OPS | `{name, object, digest, size, content_type, anchor?}` |

## Extended payloads

- **TurnPayload** / **CommentPayload** gain `mentions []string` (`json:"mentions,omitempty"`), the
  distinct valid persona names parsed from the body.

## Entities

- **Mention**: a validated persona slug from a body; recorded in the op payload's `mentions`.
- **NotifyPayload**: `{Topic string, OpID string, Author string}` — the `mention.notify` payload.
- **Notification** (materialised for a follower): `{Topic, OpID, Author}` from a received
  `mention.notify`.
- **AttachmentPayload** (`attachment.add`): `{Name, Object, Digest, Size uint64, ContentType,
  Anchor string}`. `Object` is the store key `attachments/<topic-path>/<uuid>`; `Digest` is the
  object store's `SHA-256=…`.
- **Attachment** (materialised): the payload plus `OpID, Author, Timestamp, Dangling, StreamSeq`.

## View extension

`MaterializedTopic` gains `Attachments []Attachment`, ordered by stream sequence. An `attachment.add`
is a **content op** (moves proposed → active). An attachment whose `Anchor` is absent from the topic
is flagged `Dangling`, like a comment.

## Parsing & verification (pure)

- **ParseMentions(body) []string**: `@([a-z0-9]+(-[a-z0-9]+)*)`, de-duplicated, validated via
  `identity.ValidName`.
- **VerifyDigest(data, digest) bool**: `"SHA-256="+base64url(sha256(data)) == digest`.

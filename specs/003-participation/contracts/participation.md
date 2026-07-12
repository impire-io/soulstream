# Contract: participation additions to `topic`

## Mentions

```go
// ParseMentions returns the distinct, valid persona names @mentioned in body.
func ParseMentions(body string) []string

// PostTurn / AddComment (existing) now also: fill payload.mentions and, after publishing,
// publish one mention.notify per mentioned persona. Signatures unchanged.
func (h *Handle) PostTurn(ctx context.Context, body string) (opID string, err error)
func (h *Handle) AddComment(ctx context.Context, body, anchorOpID string) (opID string, err error)
```

## Notifications

```go
const TypeMentionNotify = "mention.notify"

// NotifySubject returns a persona's inbox subject.
func NotifySubject(persona string) string // "SOULSTREAM.PERSONA.NOTIFY." + persona

type Notification struct { Topic, OpID, Author string }

// FollowInbox subscribes to persona's notify subject and calls onNotify for each
// mention.notify (history then live), until ctx is cancelled.
func FollowInbox(ctx context.Context, c *realm.Client, persona string, onNotify func(Notification)) error
```

## Attachments

```go
const TypeAttachmentAdd = "attachment.add"

// Attach stores data in the object store under attachments/<path>/<uuid> and publishes an
// attachment.add referencing it. anchor may be "" (unanchored). Rejects an empty name.
func (h *Handle) Attach(ctx context.Context, name, contentType string, data []byte, anchor string) (opID string, err error)

// GetAttachment fetches an attachment's bytes by its object key. Not-found → a clear error.
func GetAttachment(ctx context.Context, c *realm.Client, object string) ([]byte, error)

// VerifyDigest reports whether data matches an object store digest ("SHA-256=<base64url>").
func VerifyDigest(data []byte, digest string) bool

type Attachment struct {
	OpID, Author         string
	Timestamp            time.Time
	Name, Object, Digest string
	Size                 uint64
	ContentType          string
	Anchor               string
	Dangling             bool
	StreamSeq            uint64
}
```

`MaterializedTopic` gains `Attachments []Attachment`.

## Guarantees (→ spec)

- Parse fills `mentions` (FR-001/002), notify per persona (FR-003), followable inbox (FR-004),
  self-notify (FR-005), no existence check (FR-006).
- Attach stores + references with digest/size (FR-007/008/009), optional anchor + dangling (FR-010),
  empty name rejected (FR-011), materialised list + active (FR-012/013).
- GetAttachment exact bytes (FR-014), VerifyDigest (FR-015), not-found error (FR-016).

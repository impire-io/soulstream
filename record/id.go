package record

import "github.com/google/uuid"

// NewID returns a fresh operation identity: a random UUIDv4 rendered lowercase and
// hyphenated (8-4-4-4-12). It is unique without coordination and doubles as the
// message's duplicate-detection identity (Nats-Msg-Id), so a retried publish of the
// same identity is de-duplicated by the server rather than creating a second
// operation.
func NewID() string {
	return uuid.NewString()
}

package topic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/record"
)

// publishOp builds an operation record for the client's persona (author, fresh op-id,
// timestamp, given parents), serialises the payload, and publishes it to subject.
// Stamping the author as the client's own persona is the write-side attribution
// guarantee: the library cannot post as another persona.
func publishOp(ctx context.Context, c *realm.Client, subject, opType string, payload any, parents []string) (opID string, err error) {
	author := c.Persona()
	if author == "" {
		return "", fmt.Errorf("topic: a persona is required to post (client has none)")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("topic: marshal %s payload: %w", opType, err)
	}

	rec := record.Record{
		ID:        record.NewID(),
		Author:    author,
		Parents:   parents,
		Type:      opType,
		Timestamp: time.Now().UTC(),
		Payload:   data,
	}

	// Sign when the client holds a key: the signature covers the canonical record —
	// the same bytes any reader can recompute from the wire form and the subject —
	// with the Signature field still empty (the canonical form omits an empty sig).
	if signer := c.Signer(); signer != nil {
		canonical, cerr := rec.Canonical(c.Realm(), canonicalBinding(subject))
		if cerr != nil {
			return "", fmt.Errorf("topic: canonicalise %s for signing: %w", opType, cerr)
		}
		rec.Signature = signer.Sign(canonical)
	}

	headers, body, err := rec.Build()
	if err != nil {
		return "", fmt.Errorf("topic: build %s record: %w", opType, err)
	}

	msg := &nats.Msg{Subject: subject, Header: nats.Header(headers), Data: body}
	if _, err := c.JetStream().PublishMsg(ctx, msg); err != nil {
		return "", fmt.Errorf("topic: publish %s: %w", opType, err)
	}
	return rec.ID, nil
}

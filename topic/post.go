package topic

import (
	"context"
	"log"
)

// warnf reports a non-fatal advisory. It is a package variable so tests can capture it.
var warnf = func(format string, args ...any) {
	log.Printf("soulstream/topic: "+format, args...)
}

// Post builds and publishes an operation to the topic's ops subject: it stamps the
// author (the client's persona), generates the op-id, and parents onto the frontier the
// handle has observed. It returns the new op-id and advances the handle's frontier to
// it. Posting to a topic the handle last saw as closed is warned, not blocked — closed
// is not-writable by convention.
func (h *Handle) Post(ctx context.Context, opType string, payload any) (string, error) {
	if h.lifecycle == Closed {
		warnf("posting %s to closed topic %s (closed is not-writable by convention)", opType, h.path)
	}
	opID, err := publishOp(ctx, h.client, OpsSubject(h.path), opType, payload, h.frontier)
	if err != nil {
		return "", err
	}
	h.frontier = []string{opID}
	return opID, nil
}

// PostTurn posts a turn.post — a contribution to the conversation.
func (h *Handle) PostTurn(ctx context.Context, body string) (string, error) {
	return h.Post(ctx, TypeTurnPost, TurnPayload{Body: body})
}

// AddComment posts a comment.add anchored to anchorOpID.
func (h *Handle) AddComment(ctx context.Context, body, anchorOpID string) (string, error) {
	return h.Post(ctx, TypeCommentAdd, CommentPayload{Body: body, Anchor: Anchor{Kind: "op", OpID: anchorOpID}})
}

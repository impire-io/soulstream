package topic

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Work items (stage 2 of the work extension): a work-tracking vocabulary over
// ordinary ops. A task is a conversation with status, evidence, and an owner —
// and the log, not an arbiter, decides who owns it: the first claim in stream
// order wins, later claims are void by projection.

// WorkOpenPayload is the work.open payload: a task opened in the topic. The op's
// ID becomes the item's identity.
type WorkOpenPayload struct {
	Title    string   `json:"title"`              // required; empty is malformed
	Body     string   `json:"body,omitempty"`     // optional prose; @mentions parsed
	Mentions []string `json:"mentions,omitempty"` // filled by the library, like turns
}

// WorkRefPayload is the payload of work.claim / work.done / work.abandon: a
// reference to the item (its work.open op) via the usual anchor convention.
type WorkRefPayload struct {
	Anchor *Anchor `json:"anchor"` // {kind:"op", op_id:<item id>}; missing is malformed
}

// WorkStatus is a work item's derived state.
type WorkStatus string

// Work item states. Done is terminal this cycle: reopening is a new item.
const (
	WorkOpen    WorkStatus = "open"
	WorkClaimed WorkStatus = "claimed"
	WorkDone    WorkStatus = "done"
)

// WorkEvent is one op that touched an item — including the ops that lost. A void
// event changed nothing: the state machine rejected it, but it stays visible.
type WorkEvent struct {
	OpID      string    `json:"op_id"`
	Kind      string    `json:"kind"` // "claim" | "done" | "abandon"
	Author    string    `json:"author"`
	Timestamp time.Time `json:"ts"`
	Void      bool      `json:"void,omitempty"`
	Sig       SigStatus `json:"sig,omitempty"`
	StreamSeq uint64    `json:"stream_seq,omitempty"` // 0 for baked events
}

// WorkItem is a task derived from the log: identity is the opening op's ID, the
// owner is the author of the winning claim, and the timeline is every claim,
// done, and abandon that referenced it — void ones included.
type WorkItem struct {
	ID        string      `json:"id"`
	Author    string      `json:"author"` // opener
	Timestamp time.Time   `json:"ts"`     // opened at
	Title     string      `json:"title"`
	Body      string      `json:"body,omitempty"`
	Mentions  []string    `json:"mentions,omitempty"`
	Status    WorkStatus  `json:"status"`
	Owner     string      `json:"owner,omitempty"` // cleared by abandon, kept by done
	Timeline  []WorkEvent `json:"timeline,omitempty"`
	Sig       SigStatus   `json:"sig,omitempty"`
	StreamSeq uint64      `json:"stream_seq,omitempty"` // 0 for baked items
}

// workEventKind maps a work op type to its timeline kind.
func workEventKind(opType string) string {
	return strings.TrimPrefix(opType, "work.")
}

// OpenWork opens a work item: title required, body optional. It parses @mentions
// from the body, records them on the op, and notifies each mentioned persona's
// inbox. The returned op-id is the item's identity.
func (h *Handle) OpenWork(ctx context.Context, title, body string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("topic: work item title must not be empty")
	}
	mentions := ParseMentions(body)
	opID, err := h.Post(ctx, TypeWorkOpen, WorkOpenPayload{Title: title, Body: body, Mentions: mentions})
	if err != nil {
		return "", err
	}
	return opID, h.notifyMentions(ctx, opID, mentions)
}

// ClaimWork publishes a claim on the item. Publishing cannot know whether it won —
// the log decides: materialise afterwards and read the item's owner.
func (h *Handle) ClaimWork(ctx context.Context, itemID string) (string, error) {
	return h.postWorkRef(ctx, TypeWorkClaim, itemID)
}

// CompleteWork publishes work.done for the item.
func (h *Handle) CompleteWork(ctx context.Context, itemID string) (string, error) {
	return h.postWorkRef(ctx, TypeWorkDone, itemID)
}

// AbandonWork publishes work.abandon for the item: a claimed item reopens for a
// fresh first-claim race.
func (h *Handle) AbandonWork(ctx context.Context, itemID string) (string, error) {
	return h.postWorkRef(ctx, TypeWorkAbandon, itemID)
}

func (h *Handle) postWorkRef(ctx context.Context, opType, itemID string) (string, error) {
	if strings.TrimSpace(itemID) == "" {
		return "", fmt.Errorf("topic: %s needs a work item id", opType)
	}
	return h.Post(ctx, opType, WorkRefPayload{Anchor: &Anchor{Kind: "op", OpID: itemID}})
}

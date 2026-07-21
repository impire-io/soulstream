package topic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/realm"
)

// Rollup outcome errors.
var (
	// ErrRollupLost means a concurrent write invalidated the attempt: something
	// landed on the topic after the tail this rollup consumed. Nothing changed;
	// retry if compaction still matters.
	ErrRollupLost = errors.New("topic: rollup lost the race (someone wrote concurrently); try again")
	// ErrNothingToCompact means the log is already just a baseline.
	ErrNothingToCompact = errors.New("topic: nothing to compact (the log is already just a baseline)")
	// ErrTopicArchived means the topic is archived — terminal, writes are refused.
	ErrTopicArchived = errors.New("topic: archived is terminal, writes are refused")
)

// Rollup compacts the topic: it folds the current baseline plus every operation
// since into a new baseline and publishes it as the atomic replacement of both —
// the server's per-subject rollup purges the predecessors in the same stroke.
//
// Race safety is optimistic: the publish demands that the last message on the
// subject still be the last op this rollup consumed. Any concurrent write rejects
// the attempt wholesale (ErrRollupLost) and the log is untouched — first writer
// wins, nothing to clean up. Rollup is an optimisation: a topic nobody compacts is
// a valid topic.
//
// The new baseline is an ordinary operation — signed when the client holds a key —
// and its payload carries the frontier, so later ops parent across the boundary as
// if the rollup never happened. On success the handle adopts that frontier and the
// new baseline op-id is returned.
func (h *Handle) Rollup(ctx context.Context) (string, error) {
	recs, err := drainOps(ctx, h.client, h.path)
	if err != nil {
		return "", err
	}
	if err := resolveBaseline(ctx, h.client, recs); err != nil {
		return "", err
	}
	mt := apply(h.path, recs)
	if mt.Malformed != "" {
		return "", fmt.Errorf("topic: refusing to compact a malformed topic: %s", mt.Malformed)
	}
	if mt.Lifecycle == Archived {
		return "", fmt.Errorf("topic: %s is archived — %w", h.path, ErrTopicArchived)
	}
	if len(recs) <= 1 {
		return "", ErrNothingToCompact
	}

	payload := BaselinePayload{
		State:    mt.BaselineState,
		Frontier: mt.Frontier,
		Baked: &BakedState{
			Contributions: cleanBakedContributions(mt.Contributions),
			Attachments:   cleanBakedAttachments(mt.Attachments),
			Lifecycle:     mt.Lifecycle,
		},
	}

	lastSeq := recs[len(recs)-1].StreamSeq
	opID, err := publishBaseline(ctx, h, payload, mt.Frontier, lastSeq)
	if err != nil {
		return "", err
	}

	h.frontier = payload.Frontier
	h.lifecycle = mt.Lifecycle
	return opID, nil
}

// publishBaseline publishes a rollup baseline (inline or, when the state document
// exceeds the inline threshold, as a manifest) with the rollup header and the
// expected-last-subject-sequence guard.
func publishBaseline(ctx context.Context, h *Handle, payload BaselinePayload, frontier []string, lastSeq uint64) (string, error) {
	doc, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("topic: encode baseline: %w", err)
	}
	if len(doc) > InlineBaselineThreshold {
		return publishManifestBaseline(ctx, h, payload, frontier, lastSeq)
	}

	opID, err := publishOpWith(ctx, h.client, OpsSubject(h.path), TypeBaseline, payload, frontier,
		map[string]string{jetstream.MsgRollup: jetstream.MsgRollupSubject},
		[]jetstream.PublishOpt{jetstream.WithExpectLastSequencePerSubject(lastSeq)},
	)
	if err != nil {
		if isWrongLastSequence(err) {
			return "", ErrRollupLost
		}
		return "", err
	}
	return opID, nil
}

// publishManifestBaseline handles the over-threshold form. Landed with the manifest
// story; until then oversized states refuse loudly rather than split a baseline
// across messages (the single-message invariant).
func publishManifestBaseline(_ context.Context, h *Handle, _ BaselinePayload, _ []string, _ uint64) (string, error) {
	return "", fmt.Errorf("topic: %s: state document exceeds the %d-byte inline threshold; manifest baselines required", h.path, InlineBaselineThreshold)
}

// resolveBaseline rewrites a manifest baseline's payload into its inline form before
// folding. Landed with the manifest story; inline baselines need no resolution.
func resolveBaseline(_ context.Context, _ *realm.Client, _ []SeqRecord) error {
	return nil
}

// isWrongLastSequence reports whether err is the server rejecting the
// expected-last-subject-sequence guard — the rollup's lost race.
func isWrongLastSequence(err error) bool {
	var apiErr *jetstream.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode == jetstream.JSErrCodeStreamWrongLastSequence
}

// cleanBakedContributions strips the fields baked state never stores: stream
// sequences die with the compacted tail, and sig statuses are recomputed at read
// time (baked elements inherit the baseline's).
func cleanBakedContributions(cs []Contribution) []Contribution {
	out := make([]Contribution, len(cs))
	copy(out, cs)
	for i := range out {
		out[i].StreamSeq = 0
		out[i].Sig = ""
		out[i].Dangling = false // derived at read time
	}
	return out
}

func cleanBakedAttachments(as []Attachment) []Attachment {
	out := make([]Attachment, len(as))
	copy(out, as)
	for i := range out {
		out[i].StreamSeq = 0
		out[i].Sig = ""
		out[i].Dangling = false
	}
	return out
}

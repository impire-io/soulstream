package topic

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/impire/soulstream/record"
)

// Announcement is a topic's info metadata.
//
// The JSON tags on the view structs are wire contract: baked state inside rollup
// baselines serialises these exact shapes, so the keys are pinned here, not left to
// Go's default casing.
type Announcement struct {
	TopicID       string    `json:"topic_id"`
	Name          string    `json:"name"`
	SubjectMatter string    `json:"subject_matter,omitempty"`
	Parent        string    `json:"parent,omitempty"`
	Expected      []string  `json:"expected,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	Sig           SigStatus `json:"sig,omitempty"` // verification status of the announce op
}

// Contribution is a materialised turn or comment.
type Contribution struct {
	OpID      string    `json:"op_id"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"` // TypeTurnPost | TypeCommentAdd
	Body      string    `json:"body"`
	Mentions  []string  `json:"mentions,omitempty"`   // persona names mentioned in the body
	Anchor    string    `json:"anchor,omitempty"`     // comment's anchored op-id ("" for turns)
	Dangling  bool      `json:"dangling,omitempty"`   // comment anchor not present in the topic
	Sig       SigStatus `json:"sig,omitempty"`        // verification status of this op's signature
	StreamSeq uint64    `json:"stream_seq,omitempty"` // 0 for elements baked into a baseline
}

// Attachment is a materialised attachment.add — a reference to a blob in the object store.
type Attachment struct {
	OpID        string    `json:"op_id"`
	Author      string    `json:"author"`
	Timestamp   time.Time `json:"ts"`
	Name        string    `json:"name"`
	Object      string    `json:"object"`
	Digest      string    `json:"digest"`
	Size        uint64    `json:"size"`
	ContentType string    `json:"content_type,omitempty"`
	Anchor      string    `json:"anchor,omitempty"`
	Dangling    bool      `json:"dangling,omitempty"`
	Sig         SigStatus `json:"sig,omitempty"`
	StreamSeq   uint64    `json:"stream_seq,omitempty"`
}

// MaterializedTopic is the pure projection of a topic's op-log.
type MaterializedTopic struct {
	Path          string          `json:"path"`
	Announcement  *Announcement   `json:"announcement,omitempty"`
	BaselineState json.RawMessage `json:"baseline_state,omitempty"`
	// BaselineTs is the baseline op's (author-claimed) timestamp: the topic's birth
	// time, or — after a rollup — the compaction time. Informational, like every
	// timestamp; useful as "when did this topic's current zero-point happen".
	BaselineTs    time.Time      `json:"baseline_ts,omitempty"`
	Lifecycle     Lifecycle      `json:"lifecycle"`
	Contributions []Contribution `json:"contributions,omitempty"`
	Attachments   []Attachment   `json:"attachments,omitempty"`
	Frontier      []string       `json:"frontier"`            // leaf op-ids
	Malformed     string         `json:"malformed,omitempty"` // non-empty reason if the log has no usable baseline
	Warnings      []string       `json:"warnings,omitempty"`  // e.g. ignored unknown op types
}

// SeqRecord pairs a record with its JetStream stream sequence — the ordering key.
type SeqRecord struct {
	Record    record.Record
	StreamSeq uint64
}

// apply folds an ordered sequence of records (already sorted by stream sequence) into a
// materialised view. It is a pure function of the log: the same input always yields the
// same output.
func apply(path string, recs []SeqRecord) *MaterializedTopic {
	mt := &MaterializedTopic{Path: path, Lifecycle: Proposed}

	if len(recs) == 0 {
		mt.Malformed = "topic has no operations (no baseline)"
		return mt
	}
	if recs[0].Record.Type != TypeBaseline {
		mt.Malformed = "first operation is not a baseline (got " + recs[0].Record.Type + ")"
		return mt
	}

	// The baseline: its state, whatever a rollup baked in, and the DAG bookkeeping.
	mt.BaselineTs = recs[0].Record.Timestamp
	var bp BaselinePayload
	if err := json.Unmarshal(recs[0].Record.Payload, &bp); err == nil {
		mt.BaselineState = bp.State
	}
	seen := map[string]bool{recs[0].Record.ID: true}
	referenced := map[string]bool{}
	for _, p := range recs[0].Record.Parents {
		referenced[p] = true
	}

	// Seed from the baked conversation (present after a rollup). Baked op-ids stay
	// anchor-resolvable; the ones not on the frontier are interior — already built
	// upon — so they must never resurface as frontier leaves.
	if bp.Baked != nil {
		mt.Contributions = append(mt.Contributions, bp.Baked.Contributions...)
		mt.Attachments = append(mt.Attachments, bp.Baked.Attachments...)
		if bp.Baked.Lifecycle != "" {
			mt.Lifecycle = bp.Baked.Lifecycle
		}
		for _, c := range bp.Baked.Contributions {
			seen[c.OpID] = true
			referenced[c.OpID] = true
		}
		for _, a := range bp.Baked.Attachments {
			seen[a.OpID] = true
			referenced[a.OpID] = true
		}
	}

	// Frontier continuity: a non-empty payload frontier names the leaves the topic
	// continues from (the baseline op itself becomes a checkpoint, not a leaf). An
	// empty frontier is birth: the baseline op-id is the sole leaf, as always.
	if len(bp.Frontier) > 0 {
		referenced[recs[0].Record.ID] = true
		for _, id := range bp.Frontier {
			seen[id] = true
			delete(referenced, id)
		}
	}

	contentOps := 0
	for _, sr := range recs[1:] {
		r := sr.Record
		seen[r.ID] = true
		for _, p := range r.Parents {
			referenced[p] = true
		}

		switch r.Type {
		case TypeTurnPost:
			contentOps++
			var tp TurnPayload
			_ = json.Unmarshal(r.Payload, &tp)
			mt.Contributions = append(mt.Contributions, Contribution{
				OpID: r.ID, Author: r.Author, Timestamp: r.Timestamp, Type: r.Type,
				Body: tp.Body, Mentions: tp.Mentions, StreamSeq: sr.StreamSeq,
			})
		case TypeCommentAdd:
			contentOps++
			var cp CommentPayload
			_ = json.Unmarshal(r.Payload, &cp)
			mt.Contributions = append(mt.Contributions, Contribution{
				OpID: r.ID, Author: r.Author, Timestamp: r.Timestamp, Type: r.Type,
				Body: cp.Body, Mentions: cp.Mentions, Anchor: cp.Anchor.OpID, StreamSeq: sr.StreamSeq,
			})
		case TypeAttachmentAdd:
			contentOps++
			var ap AttachmentPayload
			_ = json.Unmarshal(r.Payload, &ap)
			mt.Attachments = append(mt.Attachments, Attachment{
				OpID: r.ID, Author: r.Author, Timestamp: r.Timestamp,
				Name: ap.Name, Object: ap.Object, Digest: ap.Digest, Size: ap.Size,
				ContentType: ap.ContentType, Anchor: ap.Anchor, StreamSeq: sr.StreamSeq,
			})
		case TypeLifeTransition:
			if mt.Lifecycle == Archived {
				mt.Warnings = append(mt.Warnings, "ignored transition after archived (archived is terminal)")
				continue
			}
			var lp TransitionPayload
			if err := json.Unmarshal(r.Payload, &lp); err == nil {
				switch lp.To {
				case Closed:
					mt.Lifecycle = Closed
				case Archived:
					mt.Lifecycle = Archived
				}
			}
		case TypeBaseline:
			// A live follower that retained pre-rollup history sees the landed
			// rollup as its next message: a checkpoint whose content this view
			// already holds. Skip its content, but keep the frontier consistent
			// with a cold read: the checkpoint itself is never a leaf, and the
			// leaves it recorded stay the topic's frontier.
			referenced[r.ID] = true
			var cp BaselinePayload
			if json.Unmarshal(r.Payload, &cp) == nil {
				for _, id := range cp.Frontier {
					seen[id] = true
					delete(referenced, id)
				}
			}
			mt.Warnings = append(mt.Warnings, "observed a rollup checkpoint mid-log (view already contains its content)")
		default:
			mt.Warnings = append(mt.Warnings, "ignored unknown op type: "+r.Type)
		}
	}

	// Lifecycle: closed/archived win; otherwise active once there is content.
	if mt.Lifecycle == Proposed && contentOps > 0 {
		mt.Lifecycle = Active
	}

	// Flag comments and attachments whose anchor op-id is not present in the topic.
	for i := range mt.Contributions {
		c := &mt.Contributions[i]
		if c.Type == TypeCommentAdd && c.Anchor != "" && !seen[c.Anchor] {
			c.Dangling = true
		}
	}
	for i := range mt.Attachments {
		a := &mt.Attachments[i]
		if a.Anchor != "" && !seen[a.Anchor] {
			a.Dangling = true
		}
	}

	// Frontier = observed op-ids minus those referenced as some op's parent.
	for id := range seen {
		if !referenced[id] {
			mt.Frontier = append(mt.Frontier, id)
		}
	}
	sort.Strings(mt.Frontier)

	return mt
}

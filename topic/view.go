package topic

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/impire/soulstream/record"
)

// Announcement is a topic's info metadata.
type Announcement struct {
	TopicID       string
	Name          string
	SubjectMatter string
	Parent        string
	Expected      []string
	Tags          []string
}

// Contribution is a materialised turn or comment.
type Contribution struct {
	OpID      string
	Author    string
	Timestamp time.Time
	Type      string // TypeTurnPost | TypeCommentAdd
	Body      string
	Anchor    string // comment's anchored op-id ("" for turns)
	Dangling  bool   // comment anchor not present in the topic
	StreamSeq uint64
}

// MaterializedTopic is the pure projection of a topic's op-log.
type MaterializedTopic struct {
	Path          string
	Announcement  *Announcement
	BaselineState json.RawMessage
	Lifecycle     Lifecycle
	Contributions []Contribution
	Frontier      []string // leaf op-ids
	Malformed     string   // non-empty reason if the first op is not a baseline
	Warnings      []string // e.g. ignored unknown op types
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

	// The baseline: its state, and the start of the DAG bookkeeping.
	var bp BaselinePayload
	if err := json.Unmarshal(recs[0].Record.Payload, &bp); err == nil {
		mt.BaselineState = bp.State
	}
	seen := map[string]bool{recs[0].Record.ID: true}
	referenced := map[string]bool{}
	for _, p := range recs[0].Record.Parents {
		referenced[p] = true
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
				Body: tp.Body, StreamSeq: sr.StreamSeq,
			})
		case TypeCommentAdd:
			contentOps++
			var cp CommentPayload
			_ = json.Unmarshal(r.Payload, &cp)
			mt.Contributions = append(mt.Contributions, Contribution{
				OpID: r.ID, Author: r.Author, Timestamp: r.Timestamp, Type: r.Type,
				Body: cp.Body, Anchor: cp.Anchor.OpID, StreamSeq: sr.StreamSeq,
			})
		case TypeLifeTransition:
			var lp TransitionPayload
			if err := json.Unmarshal(r.Payload, &lp); err == nil && lp.To == Closed {
				mt.Lifecycle = Closed
			}
		default:
			mt.Warnings = append(mt.Warnings, "ignored unknown op type: "+r.Type)
		}
	}

	// Lifecycle: closed wins; otherwise active once there is content.
	if mt.Lifecycle != Closed && contentOps > 0 {
		mt.Lifecycle = Active
	}

	// Flag comments whose anchor op-id is not present in the topic.
	for i := range mt.Contributions {
		c := &mt.Contributions[i]
		if c.Type == TypeCommentAdd && c.Anchor != "" && !seen[c.Anchor] {
			c.Dangling = true
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

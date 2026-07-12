package topic

import "encoding/json"

// Operation types defined this cycle. Types outside this set are ignored with a
// warning during materialisation (additive vocabulary growth).
const (
	TypeAnnounce       = "topic.announce"
	TypeBaseline       = "baseline"
	TypeTurnPost       = "turn.post"
	TypeCommentAdd     = "comment.add"
	TypeLifeTransition = "life.transition"
)

// Lifecycle is a topic's derived state.
type Lifecycle string

// Lifecycle states derivable this cycle (dormant/archived are deferred).
const (
	Proposed Lifecycle = "proposed"
	Active   Lifecycle = "active"
	Closed   Lifecycle = "closed"
)

// AnnouncePayload is the topic.announce payload, carried on the INFO subject.
type AnnouncePayload struct {
	TopicID       string   `json:"topic_id"`
	Name          string   `json:"name"`
	SubjectMatter string   `json:"subject_matter,omitempty"`
	Expected      []string `json:"expected,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Parent        string   `json:"parent,omitempty"`
}

// BaselinePayload is the baseline op payload: the materialised state plus the leaf
// op-ids at baseline time.
type BaselinePayload struct {
	State    json.RawMessage `json:"state"`
	Frontier []string        `json:"frontier"`
}

// TurnPayload is the turn.post payload.
type TurnPayload struct {
	Body string `json:"body"`
}

// Anchor references another operation by its op-id.
type Anchor struct {
	Kind string `json:"kind"` // "op"
	OpID string `json:"op_id"`
}

// CommentPayload is the comment.add payload.
type CommentPayload struct {
	Body   string `json:"body"`
	Anchor Anchor `json:"anchor"`
}

// TransitionPayload is the life.transition payload.
type TransitionPayload struct {
	To   Lifecycle `json:"to"`
	From Lifecycle `json:"from,omitempty"`
}

package topic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/record"
)

// Discovery is the realm's second, live layer of finding topics (the first is the
// durable board): scatter/gather over plain request-reply. A persona shouts a query
// at SOULSTREAM.SVC.DISCOVER with a reply inbox and a deadline; any persona that
// maintains a projection may answer from it; non-answers are silent; the asker
// merges whatever arrived in time. There is no registry to keep consistent and no
// component whose absence breaks discovery — with zero responders, asks resolve
// empty and the board still works.

// ServiceDiscover is the discovery service's name — the canonical binding signed
// over by requests and replies alike (a reply travels on an ephemeral inbox, which
// would be a meaningless thing to sign over).
const ServiceDiscover = "DISCOVER"

// Discovery defaults and caps.
const (
	DefaultDiscoverTimeout = 2 * time.Second
	DefaultDiscoverLimit   = 10
	MaxDiscoverLimit       = 50
)

// DiscoverAnswer credits one answerer for one discovered topic, with the
// verification status of its reply.
type DiscoverAnswer struct {
	Persona string    `json:"persona"`
	Sig     SigStatus `json:"sig,omitempty"`
}

// DiscoverResult is the asker's merged view of one discovered topic: the entry as
// first reported, plus every persona that reported it.
type DiscoverResult struct {
	DiscoverEntry
	Answers []DiscoverAnswer `json:"answers"`
}

// DiscoverInput is an ask: the query, how many matches each answerer should cap at,
// and how long the asker listens.
type DiscoverInput struct {
	Query   string
	Limit   int           // per-answerer cap; 0 means DefaultDiscoverLimit
	Timeout time.Duration // the ask's deadline; 0 means DefaultDiscoverTimeout
}

// Discover publishes a topic.discover request and gathers replies until the
// deadline, merging them into one result per topic path with every answerer
// credited and each answer verified against kr (nil kr: signed answers report
// unknown-key). Zero replies resolve to (nil, nil) — silence is a defined answer,
// and the durable board remains the fallback that always works.
func Discover(ctx context.Context, c *realm.Client, in DiscoverInput, kr *identity.Keyring) ([]DiscoverResult, error) {
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = DefaultDiscoverTimeout
	}
	limit := clampDiscoverLimit(in.Limit)

	nc := c.Conn()
	inbox := nc.NewRespInbox()
	sub, err := nc.SubscribeSync(inbox)
	if err != nil {
		return nil, fmt.Errorf("topic: subscribe reply inbox: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	deadline := time.Now().Add(timeout)
	reqMsg, _, err := buildOpMsg(c, SvcDiscoverSubject, ServiceDiscover, TypeDiscover,
		DiscoverPayload{Query: in.Query, Limit: limit, Deadline: deadline.UTC()}, nil, "")
	if err != nil {
		return nil, err
	}
	reqMsg.Reply = inbox
	if err := nc.PublishMsg(reqMsg); err != nil {
		return nil, fmt.Errorf("topic: publish discover request: %w", err)
	}

	// Gather until the deadline. Every reply is one answerer's testimony; malformed
	// ones are skipped, late ones never arrive (we stop listening).
	var (
		order   []string
		results = map[string]*DiscoverResult{}
	)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		msg, err := sub.NextMsg(remaining)
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				break
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("topic: gather replies: %w", err)
		}
		rec, perr := record.Parse(msg.Header, msg.Data)
		if perr != nil || rec.Type != TypeDiscoverReply {
			continue
		}
		var reply DiscoverReplyPayload
		if json.Unmarshal(rec.Payload, &reply) != nil || len(reply.Matches) == 0 {
			continue
		}
		sig := VerifyRecord(rec, c.Realm(), ServiceDiscover, kr)
		mergeReply(results, &order, rec.Author, sig, reply.Matches)
	}

	if len(order) == 0 {
		return nil, nil
	}
	out := make([]DiscoverResult, 0, len(order))
	for _, path := range order {
		out = append(out, *results[path])
	}
	return out, nil
}

// mergeReply folds one answerer's matches into the merged results: one entry per
// topic path (first-seen fields win — answers are testimony, not reconciled facts),
// one credit per (path, persona) however often it repeats itself.
func mergeReply(results map[string]*DiscoverResult, order *[]string, persona string, sig SigStatus, matches []DiscoverEntry) {
	for _, m := range matches {
		r, ok := results[m.Path]
		if !ok {
			r = &DiscoverResult{DiscoverEntry: m}
			results[m.Path] = r
			*order = append(*order, m.Path)
		}
		credited := false
		for _, a := range r.Answers {
			if a.Persona == persona {
				credited = true
				break
			}
		}
		if !credited {
			r.Answers = append(r.Answers, DiscoverAnswer{Persona: persona, Sig: sig})
		}
	}
}

// RespondDiscovery serves discovery as c's persona until ctx is cancelled: for each
// request it rebuilds the board projection, matches, and replies only when there is
// something to say (signed when the client is keyed). Answering is a habit, not a
// role — any number of responders may run, none coordinates with another, and
// stopping one changes nothing for the rest.
//
// onServed, if non-nil, is called after each request with the query and the number
// of matches sent (0 for a silent no-match, -1 for a skipped malformed or
// reply-less request) — observability for a serving process, nothing more.
func RespondDiscovery(ctx context.Context, c *realm.Client, onServed func(query string, sent int)) error {
	if c.Persona() == "" {
		return fmt.Errorf("topic: responding to discovery requires a persona")
	}
	nc := c.Conn()

	served := func(query string, sent int) {
		if onServed != nil {
			onServed(query, sent)
		}
	}

	// Plain subscribe — deliberately NO queue group: every responder must hear
	// every request; the asker's merge is the only aggregation point.
	sub, err := nc.Subscribe(SvcDiscoverSubject, func(msg *nats.Msg) {
		rec, perr := record.Parse(msg.Header, msg.Data)
		if perr != nil || rec.Type != TypeDiscover || msg.Reply == "" {
			served("", -1)
			return
		}
		var req DiscoverPayload
		if json.Unmarshal(rec.Payload, &req) != nil {
			served("", -1)
			return
		}
		if !req.Deadline.IsZero() && time.Now().After(req.Deadline) {
			served(req.Query, -1) // stale: the reply would be ignored anyway
			return
		}

		entries, berr := Board(ctx, c)
		if berr != nil {
			served(req.Query, -1)
			return
		}
		matches := matchEntries(entries, req.Query, req.Limit)
		if len(matches) == 0 {
			served(req.Query, 0) // silence is cheaper than noise
			return
		}

		reply, _, berr2 := buildOpMsg(c, msg.Reply, ServiceDiscover, TypeDiscoverReply,
			DiscoverReplyPayload{Matches: matches}, nil, "")
		if berr2 != nil {
			served(req.Query, -1)
			return
		}
		if nc.PublishMsg(reply) == nil {
			served(req.Query, len(matches))
		}
	})
	if err != nil {
		return fmt.Errorf("topic: subscribe discovery: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	<-ctx.Done()
	return nil
}

// matchEntries is the answerer's deterministic matcher: case-insensitive substring
// of the query against each topic's name, subject matter, and tags. An empty query
// matches everything. Results keep board order, capped at limit.
func matchEntries(entries []BoardEntry, query string, limit int) []DiscoverEntry {
	limit = clampDiscoverLimit(limit)
	q := strings.ToLower(strings.TrimSpace(query))

	var out []DiscoverEntry
	for _, e := range entries {
		if q != "" && !entryMatches(e, q) {
			continue
		}
		out = append(out, DiscoverEntry{
			Path:          e.Path,
			Name:          e.Announcement.Name,
			SubjectMatter: e.Announcement.SubjectMatter,
			Tags:          e.Announcement.Tags,
			Lifecycle:     e.Lifecycle,
		})
		if len(out) == limit {
			break
		}
	}
	return out
}

func entryMatches(e BoardEntry, q string) bool {
	if strings.Contains(strings.ToLower(e.Announcement.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Announcement.SubjectMatter), q) {
		return true
	}
	for _, tag := range e.Announcement.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

func clampDiscoverLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultDiscoverLimit
	case limit > MaxDiscoverLimit:
		return MaxDiscoverLimit
	default:
		return limit
	}
}

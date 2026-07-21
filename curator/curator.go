package curator

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/topic"
)

// Defaults for a curator's judgment cadence.
const (
	DefaultIdleWindow = 14 * 24 * time.Hour
	DefaultScanEvery  = time.Minute
)

// Options tunes a curator. Zero values take the defaults.
type Options struct {
	// IdleWindow is the quiet period after which a topic earns a dormancy proposal.
	IdleWindow time.Duration
	// ScanEvery is the cadence of the duplicate and dormancy passes (discovery
	// answering is event-driven, per request).
	ScanEvery time.Duration
	// OnEvent, if non-nil, receives one human-readable line per notable act
	// (projection ready, answer served, flag posted, proposal posted).
	OnEvent func(event string)
}

// Run curates as c's persona until ctx is cancelled: it maintains a warm,
// content-aware projection of the realm's topics, answers discovery from it, flags
// likely duplicates, and proposes closure for long-dormant topics. Everything it
// does is an ordinary operation any persona could perform; everything it says is a
// comment anyone may ignore. Stopping is the context cancel — nothing to
// deregister, nothing degrades.
func Run(ctx context.Context, c *realm.Client, opts Options) error {
	if c.Persona() == "" {
		return fmt.Errorf("curator: curating requires a persona")
	}
	if opts.IdleWindow <= 0 {
		opts.IdleWindow = DefaultIdleWindow
	}
	if opts.ScanEvery <= 0 {
		opts.ScanEvery = DefaultScanEvery
	}
	event := func(format string, args ...any) {
		if opts.OnEvent != nil {
			opts.OnEvent(fmt.Sprintf(format, args...))
		}
	}

	p, err := newProjection(ctx, c)
	if err != nil {
		return fmt.Errorf("curator: build projection: %w", err)
	}
	defer p.close()
	event("projection ready: %d topics", len(p.snapshot()))

	// Answer discovery from the projection — same mechanism as any responder, just
	// warmer and deeper. Refresh dirty paths before matching so a live realm is
	// searchable to the second.
	answerCtx, stopAnswering := context.WithCancel(ctx)
	defer stopAnswering()
	answerDone := make(chan struct{})
	go func() {
		defer close(answerDone)
		_ = topic.RespondDiscoveryWith(answerCtx, c, func(query string, limit int) []topic.DiscoverEntry {
			_ = p.refresh(answerCtx)
			return p.search(query, limit)
		}, func(query string, sent int) {
			if sent > 0 {
				event("answered %q: %d matches", query, sent)
			}
		})
	}()

	ticker := time.NewTicker(opts.ScanEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			<-answerDone
			return nil
		case <-ticker.C:
			if err := p.refresh(ctx); err != nil {
				return err
			}
			p.duplicatePass(ctx, event)
			p.dormancyPass(ctx, opts.IdleWindow, event)
		}
	}
}

// duplicatePass flags each unflagged topic that looks like an older one: one
// ordinary comment in the newer topic naming the older path. The log is the
// memory — a duplicate-kind suggestion already present (whoever wrote it) means
// silence.
func (p *projection) duplicatePass(ctx context.Context, event func(string, ...any)) {
	topics := p.snapshot()
	sort.Slice(topics, func(i, j int) bool {
		if !topics[i].birth.Equal(topics[j].birth) {
			return topics[i].birth.Before(topics[j].birth)
		}
		return topics[i].entry.Path < topics[j].entry.Path
	})

	for i, newer := range topics {
		if newer.malformed || newer.entry.Lifecycle == topic.Closed || newer.entry.Lifecycle == topic.Archived {
			continue // resting topics need no redirect; archived refuses writes anyway
		}
		if hasDuplicateFlag(newer.view) {
			continue
		}
		bestScore := 0.0
		bestPath := ""
		for _, older := range topics[:i] {
			if older.malformed {
				continue
			}
			if s := Similarity(newer.entry, older.entry); s > bestScore {
				bestScore, bestPath = s, older.entry.Path
			}
		}
		if bestScore < DuplicateThreshold || bestPath == "" {
			continue
		}
		if p.comment(ctx, newer, DuplicateSuggestion(bestPath)) {
			event("flagged %s as similar to %s (%.0f%%)", newer.entry.Path, bestPath, bestScore*100)
		}
	}
}

// dormancyPass proposes closure for topics quiet past the idle window: one comment
// per quiet spell. A dormancy suggestion newer than the last real activity means
// the nudge was already given; fresh activity re-arms exactly one more.
func (p *projection) dormancyPass(ctx context.Context, window time.Duration, event func(string, ...any)) {
	now := time.Now().UTC()
	for _, ct := range p.snapshot() {
		if ct.malformed || ct.entry.Lifecycle == topic.Closed || ct.entry.Lifecycle == topic.Archived {
			continue
		}
		idle := now.Sub(ct.lastReal)
		if idle <= window {
			continue
		}
		if hasDormantFlagSince(ct.view, ct.lastReal) {
			continue
		}
		if p.comment(ctx, ct, DormantSuggestion(idle)) {
			event("proposed closing %s (idle %s)", ct.entry.Path, humanSpan(idle))
		}
	}
}

// comment posts one suggestion as an ordinary comment anchored to the topic's
// current frontier, and marks the path dirty so the projection sees its own words.
func (p *projection) comment(ctx context.Context, ct *cachedTopic, body string) bool {
	if len(ct.view.Frontier) == 0 {
		return false
	}
	h := topic.Open(p.c, ct.entry.Path)
	if _, err := h.Materialise(ctx); err != nil {
		return false
	}
	if _, err := h.AddComment(ctx, body, ct.view.Frontier[0]); err != nil {
		return false
	}
	p.mu.Lock()
	p.dirty[ct.entry.Path] = true
	p.mu.Unlock()
	return true
}

// hasDuplicateFlag reports whether any duplicate-kind suggestion exists in the
// topic, whoever wrote it.
func hasDuplicateFlag(view *topic.MaterializedTopic) bool {
	for _, c := range view.Contributions {
		if IsDuplicateSuggestion(c.Body) {
			return true
		}
	}
	return false
}

// hasDormantFlagSince reports whether a dormancy-kind suggestion newer than the
// topic's last real activity exists — the "already nudged this quiet spell" test.
func hasDormantFlagSince(view *topic.MaterializedTopic, lastReal time.Time) bool {
	for _, c := range view.Contributions {
		if IsDormantSuggestion(c.Body) && c.Timestamp.After(lastReal) {
			return true
		}
	}
	return false
}

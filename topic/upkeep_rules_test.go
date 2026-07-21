package topic

import (
	"testing"
	"time"
)

var ruleNow = time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)

// TestDormantEligible: the clock is the newest op of ANY kind; only proposed and
// active topics are eligible.
func TestDormantEligible(t *testing.T) {
	window := 14 * 24 * time.Hour
	old := ruleNow.Add(-15 * 24 * time.Hour)
	fresh := ruleNow.Add(-time.Hour)

	base := func(lc Lifecycle) *MaterializedTopic {
		return &MaterializedTopic{Lifecycle: lc, BaselineTs: old}
	}

	if !DormantEligible(base(Active), window, ruleNow) {
		t.Error("idle active topic not eligible")
	}
	if !DormantEligible(base(Proposed), window, ruleNow) {
		t.Error("idle proposed topic not eligible (birth counts)")
	}
	for _, lc := range []Lifecycle{Dormant, Closed, Archived} {
		if DormantEligible(base(lc), window, ruleNow) {
			t.Errorf("%s topic eligible", lc)
		}
	}
	if DormantEligible(&MaterializedTopic{Lifecycle: Active, BaselineTs: old, Malformed: "x"}, window, ruleNow) {
		t.Error("malformed topic eligible")
	}

	// Any newest op defers eligibility: a contribution, an edit stamp, a resolve
	// mark, a removal mark, a work event.
	cases := map[string]*MaterializedTopic{
		"fresh turn": {Lifecycle: Active, BaselineTs: old,
			Contributions: []Contribution{{OpID: "t", Timestamp: fresh}}},
		"fresh edit stamp": {Lifecycle: Active, BaselineTs: old,
			Contributions: []Contribution{{OpID: "t", Timestamp: old, Edits: []EditStamp{{OpID: "e", Ts: fresh}}}}},
		"fresh resolve": {Lifecycle: Active, BaselineTs: old,
			Contributions: []Contribution{{OpID: "t", Timestamp: old, Resolved: true, ResolvedTs: fresh}}},
		"fresh removal": {Lifecycle: Active, BaselineTs: old,
			Attachments: []Attachment{{OpID: "a", Timestamp: old, Removed: true, RemovedTs: fresh}}},
		"fresh work event": {Lifecycle: Active, BaselineTs: old,
			WorkItems: []WorkItem{{ID: "w", Timestamp: old, Timeline: []WorkEvent{{OpID: "e", Timestamp: fresh}}}}},
	}
	for name, mt := range cases {
		if DormantEligible(mt, window, ruleNow) {
			t.Errorf("%s did not defer eligibility", name)
		}
		if got := NewestOpTs(mt); !got.Equal(fresh) {
			t.Errorf("%s: NewestOpTs = %v, want %v", name, got, fresh)
		}
	}
}

// TestStaleClaims: claimed items only; the clock is the newest related activity —
// claim, later timeline events, anchored evidence.
func TestStaleClaims(t *testing.T) {
	window := 7 * 24 * time.Hour
	old := ruleNow.Add(-8 * 24 * time.Hour)
	fresh := ruleNow.Add(-time.Hour)

	claimed := func(ts time.Time) WorkItem {
		return WorkItem{ID: "w1", Status: WorkClaimed, Owner: "scribe", Timestamp: old,
			Timeline: []WorkEvent{{OpID: "clm", Kind: "claim", Author: "scribe", Timestamp: ts}}}
	}

	mt := &MaterializedTopic{WorkItems: []WorkItem{claimed(old)}}
	if got := StaleClaims(mt, window, ruleNow); len(got) != 1 || got[0] != "w1" {
		t.Errorf("stale claim not found: %v", got)
	}
	if got := StaleClaims(&MaterializedTopic{WorkItems: []WorkItem{claimed(fresh)}}, window, ruleNow); len(got) != 0 {
		t.Errorf("fresh claim reported stale: %v", got)
	}

	// Open and done items are never stale.
	for _, st := range []WorkStatus{WorkOpen, WorkDone} {
		mt := &MaterializedTopic{WorkItems: []WorkItem{{ID: "w1", Status: st, Timestamp: old}}}
		if got := StaleClaims(mt, window, ruleNow); len(got) != 0 {
			t.Errorf("%s item reported stale", st)
		}
	}

	// Evidence anchored to the item resets its clock.
	mt = &MaterializedTopic{
		WorkItems:     []WorkItem{claimed(old)},
		Contributions: []Contribution{{OpID: "c", Anchor: "w1", Timestamp: fresh}},
	}
	if got := StaleClaims(mt, window, ruleNow); len(got) != 0 {
		t.Errorf("anchored comment did not reset the clock: %v", got)
	}
	mt = &MaterializedTopic{
		WorkItems:   []WorkItem{claimed(old)},
		Attachments: []Attachment{{OpID: "a", Anchor: "w1", Timestamp: fresh}},
	}
	if got := StaleClaims(mt, window, ruleNow); len(got) != 0 {
		t.Errorf("anchored attachment did not reset the clock: %v", got)
	}

	// A later void claim attempt also counts as activity (interest is visible).
	item := claimed(old)
	item.Timeline = append(item.Timeline, WorkEvent{OpID: "clm2", Kind: "claim", Void: true, Timestamp: fresh})
	if got := StaleClaims(&MaterializedTopic{WorkItems: []WorkItem{item}}, window, ruleNow); len(got) != 0 {
		t.Errorf("void-claim activity ignored: %v", got)
	}
}

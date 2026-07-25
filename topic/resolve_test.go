package topic

import (
	"testing"
	"time"
)

func TestContainsOp(t *testing.T) {
	now := time.Now().UTC()
	mt := &MaterializedTopic{
		Path:         "vat-q2-x7m2",
		BaselineID:   "baseline-id",
		Announcement: &Announcement{OpID: "announce-id", TopicID: "vat-q2-x7m2", Name: "Q2 VAT"},
		Contributions: []Contribution{
			{OpID: "turn-live", Author: "daan", Timestamp: now, Type: TypeTurnPost, Body: "hi", StreamSeq: 3},
			{OpID: "turn-baked", Author: "daan", Timestamp: now, Type: TypeTurnPost, Body: "old",
				Edits: []EditStamp{{OpID: "edit-baked", Author: "daan", Ts: now}}}, // StreamSeq 0: baked
		},
		Attachments: []Attachment{{OpID: "att-1", Author: "daan", Timestamp: now, Name: "n.txt"}},
		WorkItems: []WorkItem{{
			ID: "work-1", Author: "daan", Timestamp: now, Title: "t", Status: WorkClaimed, Owner: "kit",
			Timeline: []WorkEvent{{OpID: "claim-1", Kind: "claim", Author: "kit", Timestamp: now}},
		}},
		Frontier: []string{"leaf-transition"},
	}

	resolves := []string{
		"baseline-id", "announce-id",
		"turn-live", "turn-baked", "edit-baked",
		"att-1", "work-1", "claim-1",
		"leaf-transition",
	}
	for _, id := range resolves {
		if !mt.ContainsOp(id) {
			t.Errorf("ContainsOp(%q) = false, want true", id)
		}
	}

	doesNot := []string{
		"", "never-existed",
		"compacted-mark", // a resolve/remove mark compaction consumed without baking an id
	}
	for _, id := range doesNot {
		if mt.ContainsOp(id) {
			t.Errorf("ContainsOp(%q) = true, want false", id)
		}
	}

	// A nil view (unreadable topic) resolves nothing and never panics.
	var nilView *MaterializedTopic
	if nilView.ContainsOp("anything") {
		t.Error("nil view must resolve nothing")
	}
}

func TestGradeForVerdict(t *testing.T) {
	cases := map[SigStatus]MemoryGrade{
		SigVerified:   GradeProvenance,
		SigUnsigned:   GradeTestimony,
		SigFailed:     GradeUnverifiable,
		SigUnknownKey: GradeUnverifiable,
	}
	for verdict, want := range cases {
		if got := GradeForVerdict(verdict); got != want {
			t.Errorf("GradeForVerdict(%s) = %s, want %s", verdict, got, want)
		}
	}
}

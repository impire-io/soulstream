package topic

import (
	"encoding/json"
	"strings"
	"testing"
)

// convLog builds: baseline, a turn by daan, a comment by scribe anchored to it.
func convLog(tail ...SeqRecord) []SeqRecord {
	recs := []SeqRecord{
		foldRec(1, "base-0", "daan", TypeBaseline, BaselinePayload{Frontier: []string{}}),
		foldRec(2, "turn-1", "daan", TypeTurnPost, TurnPayload{Body: "lets ship thursdy"}, "base-0"),
		foldRec(3, "cmnt-1", "scribe", TypeCommentAdd, CommentPayload{Body: "which thursday?", Anchor: Anchor{Kind: "op", OpID: "turn-1"}}, "turn-1"),
	}
	return append(recs, tail...)
}

func editRec(seq uint64, id, author, target, body string, parents ...string) SeqRecord {
	return foldRec(seq, id, author, TypeEdit, CommentPayload{Body: body, Anchor: Anchor{Kind: "op", OpID: target}}, parents...)
}

func refRec(seq uint64, id, author, opType, target string, parents ...string) SeqRecord {
	return foldRec(seq, id, author, opType, RefPayload{Anchor: &Anchor{Kind: "op", OpID: target}}, parents...)
}

func contribution(t *testing.T, mt *MaterializedTopic, opID string) Contribution {
	t.Helper()
	for _, c := range mt.Contributions {
		if c.OpID == opID {
			return c
		}
	}
	t.Fatalf("contribution %s not in view", opID)
	return Contribution{}
}

// TestUpkeepFoldEdit: the author's edit rewrites the rendered body in place,
// stamps the trail, and chains — via the original or any prior edit; concurrent
// edits converge on the newest in stream order.
func TestUpkeepFoldEdit(t *testing.T) {
	mt := apply("t", convLog(
		editRec(4, "edit-1", "daan", "turn-1", "let's ship Thursday", "cmnt-1"),
		editRec(5, "edit-2", "daan", "edit-1", "let's ship Thursday the 30th", "edit-1"),
	))
	c := contribution(t, mt, "turn-1")
	if c.Body != "let's ship Thursday the 30th" {
		t.Errorf("rendered body = %q, want the newest edit's", c.Body)
	}
	if c.Author != "daan" || len(c.Edits) != 2 || c.Edits[1].OpID != "edit-2" {
		t.Errorf("edit trail wrong: author=%s edits=%+v", c.Author, c.Edits)
	}
	if len(mt.Warnings) != 0 {
		t.Errorf("clean edits warned: %v", mt.Warnings)
	}

	// Concurrent edits of the same target: both anchored to turn-1, later wins.
	mt = apply("t", convLog(
		editRec(4, "edit-a", "daan", "turn-1", "version A", "cmnt-1"),
		editRec(5, "edit-b", "daan", "turn-1", "version B", "cmnt-1"),
	))
	if got := contribution(t, mt, "turn-1").Body; got != "version B" {
		t.Errorf("concurrent edits rendered %q, want stream-order winner", got)
	}
}

// TestUpkeepFoldEditRejections: foreign, unknown-target, empty, unreadable, and
// non-contribution edits warn and change nothing.
func TestUpkeepFoldEditRejections(t *testing.T) {
	t.Run("foreign edit", func(t *testing.T) {
		mt := apply("t", convLog(editRec(4, "edit-x", "scribe", "turn-1", "let's ship Friday", "cmnt-1")))
		if got := contribution(t, mt, "turn-1"); got.Body != "lets ship thursdy" || len(got.Edits) != 0 {
			t.Errorf("foreign edit took effect: %+v", got)
		}
		if !warningsContain(mt, "only the author may edit") {
			t.Errorf("warnings = %v", mt.Warnings)
		}
	})
	t.Run("unknown target", func(t *testing.T) {
		mt := apply("t", convLog(editRec(4, "edit-x", "daan", "ghost", "text", "cmnt-1")))
		if !warningsContain(mt, "unknown or non-editable") {
			t.Errorf("warnings = %v", mt.Warnings)
		}
	})
	t.Run("empty body is malformed", func(t *testing.T) {
		mt := apply("t", convLog(editRec(4, "edit-x", "daan", "turn-1", "  ", "cmnt-1")))
		if !warningsContain(mt, "malformed edit") {
			t.Errorf("warnings = %v", mt.Warnings)
		}
		if got := contribution(t, mt, "turn-1").Body; got != "lets ship thursdy" {
			t.Errorf("empty edit blanked the body: %q", got)
		}
	})
	t.Run("attachment is not editable prose", func(t *testing.T) {
		mt := apply("t", convLog(
			foldRec(4, "attn-1", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "n", Object: "o", Digest: "d", Size: 1}, "cmnt-1"),
			editRec(5, "edit-x", "daan", "attn-1", "text", "attn-1"),
		))
		if !warningsContain(mt, "unknown or non-editable") {
			t.Errorf("warnings = %v", mt.Warnings)
		}
	})
}

// TestUpkeepFoldReplyAndResolve: replies thread like comments; resolve marks the
// comment (first resolver, idempotent duplicates, turns refuse).
func TestUpkeepFoldReplyAndResolve(t *testing.T) {
	mt := apply("t", convLog(
		foldRec(4, "rply-1", "daan", TypeCommentReply, CommentPayload{Body: "the 30th", Anchor: Anchor{Kind: "op", OpID: "cmnt-1"}}, "cmnt-1"),
		refRec(5, "rslv-1", "daan", TypeCommentResolve, "cmnt-1", "rply-1"),
		refRec(6, "rslv-2", "scribe", TypeCommentResolve, "cmnt-1", "rslv-1"),
	))
	r := contribution(t, mt, "rply-1")
	if r.Type != TypeCommentReply || r.Anchor != "cmnt-1" || r.Dangling {
		t.Errorf("reply wrong: %+v", r)
	}
	c := contribution(t, mt, "cmnt-1")
	if !c.Resolved || c.ResolvedBy != "daan" {
		t.Errorf("resolve mark wrong: %+v", c)
	}
	if len(mt.Warnings) != 0 {
		t.Errorf("duplicate resolve warned: %v", mt.Warnings)
	}

	// Resolving a turn or a ghost warns.
	mt = apply("t", convLog(
		refRec(4, "rslv-x", "daan", TypeCommentResolve, "turn-1", "cmnt-1"),
		refRec(5, "rslv-y", "daan", TypeCommentResolve, "ghost", "rslv-x"),
	))
	if !warningsContain(mt, "not a comment") {
		t.Errorf("warnings = %v", mt.Warnings)
	}
	if contribution(t, mt, "turn-1").Resolved {
		t.Error("a turn got resolved")
	}
	// A reply can itself be resolved.
	mt = apply("t", convLog(
		foldRec(4, "rply-1", "daan", TypeCommentReply, CommentPayload{Body: "sub-q", Anchor: Anchor{Kind: "op", OpID: "cmnt-1"}}, "cmnt-1"),
		refRec(5, "rslv-1", "scribe", TypeCommentResolve, "rply-1", "rply-1"),
	))
	if got := contribution(t, mt, "rply-1"); !got.Resolved || got.ResolvedBy != "scribe" {
		t.Errorf("reply resolve wrong: %+v", got)
	}
}

// TestUpkeepFoldRemove: removal marks the attachment (attributed, idempotent),
// warns on non-attachments, and artefact tips fall back to standing revisions.
func TestUpkeepFoldRemove(t *testing.T) {
	mt := apply("t", convLog(
		foldRec(4, "attn-1", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "specs.pdf", Object: "o1", Digest: "d1", Size: 1}, "cmnt-1"),
		foldRec(5, "attn-2", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "specs.pdf", Object: "o2", Digest: "d2", Size: 2, Anchor: "attn-1"}, "attn-1"),
		refRec(6, "rmov-1", "scribe", TypeAttachmentRemove, "attn-2", "attn-2"),
		refRec(7, "rmov-2", "daan", TypeAttachmentRemove, "attn-2", "rmov-1"),
	))
	var removed Attachment
	for _, a := range mt.Attachments {
		if a.OpID == "attn-2" {
			removed = a
		}
	}
	if !removed.Removed || removed.RemovedBy != "scribe" {
		t.Errorf("removed mark wrong (first remover attributes): %+v", removed)
	}
	if len(mt.Warnings) != 0 {
		t.Errorf("duplicate remove warned: %v", mt.Warnings)
	}
	arts := mt.Artefacts()
	if len(arts) != 1 || arts[0].Tip.OpID != "attn-1" {
		t.Fatalf("tip did not fall back: %+v", arts)
	}
	if len(arts[0].Revisions) != 2 {
		t.Errorf("removed revision left the history")
	}

	// Removing the whole lineage withdraws the artefact from the list.
	mt = apply("t", convLog(
		foldRec(4, "attn-1", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "x", Object: "o", Digest: "d", Size: 1}, "cmnt-1"),
		refRec(5, "rmov-1", "daan", TypeAttachmentRemove, "attn-1", "attn-1"),
	))
	if arts := mt.Artefacts(); len(arts) != 0 {
		t.Errorf("fully-removed lineage still listed: %+v", arts)
	}
	if len(mt.Attachments) != 1 || !mt.Attachments[0].Removed {
		t.Errorf("removed entry vanished from the view: %+v", mt.Attachments)
	}

	// Removing a turn warns.
	mt = apply("t", convLog(refRec(4, "rmov-x", "daan", TypeAttachmentRemove, "turn-1", "cmnt-1")))
	if !warningsContain(mt, "not an attachment") {
		t.Errorf("warnings = %v", mt.Warnings)
	}
}

// TestUpkeepFoldDormant: proposed/active → dormant; closed/archived ignore it;
// any content op wakes the topic — order-sensitively.
func TestUpkeepFoldDormant(t *testing.T) {
	dormantAt := func(seq uint64, id string, parents ...string) SeqRecord {
		return foldRec(seq, id, "daan", TypeLifeTransition, TransitionPayload{To: Dormant}, parents...)
	}

	mt := apply("t", convLog(dormantAt(4, "life-1", "cmnt-1")))
	if mt.Lifecycle != Dormant {
		t.Fatalf("lifecycle = %s, want dormant", mt.Lifecycle)
	}

	// Each content-op kind wakes it.
	wakers := []SeqRecord{
		foldRec(5, "wake-t", "x", TypeTurnPost, TurnPayload{Body: "back"}, "life-1"),
		foldRec(5, "wake-r", "x", TypeCommentReply, CommentPayload{Body: "back", Anchor: Anchor{Kind: "op", OpID: "cmnt-1"}}, "life-1"),
		editRec(5, "wake-e", "daan", "turn-1", "edited awake", "life-1"),
		refRec(5, "wake-v", "x", TypeCommentResolve, "cmnt-1", "life-1"),
		foldRec(5, "wake-a", "x", TypeAttachmentAdd, AttachmentPayload{Name: "n", Object: "o", Digest: "d", Size: 1}, "life-1"),
		foldRec(5, "wake-w", "x", TypeWorkOpen, WorkOpenPayload{Title: "chore"}, "life-1"),
	}
	for _, wake := range wakers {
		if got := apply("t", convLog(dormantAt(4, "life-1", "cmnt-1"), wake)).Lifecycle; got != Active {
			t.Errorf("%s did not wake the topic (lifecycle %s)", wake.Record.Type, got)
		}
	}

	// A void/no-effect op does not wake: a foreign edit lands, changes nothing.
	mt = apply("t", convLog(dormantAt(4, "life-1", "cmnt-1"),
		editRec(5, "edit-x", "scribe", "turn-1", "rewrite", "life-1")))
	if mt.Lifecycle != Dormant {
		t.Errorf("a rejected op woke the topic")
	}

	// Content BEFORE the mark does not wake it (order matters).
	if got := apply("t", convLog(dormantAt(4, "life-1", "cmnt-1"))).Lifecycle; got != Dormant {
		t.Errorf("pre-mark content woke the topic: %s", got)
	}

	// Closed ignores dormant (with a warning); archived stays terminal.
	mt = apply("t", convLog(
		foldRec(4, "life-c", "daan", TypeLifeTransition, TransitionPayload{To: Closed}, "cmnt-1"),
		dormantAt(5, "life-d", "life-c"),
	))
	if mt.Lifecycle != Closed || !warningsContain(mt, "closed topic") {
		t.Errorf("closed → dormant: lifecycle %s, warnings %v", mt.Lifecycle, mt.Warnings)
	}
	// Dormant never blocks closing.
	mt = apply("t", convLog(dormantAt(4, "life-1", "cmnt-1"),
		foldRec(5, "life-c", "daan", TypeLifeTransition, TransitionPayload{To: Closed}, "life-1")))
	if mt.Lifecycle != Closed {
		t.Errorf("dormant blocked close: %s", mt.Lifecycle)
	}
}

// TestUpkeepFoldRoundTrip: edits (with a post-rollup chain continuation),
// resolves, removes, and dormant all survive compaction.
func TestUpkeepFoldRoundTrip(t *testing.T) {
	logRecs := convLog(
		editRec(4, "edit-1", "daan", "turn-1", "let's ship Thursday", "cmnt-1"),
		foldRec(5, "rply-1", "daan", TypeCommentReply, CommentPayload{Body: "the 30th", Anchor: Anchor{Kind: "op", OpID: "cmnt-1"}}, "edit-1"),
		refRec(6, "rslv-1", "daan", TypeCommentResolve, "cmnt-1", "rply-1"),
		foldRec(7, "attn-1", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "n", Object: "o", Digest: "d", Size: 1}, "rslv-1"),
		refRec(8, "rmov-1", "scribe", TypeAttachmentRemove, "attn-1", "attn-1"),
	)
	before := apply("t", logRecs)
	viewsEqual(t, apply("t", logRecs), apply("t", []SeqRecord{rollupOf(logRecs, "base-1")}))

	// The chain survives: an edit anchoring the COMPACTED edit-1 still applies.
	tail := editRec(101, "edit-2", "daan", "edit-1", "let's ship Thursday the 30th", before.Frontier...)
	full := apply("t", append(convLog(
		editRec(4, "edit-1", "daan", "turn-1", "let's ship Thursday", "cmnt-1"),
		foldRec(5, "rply-1", "daan", TypeCommentReply, CommentPayload{Body: "the 30th", Anchor: Anchor{Kind: "op", OpID: "cmnt-1"}}, "edit-1"),
		refRec(6, "rslv-1", "daan", TypeCommentResolve, "cmnt-1", "rply-1"),
		foldRec(7, "attn-1", "daan", TypeAttachmentAdd, AttachmentPayload{Name: "n", Object: "o", Digest: "d", Size: 1}, "rslv-1"),
		refRec(8, "rmov-1", "scribe", TypeAttachmentRemove, "attn-1", "attn-1"),
	), tail))
	baked := apply("t", []SeqRecord{rollupOf(logRecs, "base-1"), tail})
	viewsEqual(t, full, baked)
	if got := contribution(t, baked, "turn-1"); got.Body != "let's ship Thursday the 30th" || len(got.Edits) != 2 {
		t.Errorf("post-rollup edit of a compacted chain member failed: %+v", got)
	}

	// Dormant bakes; a tail content op wakes the baked state.
	dLog := convLog(foldRec(4, "life-1", "daan", TypeLifeTransition, TransitionPayload{To: Dormant}, "cmnt-1"))
	dBefore := apply("t", dLog)
	if got := apply("t", []SeqRecord{rollupOf(dLog, "base-1")}).Lifecycle; got != Dormant {
		t.Fatalf("baked lifecycle = %s, want dormant", got)
	}
	wake := foldRec(101, "wake-1", "x", TypeTurnPost, TurnPayload{Body: "hi"}, dBefore.Frontier...)
	if got := apply("t", []SeqRecord{rollupOf(dLog, "base-1"), wake}).Lifecycle; got != Active {
		t.Errorf("baked dormant not woken by tail content: %s", got)
	}
}

// TestUpkeepFoldNoNewFieldsWhenUnused (FR-016): a log without 011 vocabulary
// serialises with none of the new keys.
func TestUpkeepFoldNoNewFieldsWhenUnused(t *testing.T) {
	out, err := json.Marshal(apply("t", convLog()))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"edits", "resolved", "resolved_by", "resolved_ts", "removed", "removed_by", "removed_ts"} {
		if strings.Contains(string(out), `"`+key+`"`) {
			t.Errorf("unused field %q serialised: %s", key, out)
		}
	}
}

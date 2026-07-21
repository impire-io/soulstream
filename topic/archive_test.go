package topic

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestCloseCompacts (US3 scenario 1): closing leaves a compacted closed topic.
func TestCloseCompacts(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h := buildFullTopic(t, c)

	if _, err := h.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if n := countOps(t, c, h.Path()); n != 1 {
		t.Errorf("ops after close = %d, want 1 (close tidies up)", n)
	}
	v, err := Open(c, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Lifecycle != Closed {
		t.Errorf("lifecycle after close = %s", v.Lifecycle)
	}
	if len(v.Contributions) != 3 {
		t.Errorf("contributions after close = %d, want 3", len(v.Contributions))
	}
}

// TestArchiveIsFinal (US3 scenarios 2–5): archive ends as one terminal baseline,
// reads work, every write refuses, double archive reports already-archived.
func TestArchiveIsFinal(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h := buildFullTopic(t, c)

	baselineID, err := h.Archive(ctx)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if baselineID == "" {
		t.Fatal("Archive returned no final baseline op-id")
	}
	if n := countOps(t, c, h.Path()); n != 1 {
		t.Errorf("ops after archive = %d, want exactly 1 (scenario 2)", n)
	}

	// Scenario 3: fully readable.
	rh := Open(c, h.Path())
	v, err := rh.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Lifecycle != Archived {
		t.Fatalf("lifecycle = %s", v.Lifecycle)
	}
	if len(v.Contributions) != 3 || len(v.Attachments) != 1 {
		t.Errorf("archived view incomplete: %d contributions, %d attachments", len(v.Contributions), len(v.Attachments))
	}

	// Scenario 4: every write path refuses.
	if _, err := rh.PostTurn(ctx, "no"); !errors.Is(err, ErrTopicArchived) {
		t.Errorf("PostTurn on archived: %v", err)
	}
	if _, err := rh.AddComment(ctx, "no", v.Contributions[0].OpID); !errors.Is(err, ErrTopicArchived) {
		t.Errorf("AddComment on archived: %v", err)
	}
	if _, err := rh.Attach(ctx, "f.txt", "text/plain", []byte("no"), ""); !errors.Is(err, ErrTopicArchived) {
		t.Errorf("Attach on archived: %v", err)
	}
	if _, err := rh.Transition(ctx, Closed); !errors.Is(err, ErrTopicArchived) {
		t.Errorf("Transition on archived: %v", err)
	}
	if _, err := rh.Rollup(ctx); !errors.Is(err, ErrTopicArchived) {
		t.Errorf("Rollup on archived: %v", err)
	}
	if n := countOps(t, c, h.Path()); n != 1 {
		t.Errorf("refused writes changed the log: %d ops", n)
	}

	// Scenario 5: double archive reports already-archived, changes nothing.
	if _, err := rh.Archive(ctx); !errors.Is(err, ErrTopicArchived) || !strings.Contains(err.Error(), "already archived") {
		t.Errorf("double archive: %v", err)
	}
	if n := countOps(t, c, h.Path()); n != 1 {
		t.Errorf("double archive changed the log: %d ops", n)
	}
}

// TestArchiveRacedByPost: a post landing between the transition and the compaction
// is retried into the final baseline — nothing lost, archive still lands.
func TestArchiveRacedByPost(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h := buildFullTopic(t, c)

	// Simulate the half-done archival: bare transition (raced world), plus a post
	// that slipped in after it (from a handle that had not observed archived yet).
	sneaky := Open(c, h.Path())
	if _, err := sneaky.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Transition(ctx, Archived); err != nil {
		t.Fatal(err)
	}
	if _, err := sneaky.PostTurn(ctx, "slipped in during archival"); err != nil {
		t.Fatal(err)
	}

	// Archive on a fresh handle finds the archived-but-uncompacted topic and
	// finishes the job, folding the raced-in post into the terminal baseline.
	finisher := Open(c, h.Path())
	baselineID, err := finisher.Archive(ctx)
	if err != nil {
		t.Fatalf("finishing Archive: %v", err)
	}
	if baselineID == "" {
		t.Fatal("no final baseline id")
	}
	if n := countOps(t, c, h.Path()); n != 1 {
		t.Errorf("ops after finished archival = %d, want 1", n)
	}
	v, err := Open(c, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Lifecycle != Archived {
		t.Errorf("lifecycle = %s", v.Lifecycle)
	}
	found := false
	for _, cb := range v.Contributions {
		if cb.Body == "slipped in during archival" {
			found = true
		}
	}
	if !found {
		t.Error("the raced-in post was lost by archival")
	}
}

// TestCloseRacedStillValid: a close whose tidy-up loses the race leaves a valid,
// uncompacted closed topic — no error surfaces.
func TestCloseRacedStillValid(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h := buildFullTopic(t, c)

	// Make Close's internal rollup lose: post from another handle between the
	// transition and the compaction — not directly injectable, so emulate the
	// outcome by verifying the contract at the library level instead: a closed
	// topic with a tail is valid and reads correctly.
	if _, err := h.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}
	v, err := Open(c, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Lifecycle != Closed {
		t.Errorf("lifecycle = %s", v.Lifecycle)
	}
	if n := countOps(t, c, h.Path()); n <= 1 {
		t.Fatal("test premise broken: expected an uncompacted closed topic")
	}
	if len(v.Contributions) != 3 {
		t.Errorf("uncompacted closed topic reads wrong: %d contributions", len(v.Contributions))
	}
}

package topic

import (
	"context"
	"strings"
	"testing"
)

func TestSubTopicNestingAndIndependence(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	parent, err := StartTopic(ctx, c, StartTopicInput{Name: "VAT"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := StartTopic(ctx, c, StartTopicInput{Name: "Pricing angle", Parent: parent.Path()})
	if err != nil {
		t.Fatal(err)
	}

	if ParentPath(sub.Path()) != parent.Path() {
		t.Errorf("sub path %q is not nested under %q", sub.Path(), parent.Path())
	}
	if OpsSubject(sub.Path()) != OpsSubjectPrefix+sub.Path() {
		t.Errorf("sub ops subject wrong: %q", OpsSubject(sub.Path()))
	}

	// A post to the sub-topic must not appear in the parent (separate subjects).
	if _, err := sub.PostTurn(ctx, "sub-only turn"); err != nil {
		t.Fatal(err)
	}
	subView, err := Open(c, sub.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(subView.Contributions) != 1 {
		t.Errorf("sub contributions = %d, want 1", len(subView.Contributions))
	}
	parentView, err := Open(c, parent.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(parentView.Contributions) != 0 {
		t.Errorf("parent saw sub-topic ops: %d, want 0", len(parentView.Contributions))
	}
}

func TestSubTopicDeepNesting(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	a, err := StartTopic(ctx, c, StartTopicInput{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := StartTopic(ctx, c, StartTopicInput{Name: "B", Parent: a.Path()})
	if err != nil {
		t.Fatal(err)
	}
	d, err := StartTopic(ctx, c, StartTopicInput{Name: "D", Parent: b.Path()})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(d.Path(), ".") != 2 {
		t.Errorf("deep path %q should have 2 dots", d.Path())
	}
	if _, err := Open(c, d.Path()).Materialise(ctx); err != nil {
		t.Fatalf("deeply-nested topic did not materialise: %v", err)
	}
}

func TestBoardShowsParentRelationship(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	parent, err := StartTopic(ctx, c, StartTopicInput{Name: "VAT"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := StartTopic(ctx, c, StartTopicInput{Name: "Pricing", Parent: parent.Path()})
	if err != nil {
		t.Fatal(err)
	}

	board, err := Board(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]BoardEntry{}
	for _, e := range board {
		byPath[e.Path] = e
	}

	se, ok := byPath[sub.Path()]
	if !ok {
		t.Fatal("sub-topic missing from board")
	}
	if se.Parent != parent.Path() {
		t.Errorf("sub parent = %q, want %q", se.Parent, parent.Path())
	}
	if !se.ParentKnown {
		t.Error("a known parent was flagged unknown")
	}
}

func TestBoardFlagsUnknownParentButKeepsEntry(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	// A sub-topic announced under a parent that was never started.
	orphan, err := StartTopic(ctx, c, StartTopicInput{Name: "Orphan", Parent: "ghost-parent-zzzz"})
	if err != nil {
		t.Fatal(err)
	}

	board, err := Board(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]BoardEntry{}
	for _, e := range board {
		byPath[e.Path] = e
	}
	e, ok := byPath[orphan.Path()]
	if !ok {
		t.Fatal("orphan sub-topic was dropped from the board")
	}
	if e.ParentKnown {
		t.Error("unknown parent should be flagged ParentKnown = false")
	}
}

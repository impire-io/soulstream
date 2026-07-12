package topic

import (
	"context"
	"testing"
)

// TestQuickstartWalkthrough mirrors specs/002-topics/quickstart.md end to end (using a
// direct connection in place of a named context): start a topic, converse, close, add a
// sub-topic, and list the board. If a public signature drifts from the quickstart, this
// test breaks.
func TestQuickstartWalkthrough(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	// 1. Start a topic.
	h, err := StartTopic(ctx, c, StartTopicInput{
		Name:          "Q2 VAT filing",
		SubjectMatter: "Preparing and checking the Q2 2026 VAT return.",
		Tags:          []string{"finance", "recurring"},
		Expected:      []string{"daan", "bookkeeper-agent"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Talk, comment, close.
	turnID, err := h.PostTurn(ctx, "Numbers are in — can you sanity-check box 5?")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddComment(ctx, "Box 5 looks off by the reverse-charge total.", turnID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}

	// 3. Materialise.
	view, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if view.Lifecycle != Closed {
		t.Errorf("lifecycle = %q, want closed", view.Lifecycle)
	}
	if len(view.Contributions) != 2 {
		t.Errorf("contributions = %d, want 2", len(view.Contributions))
	}

	// 5. Sub-topic.
	sub, err := StartTopic(ctx, c, StartTopicInput{Name: "Pricing angle", Parent: h.Path()})
	if err != nil {
		t.Fatal(err)
	}
	if ParentPath(sub.Path()) != h.Path() {
		t.Errorf("sub-topic %q not nested under %q", sub.Path(), h.Path())
	}

	// 6. Board.
	board, err := Board(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(board) != 2 {
		t.Errorf("board = %d entries, want 2 (topic + sub-topic)", len(board))
	}
}

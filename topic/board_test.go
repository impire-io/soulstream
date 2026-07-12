package topic

import (
	"context"
	"testing"
)

func TestBoardEmptyRealm(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	board, err := Board(ctx, c)
	if err != nil {
		t.Fatalf("Board(empty): %v", err)
	}
	if len(board) != 0 {
		t.Errorf("empty realm board = %d entries, want 0", len(board))
	}
}

func TestBoardListsStartedTopics(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	h1, err := StartTopic(ctx, c, StartTopicInput{Name: "Alpha", Tags: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StartTopic(ctx, c, StartTopicInput{Name: "Beta"}); err != nil {
		t.Fatal(err)
	}

	board, err := Board(ctx, c)
	if err != nil {
		t.Fatalf("Board: %v", err)
	}
	if len(board) != 2 {
		t.Fatalf("board = %d entries, want 2", len(board))
	}

	byPath := map[string]BoardEntry{}
	for _, e := range board {
		byPath[e.Path] = e
	}
	e1, ok := byPath[h1.Path()]
	if !ok {
		t.Fatalf("topic %s missing from board", h1.Path())
	}
	if e1.Announcement.Name != "Alpha" {
		t.Errorf("name = %q, want Alpha", e1.Announcement.Name)
	}
	if len(e1.Announcement.Tags) != 1 || e1.Announcement.Tags[0] != "x" {
		t.Errorf("tags = %v, want [x]", e1.Announcement.Tags)
	}
	if e1.Lifecycle != Proposed {
		t.Errorf("lifecycle = %q, want proposed", e1.Lifecycle)
	}
	if !e1.ParentKnown {
		t.Error("top-level topic should have ParentKnown = true")
	}
}

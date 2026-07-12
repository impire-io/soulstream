package topic

import (
	"context"
	"testing"
)

func TestMaterialiseConversation(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "chat"})
	if err != nil {
		t.Fatal(err)
	}

	turnID, err := h.PostTurn(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.PostTurn(ctx, "world"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddComment(ctx, "re hello", turnID); err != nil {
		t.Fatal(err)
	}

	mt, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mt.Lifecycle != Active {
		t.Errorf("lifecycle = %q, want active", mt.Lifecycle)
	}
	if len(mt.Contributions) != 3 {
		t.Fatalf("contributions = %d, want 3", len(mt.Contributions))
	}
	if mt.Contributions[0].Body != "hello" || mt.Contributions[1].Body != "world" {
		t.Errorf("contributions out of stream order: %+v", mt.Contributions)
	}

	var comment *Contribution
	for i := range mt.Contributions {
		if mt.Contributions[i].Type == TypeCommentAdd {
			comment = &mt.Contributions[i]
		}
	}
	if comment == nil {
		t.Fatal("comment not materialised")
	}
	if comment.Anchor != turnID {
		t.Errorf("comment anchor = %q, want %q", comment.Anchor, turnID)
	}
	if comment.Dangling {
		t.Error("comment to a present op flagged dangling")
	}
}

func TestMaterialiseDeterministicAcrossHandles(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, _ := StartTopic(ctx, c, StartTopicInput{Name: "chat"})
	_, _ = h.PostTurn(ctx, "one")
	_, _ = h.PostTurn(ctx, "two")

	a, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// A fresh, cold handle must produce the identical view.
	b, err := Open(c, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Contributions) != len(b.Contributions) {
		t.Fatalf("contribution counts differ: %d vs %d", len(a.Contributions), len(b.Contributions))
	}
	for i := range a.Contributions {
		if a.Contributions[i] != b.Contributions[i] {
			t.Errorf("contribution %d differs between materialisations", i)
		}
	}
}

func TestMaterialiseDanglingComment(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, _ := StartTopic(ctx, c, StartTopicInput{Name: "chat"})

	if _, err := h.AddComment(ctx, "orphan", "no-such-op"); err != nil {
		t.Fatal(err)
	}
	mt, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(mt.Contributions) != 1 || !mt.Contributions[0].Dangling {
		t.Errorf("comment with missing anchor not flagged dangling: %+v", mt.Contributions)
	}
}

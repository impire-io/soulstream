package topic

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestFetchInbox(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")

	// Empty inbox.
	got, err := FetchInbox(ctx, c, "bookkeeper-agent", 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("empty inbox returned %d, want 0", len(got))
	}

	// Post three notifications.
	for i := range 3 {
		if err := publishNotify(ctx, c, "bookkeeper-agent", NotifyPayload{
			Topic: "vat", OpID: fmt.Sprintf("op-%d", i), Author: "daan",
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err = FetchInbox(ctx, c, "bookkeeper-agent", 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("inbox returned %d, want 3", len(got))
	}
	if got[0].OpID != "op-2" {
		t.Errorf("inbox not newest-first: %+v", got)
	}

	// Honour the limit.
	got, err = FetchInbox(ctx, c, "bookkeeper-agent", 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].OpID != "op-2" {
		t.Errorf("limited inbox = %+v, want 2 newest-first", got)
	}
}

func TestMentionNotifiesInbox(t *testing.T) {
	ctx := context.Background()
	url := testServer(t)
	c1 := connectClient(t, url, "daan")
	if _, err := c1.Provision(ctx); err != nil {
		t.Fatal(err)
	}
	c2 := connectClient(t, url, "bookkeeper-agent")

	h, err := StartTopic(ctx, c1, StartTopicInput{Name: "VAT"})
	if err != nil {
		t.Fatal(err)
	}

	fctx, cancel := context.WithCancel(ctx)
	defer cancel()
	notes := make(chan Notification, 8)
	go func() { _ = FollowInbox(fctx, c2, "bookkeeper-agent", nil, func(n Notification) { notes <- n }) }()

	opID, err := h.PostTurn(ctx, "please check box 5 @bookkeeper-agent")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case n := <-notes:
		if n.Topic != h.Path() {
			t.Errorf("notification topic = %q, want %q", n.Topic, h.Path())
		}
		if n.OpID != opID {
			t.Errorf("notification op-id = %q, want %q", n.OpID, opID)
		}
		if n.Author != "daan" {
			t.Errorf("notification author = %q, want daan", n.Author)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no notification received on the inbox")
	}

	// The op payload records the mention (surfaced on the materialised contribution).
	view, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Contributions) != 1 {
		t.Fatalf("contributions = %d, want 1", len(view.Contributions))
	}
	if got := view.Contributions[0].Mentions; len(got) != 1 || got[0] != "bookkeeper-agent" {
		t.Errorf("mentions = %v, want [bookkeeper-agent]", got)
	}
}

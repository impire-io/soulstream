package topic

import (
	"context"
	"testing"
	"time"
)

func waitForView(t *testing.T, views <-chan *MaterializedTopic, pred func(*MaterializedTopic) bool) *MaterializedTopic {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case mt := <-views:
			if pred(mt) {
				return mt
			}
		case <-timeout:
			t.Fatal("timed out waiting for the expected view")
			return nil
		}
	}
}

func TestFollowLive(t *testing.T) {
	ctx := context.Background()
	url := testServer(t)

	c1 := connectClient(t, url, "daan")
	if _, err := c1.Provision(ctx); err != nil {
		t.Fatal(err)
	}
	c2 := connectClient(t, url, "bookkeeper-agent")

	h, err := StartTopic(ctx, c1, StartTopicInput{Name: "live topic"})
	if err != nil {
		t.Fatal(err)
	}

	follower := Open(c2, h.Path())
	fctx, cancel := context.WithCancel(ctx)
	defer cancel()

	views := make(chan *MaterializedTopic, 32)
	done := make(chan error, 1)
	go func() {
		done <- follower.Follow(fctx, func(mt *MaterializedTopic) { views <- mt })
	}()

	// The follower first replays history: the baseline (proposed, no contributions).
	waitForView(t, views, func(mt *MaterializedTopic) bool {
		return mt.Lifecycle == Proposed && len(mt.Contributions) == 0
	})

	// A turn posted from the other connection must arrive live.
	if _, err := h.PostTurn(ctx, "hello from daan"); err != nil {
		t.Fatal(err)
	}
	mt := waitForView(t, views, func(mt *MaterializedTopic) bool {
		return len(mt.Contributions) == 1 && mt.Contributions[0].Body == "hello from daan"
	})
	if len(mt.Frontier) != 1 {
		t.Errorf("frontier not advanced to a single leaf: %v", mt.Frontier)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Follow returned error after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Follow did not return after cancel")
	}
}

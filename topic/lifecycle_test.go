package topic

import (
	"context"
	"fmt"
	"testing"
)

func TestLifecycleDerivation(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "lc"})
	if err != nil {
		t.Fatal(err)
	}

	mt, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mt.Lifecycle != Proposed {
		t.Errorf("initial lifecycle = %q, want proposed", mt.Lifecycle)
	}

	if _, err := h.PostTurn(ctx, "starting work"); err != nil {
		t.Fatal(err)
	}
	if mt, err = h.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	if mt.Lifecycle != Active {
		t.Errorf("after a turn, lifecycle = %q, want active", mt.Lifecycle)
	}

	if _, err := h.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}
	if mt, err = h.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	if mt.Lifecycle != Closed {
		t.Errorf("after close, lifecycle = %q, want closed", mt.Lifecycle)
	}
}

func TestLifecycleRejectsUndefined(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "lc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Transition(ctx, Lifecycle("dormant")); err == nil {
		t.Error("transition to an undefined state should be rejected")
	}
}

func TestLifecycleConcurrentCloseConverges(t *testing.T) {
	ctx := context.Background()
	url := testServer(t)
	c1 := connectClient(t, url, "daan")
	if _, err := c1.Provision(ctx); err != nil {
		t.Fatal(err)
	}
	c2 := connectClient(t, url, "architect")

	h, err := StartTopic(ctx, c1, StartTopicInput{Name: "lc"})
	if err != nil {
		t.Fatal(err)
	}

	h1, h2 := Open(c1, h.Path()), Open(c2, h.Path())
	if _, err := h1.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := h2.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := h1.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}
	if _, err := h2.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}

	mt, err := Open(c1, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mt.Lifecycle != Closed {
		t.Errorf("converged lifecycle = %q, want closed", mt.Lifecycle)
	}
}

func TestPostToClosedWarnsNotBlocks(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "lc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Transition(ctx, Closed); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Materialise(ctx); err != nil { // handle now knows it is closed
		t.Fatal(err)
	}

	var warned []string
	old := warnf
	warnf = func(format string, args ...any) { warned = append(warned, fmt.Sprintf(format, args...)) }
	defer func() { warnf = old }()

	// Not blocked — the post succeeds.
	if _, err := h.PostTurn(ctx, "still typing after close"); err != nil {
		t.Fatalf("post to closed topic should succeed (warn, not block): %v", err)
	}
	if len(warned) == 0 {
		t.Error("posting to a closed topic did not warn")
	}
}

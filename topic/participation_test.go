package topic

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// TestParticipationWalkthrough mirrors specs/003-participation/quickstart.md: mention a
// persona (delivered to their inbox) and attach a file (materialised, retrievable,
// digest-verifiable).
func TestParticipationWalkthrough(t *testing.T) {
	ctx := context.Background()
	url := testServer(t)
	c := connectClient(t, url, "daan")
	if _, err := c.Provision(ctx); err != nil {
		t.Fatal(err)
	}
	agent := connectClient(t, url, "bookkeeper-agent")

	h, err := StartTopic(ctx, c, StartTopicInput{Name: "Q2 VAT filing"})
	if err != nil {
		t.Fatal(err)
	}

	// Mention → inbox.
	fctx, cancel := context.WithCancel(ctx)
	defer cancel()
	notes := make(chan Notification, 4)
	go func() { _ = FollowInbox(fctx, agent, "bookkeeper-agent", nil, func(n Notification) { notes <- n }) }()

	if _, err := h.PostTurn(ctx, "numbers are in, @bookkeeper-agent please check box 5"); err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-notes:
		if n.Author != "daan" || n.Topic != h.Path() {
			t.Errorf("notification = %+v", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("mention notification not received")
	}

	// Attach → materialise → get + verify.
	data := []byte("id,amount\n1,100\n")
	if _, err := h.Attach(ctx, "q2-lines.csv", "text/csv", data, ""); err != nil {
		t.Fatal(err)
	}
	view, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(view.Attachments))
	}
	a := view.Attachments[0]
	got, err := GetAttachment(ctx, c, a.Object)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) || !VerifyDigest(got, a.Digest) {
		t.Error("attachment did not round-trip and verify")
	}
}

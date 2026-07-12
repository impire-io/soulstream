package topic

import (
	"bytes"
	"context"
	"testing"
)

func TestAttachMaterialiseRetrieve(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "vat"})
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("id,amount\n1,100\n")
	if _, err := h.Attach(ctx, "q2.csv", "text/csv", data, ""); err != nil {
		t.Fatal(err)
	}

	view, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if view.Lifecycle != Active {
		t.Errorf("lifecycle = %q, want active", view.Lifecycle)
	}
	if len(view.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(view.Attachments))
	}
	a := view.Attachments[0]
	if a.Name != "q2.csv" || a.ContentType != "text/csv" || a.Size != uint64(len(data)) {
		t.Errorf("attachment metadata wrong: %+v", a)
	}
	if a.Digest == "" || !VerifyDigest(data, a.Digest) {
		t.Errorf("digest not recorded/verifiable: %q", a.Digest)
	}

	// Retrieve the exact bytes and verify.
	got, err := GetAttachment(ctx, c, a.Object)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("retrieved bytes differ from the original")
	}
	if !VerifyDigest(got, a.Digest) {
		t.Error("retrieved bytes fail digest verification")
	}
	if VerifyDigest([]byte("tampered"), a.Digest) {
		t.Error("tampered bytes should fail digest verification")
	}
}

func TestAttachEmptyNameRejected(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "vat"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Attach(ctx, "   ", "text/plain", []byte("x"), ""); err == nil {
		t.Error("an empty attachment name should be rejected")
	}
}

func TestAttachZeroByteAllowed(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "vat"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Attach(ctx, "empty.bin", "application/octet-stream", []byte{}, ""); err != nil {
		t.Errorf("a zero-byte attachment should be allowed: %v", err)
	}
}

func TestAttachDanglingAnchor(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "vat"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Attach(ctx, "x.txt", "text/plain", []byte("hi"), "no-such-op"); err != nil {
		t.Fatal(err)
	}
	view, err := h.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Attachments) != 1 || !view.Attachments[0].Dangling {
		t.Errorf("attachment anchored to a missing op not flagged dangling: %+v", view.Attachments)
	}
}

func TestGetAttachmentNotFound(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "daan")
	if _, err := GetAttachment(ctx, c, "attachments/nope/does-not-exist"); err == nil {
		t.Error("fetching a missing object should return an error")
	}
}

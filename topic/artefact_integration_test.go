package topic

import (
	"bytes"
	"context"
	"testing"
)

// TestArtefactWalkthrough (US1): attach, revise twice from two personas, read cold —
// one artefact, three revisions in order with correct authors, tip last; every
// revision's bytes still fetchable and digest-verifiable; identical after a rollup.
func TestArtefactWalkthrough(t *testing.T) {
	ctx := context.Background()
	url := testServer(t)
	daan := connectClient(t, url, "daan")
	if _, err := daan.Provision(ctx); err != nil {
		t.Fatal(err)
	}
	scribe := connectClient(t, url, "scribe")

	h, err := StartTopic(ctx, daan, StartTopicInput{Name: "design notes"})
	if err != nil {
		t.Fatal(err)
	}

	v1 := []byte("draft one")
	rootID, err := h.Attach(ctx, "notes.md", "text/markdown", v1, "")
	if err != nil {
		t.Fatal(err)
	}

	// scribe revises the tip; then daan revises scribe's revision.
	sh := Open(scribe, h.Path())
	if _, err := sh.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	v2 := []byte("draft two — scribe's pass")
	rev2, err := sh.Revise(ctx, "notes.md", "text/markdown", v2, rootID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Materialise(ctx); err != nil {
		t.Fatal(err)
	}
	v3 := []byte("draft three — final")
	if _, err := h.Revise(ctx, "notes.md", "text/markdown", v3, rev2); err != nil {
		t.Fatal(err)
	}

	verify := func(mt *MaterializedTopic) Artefact {
		t.Helper()
		arts := mt.Artefacts()
		if len(arts) != 1 {
			t.Fatalf("artefacts = %d, want 1", len(arts))
		}
		a := arts[0]
		if a.Root != rootID || len(a.Revisions) != 3 {
			t.Fatalf("lineage = root %s, %d revisions", a.Root, len(a.Revisions))
		}
		authors := []string{a.Revisions[0].Author, a.Revisions[1].Author, a.Revisions[2].Author}
		if authors[0] != "daan" || authors[1] != "scribe" || authors[2] != "daan" {
			t.Errorf("revision authors = %v", authors)
		}
		if a.Tip.OpID != a.Revisions[2].OpID {
			t.Errorf("tip is not the latest revision")
		}
		// Tip bytes and the oldest revision's bytes both fetch and verify.
		tipData, err := GetAttachment(ctx, daan, a.Tip.Object)
		if err != nil || !bytes.Equal(tipData, v3) || !VerifyDigest(tipData, a.Tip.Digest) {
			t.Errorf("tip fetch/verify failed: %v", err)
		}
		oldData, err := GetAttachment(ctx, daan, a.Revisions[0].Object)
		if err != nil || !bytes.Equal(oldData, v1) || !VerifyDigest(oldData, a.Revisions[0].Digest) {
			t.Errorf("old revision fetch/verify failed: %v", err)
		}
		return a
	}

	// A cold reader agrees with the writer.
	cold := Open(scribe, h.Path())
	mt, err := cold.Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before := verify(mt)

	// And a rollup changes nothing an artefact reader can see.
	if _, err := h.Rollup(ctx); err != nil {
		t.Fatal(err)
	}
	mt2, err := Open(scribe, h.Path()).Materialise(ctx)
	if err != nil {
		t.Fatal(err)
	}
	after := verify(mt2)
	if before.Tip.OpID != after.Tip.OpID || before.Root != after.Root {
		t.Error("rollup changed the artefact's identity or tip")
	}

	// Revise with no predecessor is refused — Attach is the way to start fresh.
	if _, err := h.Revise(ctx, "notes.md", "text/markdown", v3, ""); err == nil {
		t.Error("Revise accepted an empty predecessor")
	}
}

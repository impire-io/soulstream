package topic

import (
	"errors"
	"strings"
	"testing"
)

// att builds a view attachment for derivation tests (pure — no fold needed).
func att(opID, author, name, anchor string) Attachment {
	return Attachment{OpID: opID, Author: author, Name: name, Object: "obj-" + opID, Digest: "d", Size: 1, Anchor: anchor}
}

// TestArtefactsGrouping: anchor-to-attachment extends the lineage; everything else
// roots its own — including an anchor to a non-attachment op.
func TestArtefactsGrouping(t *testing.T) {
	mt := &MaterializedTopic{Attachments: []Attachment{
		att("a1", "daan", "notes.md", ""),
		att("a2", "scribe", "notes.md", "a1"),
		att("b1", "daan", "photo.png", "turn-9"), // anchored to a turn: contextual, own root
		att("a3", "daan", "notes.md", "a2"),
	}}
	arts := mt.Artefacts()
	if len(arts) != 2 {
		t.Fatalf("artefacts = %d, want 2", len(arts))
	}
	notes := arts[0]
	if notes.Root != "a1" || len(notes.Revisions) != 3 || notes.Tip.OpID != "a3" {
		t.Errorf("notes lineage wrong: %+v", notes)
	}
	if arts[1].Root != "b1" || len(arts[1].Revisions) != 1 {
		t.Errorf("photo lineage wrong: %+v", arts[1])
	}
}

// TestArtefactsMidChainAnchor: anchoring to any member joins that member's root —
// a concurrent revision of an old tip is the same artefact, and the tip is still
// the member latest in stream order.
func TestArtefactsMidChainAnchor(t *testing.T) {
	mt := &MaterializedTopic{Attachments: []Attachment{
		att("a1", "daan", "spec.md", ""),
		att("a2", "scribe", "spec.md", "a1"),
		att("a3", "daan", "spec.md", "a1"), // concurrent with a2, anchored mid-chain
	}}
	arts := mt.Artefacts()
	if len(arts) != 1 {
		t.Fatalf("mid-chain anchor split the lineage: %d artefacts", len(arts))
	}
	if arts[0].Tip.OpID != "a3" {
		t.Errorf("tip = %s, want a3 (latest in stream order)", arts[0].Tip.OpID)
	}
	if len(arts[0].Revisions) != 3 {
		t.Errorf("revisions = %d, want 3 (both branches kept)", len(arts[0].Revisions))
	}
}

// TestArtefactsDanglingAnchorIsRoot: an anchor naming an op the topic does not
// hold starts a lineage — never an error.
func TestArtefactsDanglingAnchorIsRoot(t *testing.T) {
	mt := &MaterializedTopic{Attachments: []Attachment{att("a1", "daan", "x.md", "ghost")}}
	arts := mt.Artefacts()
	if len(arts) != 1 || arts[0].Root != "a1" {
		t.Errorf("dangling-anchored attachment did not root itself: %+v", arts)
	}
}

// TestArtefactsRename: the display name follows the tip — a rename is a revision.
func TestArtefactsRename(t *testing.T) {
	mt := &MaterializedTopic{Attachments: []Attachment{
		att("a1", "daan", "draft.md", ""),
		att("a2", "daan", "final.md", "a1"),
	}}
	arts := mt.Artefacts()
	if arts[0].Name != "final.md" {
		t.Errorf("artefact name = %q, want the tip's", arts[0].Name)
	}
}

// TestArtefactsCycleGuard: a crafted anchor cycle terminates deterministically as
// one lineage instead of hanging the derivation.
func TestArtefactsCycleGuard(t *testing.T) {
	mt := &MaterializedTopic{Attachments: []Attachment{
		att("a1", "daan", "x.md", "a2"),
		att("a2", "daan", "x.md", "a1"),
	}}
	arts := mt.Artefacts()
	if len(arts) != 1 || len(arts[0].Revisions) != 2 {
		t.Errorf("cycle did not collapse into one lineage: %+v", arts)
	}
}

// TestFindArtefact: resolution by root id, member id, and unique name; ambiguity
// names the candidate roots; a miss is a plain error.
func TestFindArtefact(t *testing.T) {
	mt := &MaterializedTopic{Path: "tp", Attachments: []Attachment{
		att("a1", "daan", "notes.md", ""),
		att("a2", "scribe", "notes.md", "a1"),
		att("b1", "daan", "notes.md", ""), // second lineage with the same display name
		att("c1", "daan", "logo.svg", ""),
	}}

	if a, err := FindArtefact(mt, "a1"); err != nil || a.Root != "a1" {
		t.Errorf("by root: %v %v", a, err)
	}
	if a, err := FindArtefact(mt, "a2"); err != nil || a.Root != "a1" {
		t.Errorf("by member: %v %v", a, err)
	}
	if a, err := FindArtefact(mt, "logo.svg"); err != nil || a.Root != "c1" {
		t.Errorf("by unique name: %v %v", a, err)
	}
	_, err := FindArtefact(mt, "notes.md")
	if !errors.Is(err, ErrAmbiguousArtefact) {
		t.Fatalf("ambiguous name error = %v", err)
	}
	if !strings.Contains(err.Error(), "a1") || !strings.Contains(err.Error(), "b1") {
		t.Errorf("ambiguity error does not name the roots: %v", err)
	}
	if _, err := FindArtefact(mt, "missing.md"); err == nil || errors.Is(err, ErrAmbiguousArtefact) {
		t.Errorf("miss error = %v", err)
	}
}

// TestArtefactsRoundTripAcrossRollup (FR-006): lineages, order, authorship, and
// tip are identical whether derived from full history or from a baked baseline —
// including a revision landing after the compaction.
func TestArtefactsRoundTripAcrossRollup(t *testing.T) {
	logRecs := fullLog()
	before := apply("t", logRecs)
	tail := foldRec(101, "attn-3", "scribe", TypeAttachmentAdd,
		AttachmentPayload{Name: "n-final.txt", Object: "o3", Digest: "d3", Size: 5, Anchor: "attn-2"},
		before.Frontier...)

	full := apply("t", append(fullLog(), tail))
	baked := apply("t", []SeqRecord{rollupOf(fullLog(), "base-1"), tail})

	fa, ba := full.Artefacts(), baked.Artefacts()
	if len(fa) != len(ba) || len(fa) == 0 {
		t.Fatalf("artefact counts differ: full %d, baked %d", len(fa), len(ba))
	}
	for i := range fa {
		if fa[i].Root != ba[i].Root || fa[i].Tip.OpID != ba[i].Tip.OpID || fa[i].Name != ba[i].Name {
			t.Errorf("artefact %d differs: full %+v, baked %+v", i, fa[i], ba[i])
		}
		if len(fa[i].Revisions) != len(ba[i].Revisions) {
			t.Errorf("artefact %d revision counts differ", i)
			continue
		}
		for j := range fa[i].Revisions {
			if fa[i].Revisions[j].OpID != ba[i].Revisions[j].OpID || fa[i].Revisions[j].Author != ba[i].Revisions[j].Author {
				t.Errorf("artefact %d revision %d differs", i, j)
			}
		}
	}
	if fa[0].Tip.OpID != "attn-3" {
		t.Errorf("post-rollup revision did not become the tip: %s", fa[0].Tip.OpID)
	}
}

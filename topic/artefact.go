package topic

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Versioned artefacts (stage 1 of the work extension): a document is a
// topic-anchored attachment revised whole-file. Each revision is an ordinary
// attachment.add whose anchor points at a prior attachment op — that is the
// entire mechanism. An artefact is the derived lineage: nothing here is folded
// or persisted; the projection rule below is computed from the attachments any
// view already carries, so it survives compaction by construction.

// ErrAmbiguousArtefact means a display name matched more than one lineage.
// Names are labels, not keys — the lineage's root op-id is the stable handle.
var ErrAmbiguousArtefact = errors.New("topic: artefact name is ambiguous")

// Artefact is a lineage of whole-file revisions of one document within a topic.
// Identity is the root revision's op-id; the display name is the tip's file name
// (a rename is a revision like any other change).
type Artefact struct {
	Root      string       `json:"root"`
	Name      string       `json:"name"`
	Tip       Attachment   `json:"tip"`
	Revisions []Attachment `json:"revisions"` // stream order, root first
}

// Artefacts groups the topic's attachments into lineages. An attachment whose
// anchor resolves to another attachment extends that attachment's lineage
// (anchoring to any member joins that member's root); every other attachment —
// unanchored, anchored to a non-attachment op, or dangling — is a lineage of its
// own. The tip is the lineage member latest in stream order: attachment slice
// order is fold order, and baked entries precede the tail in their original
// order, so slice position is the one comparison that stays correct across
// compaction (StreamSeq is zeroed on baked entries by design).
func (mt *MaterializedTopic) Artefacts() []Artefact {
	n := len(mt.Attachments)
	if n == 0 {
		return nil
	}

	idx := make(map[string]int, n)
	for i, a := range mt.Attachments {
		idx[a.OpID] = i
	}
	parent := make([]int, n)
	for i, a := range mt.Attachments {
		parent[i] = -1
		if a.Anchor != "" {
			if j, ok := idx[a.Anchor]; ok && j != i {
				parent[i] = j
			}
		}
	}

	// Resolve each attachment to its lineage root by walking parents. A cycle
	// (possible only in a crafted log) breaks deterministically at the first
	// node the walk revisits.
	const unresolved = -1
	root := make([]int, n)
	for i := range root {
		root[i] = unresolved
	}
	for i := 0; i < n; i++ {
		if root[i] != unresolved {
			continue
		}
		var path []int
		onPath := make(map[int]bool)
		cur := i
		for root[cur] == unresolved {
			if onPath[cur] || parent[cur] < 0 {
				root[cur] = cur
				break
			}
			onPath[cur] = true
			path = append(path, cur)
			cur = parent[cur]
		}
		for _, node := range path {
			root[node] = root[cur]
		}
	}

	members := map[int][]int{} // root index → member indices, ascending
	for i := 0; i < n; i++ {
		members[root[i]] = append(members[root[i]], i)
	}
	roots := make([]int, 0, len(members))
	for r := range members {
		roots = append(roots, r)
	}
	sort.Ints(roots)

	out := make([]Artefact, 0, len(roots))
	for _, r := range roots {
		revs := make([]Attachment, len(members[r]))
		for k, m := range members[r] {
			revs[k] = mt.Attachments[m]
		}
		// The tip is the newest revision still standing: withdrawn (removed)
		// revisions stay in the history but are never served as current. A
		// lineage with nothing standing is a withdrawn document — not listed.
		tipAt := -1
		for k := len(revs) - 1; k >= 0; k-- {
			if !revs[k].Removed {
				tipAt = k
				break
			}
		}
		if tipAt < 0 {
			continue
		}
		tip := revs[tipAt]
		out = append(out, Artefact{
			Root: mt.Attachments[r].OpID, Name: tip.Name, Tip: tip, Revisions: revs,
		})
	}
	return out
}

// FindArtefact resolves ref against the view's artefacts: a root op-id, any
// revision's op-id, or a display name. A name matching several lineages returns
// ErrAmbiguousArtefact naming the candidate roots.
func FindArtefact(mt *MaterializedTopic, ref string) (Artefact, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Artefact{}, fmt.Errorf("topic: artefact reference must not be empty")
	}
	arts := mt.Artefacts()
	for _, a := range arts {
		if a.Root == ref {
			return a, nil
		}
		for _, r := range a.Revisions {
			if r.OpID == ref {
				return a, nil
			}
		}
	}
	var matches []Artefact
	for _, a := range arts {
		if a.Name == ref {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return Artefact{}, fmt.Errorf("topic: no artefact in %s matches %q", mt.Path, ref)
	default:
		roots := make([]string, len(matches))
		for i, m := range matches {
			roots[i] = m.Root
		}
		return Artefact{}, fmt.Errorf("%w: %q names lineages rooted at %s — use a root op-id",
			ErrAmbiguousArtefact, ref, strings.Join(roots, ", "))
	}
}

// Revise attaches data as a new whole-file revision superseding predecessor (an
// attachment op-id in this topic — usually the current tip). It is Attach with
// the anchor required: anchoring an attachment to an attachment is what makes it
// a revision.
func (h *Handle) Revise(ctx context.Context, name, contentType string, data []byte, predecessor string) (string, error) {
	if strings.TrimSpace(predecessor) == "" {
		return "", fmt.Errorf("topic: a revision needs its predecessor's op-id (use Attach to start a fresh artefact)")
	}
	return h.Attach(ctx, name, contentType, data, predecessor)
}

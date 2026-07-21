package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/topic"
)

// sigGlyph renders a verification status as a one-character marker: ✓ verified,
// ✗ failed, ? unknown-key, nothing for unsigned (a realm that never signed should
// look exactly as it always did). The leading space keeps columns tidy.
func sigGlyph(s topic.SigStatus) string {
	switch s {
	case topic.SigVerified:
		return " ✓"
	case topic.SigFailed:
		return " ✗"
	case topic.SigUnknownKey:
		return " ?"
	default:
		return ""
	}
}

// warnDistrusted prints the substitution-attack banner for every distrusted persona,
// on stdout (first, unmissable) and mirrored to stderr (machine-distinguishable).
func warnDistrusted(stdout, stderr io.Writer, kr *identity.Keyring) {
	if kr == nil {
		return
	}
	var names []string
	for name, distrusted := range kr.Distrusted {
		if distrusted {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		line := fmt.Sprintf("!! possible key substitution for %s — signatures from this persona are not trusted", name)
		fmt.Fprintln(stdout, line)
		fmt.Fprintln(stderr, line)
	}
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func renderReport(w io.Writer, r *realm.ProvisionReport) {
	for _, res := range r.Results {
		fmt.Fprintf(w, "%-13s %s", res.Artefact, res.Outcome)
		if len(res.Nonconformities) > 0 {
			fmt.Fprintf(w, " %v", res.Nonconformities)
		}
		fmt.Fprintln(w)
	}
}

func renderBoard(w io.Writer, entries []topic.BoardEntry) {
	for _, e := range entries {
		lc := string(e.Lifecycle)
		if lc == "" {
			lc = "-"
		}
		fmt.Fprintf(w, "%-9s %-40s %s\n", lc, e.Path, e.Announcement.Name)
	}
}

func renderView(w io.Writer, v *topic.MaterializedTopic) {
	fmt.Fprintf(w, "topic:     %s\n", v.Path)
	if v.Announcement != nil {
		fmt.Fprintf(w, "name:      %s\n", v.Announcement.Name)
		if v.Announcement.SubjectMatter != "" {
			fmt.Fprintf(w, "about:     %s\n", v.Announcement.SubjectMatter)
		}
	}
	fmt.Fprintf(w, "lifecycle: %s\n", v.Lifecycle)
	if v.Malformed != "" {
		fmt.Fprintf(w, "malformed: %s\n", v.Malformed)
	}
	if len(v.Contributions) > 0 {
		fmt.Fprintln(w, "contributions:")
		for _, c := range v.Contributions {
			renderContribution(w, c)
		}
	}
	if len(v.Attachments) > 0 {
		fmt.Fprintln(w, "attachments:")
		for _, a := range v.Attachments {
			fmt.Fprintf(w, "  [%s]%s %s (%s, %d bytes) object=%s\n",
				shortID(a.OpID), sigGlyph(a.Sig), a.Name, a.ContentType, a.Size, a.Object)
		}
	}
}

func renderContribution(w io.Writer, c topic.Contribution) {
	kind := "turn"
	if c.Type == topic.TypeCommentAdd {
		kind = "comment"
	}
	fmt.Fprintf(w, "  [%s]%s %s (%s", shortID(c.OpID), sigGlyph(c.Sig), c.Author, kind)
	if c.Anchor != "" {
		fmt.Fprintf(w, " -> %s", shortID(c.Anchor))
		if c.Dangling {
			fmt.Fprint(w, " dangling")
		}
	}
	fmt.Fprintf(w, "): %s", c.Body)
	if len(c.Mentions) > 0 {
		fmt.Fprintf(w, "  mentions=%v", c.Mentions)
	}
	fmt.Fprintln(w)
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/topic"
)

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
			fmt.Fprintf(w, "  [%s] %s (%s, %d bytes) object=%s\n",
				shortID(a.OpID), a.Name, a.ContentType, a.Size, a.Object)
		}
	}
}

func renderContribution(w io.Writer, c topic.Contribution) {
	kind := "turn"
	if c.Type == topic.TypeCommentAdd {
		kind = "comment"
	}
	fmt.Fprintf(w, "  [%s] %s (%s", shortID(c.OpID), c.Author, kind)
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

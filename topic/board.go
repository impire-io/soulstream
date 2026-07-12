package topic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/record"
)

// BoardEntry is one topic on the discovery board.
type BoardEntry struct {
	Path         string
	Announcement Announcement
	Parent       string
	ParentKnown  bool
	Lifecycle    Lifecycle
}

// Board replays the realm's info board and returns one entry per topic — the latest
// announcement per info subject. An empty realm yields an empty board, not an error. A
// sub-topic whose parent is absent is flagged (ParentKnown == false), never dropped.
func Board(ctx context.Context, c *realm.Client) ([]BoardEntry, error) {
	stream, err := c.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		return nil, fmt.Errorf("topic: look up stream: %w", err)
	}

	info, err := stream.Info(ctx, jetstream.WithSubjectFilter(InfoSubjectWildcard))
	if err != nil {
		return nil, fmt.Errorf("topic: stream info: %w", err)
	}

	paths := make([]string, 0, len(info.State.Subjects))
	known := make(map[string]bool, len(info.State.Subjects))
	for subj := range info.State.Subjects {
		path := strings.TrimPrefix(subj, InfoSubjectPrefix)
		paths = append(paths, path)
		known[path] = true
	}
	sort.Strings(paths)

	entries := make([]BoardEntry, 0, len(paths))
	for _, path := range paths {
		raw, err := stream.GetLastMsgForSubject(ctx, InfoSubject(path))
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgNotFound) {
				continue
			}
			return nil, fmt.Errorf("topic: last announce for %s: %w", path, err)
		}

		rec, err := record.Parse(raw.Header, raw.Data)
		if err != nil {
			continue // skip a malformed announcement rather than fail the whole board
		}
		var ap AnnouncePayload
		_ = json.Unmarshal(rec.Payload, &ap)

		parent := ParentPath(path)
		entry := BoardEntry{
			Path: path,
			Announcement: Announcement{
				TopicID:       ap.TopicID,
				Name:          ap.Name,
				SubjectMatter: ap.SubjectMatter,
				Parent:        ap.Parent,
				Expected:      ap.Expected,
				Tags:          ap.Tags,
			},
			Parent:      parent,
			ParentKnown: parent == "" || known[parent],
		}

		// Lifecycle where derivable: materialise the topic's ops.
		if recs, err := drainOps(ctx, c, path); err == nil {
			entry.Lifecycle = apply(path, recs).Lifecycle
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

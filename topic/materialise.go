package topic

import "context"

// Materialise drains the topic's ops backlog and returns the current view, updating the
// handle's observed frontier and lifecycle so subsequent posts parent correctly. It is a
// pure replay: two consumers replaying the same log produce the same view.
func (h *Handle) Materialise(ctx context.Context) (*MaterializedTopic, error) {
	recs, err := drainOps(ctx, h.client, h.path)
	if err != nil {
		return nil, err
	}
	mt := apply(h.path, recs)
	baselineID := ""
	if len(recs) > 0 {
		baselineID = recs[0].Record.ID
	}
	annotateView(mt, annotate(recs, h.client.Realm(), h.path, h.keyring), baselineID)

	// Enrich the view with the topic's announcement (from its INFO subject).
	if ann, annRec, err := fetchAnnouncement(ctx, h.client, h.path); err == nil && ann != nil {
		ann.Sig = VerifyRecord(*annRec, h.client.Realm(), h.path, h.keyring)
		mt.Announcement = ann
	}

	h.adopt(mt)
	return mt, nil
}

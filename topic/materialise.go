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
	h.adopt(mt)
	return mt, nil
}

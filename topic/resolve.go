package topic

// ContainsOp reports whether an op-id resolves in the topic's current state: as a
// live op or baked into the current baseline. It scans every op-id the view
// exposes — the announcement, contributions and their edit stamps, attachments,
// work items and their timelines, the current baseline checkpoint, and the
// frontier. Ops that compaction consumes without baking an id (marks, transitions,
// superseded chain interiors) honestly do not resolve — state-level history
// survives a rollup, op-level forensics don't; exhibits exist for those.
func (mt *MaterializedTopic) ContainsOp(opID string) bool {
	if mt == nil || opID == "" {
		return false
	}
	if mt.BaselineID == opID {
		return true
	}
	if mt.Announcement != nil && mt.Announcement.OpID == opID {
		return true
	}
	for _, c := range mt.Contributions {
		if c.OpID == opID {
			return true
		}
		for _, e := range c.Edits {
			if e.OpID == opID {
				return true
			}
		}
	}
	for _, a := range mt.Attachments {
		if a.OpID == opID {
			return true
		}
	}
	for _, w := range mt.WorkItems {
		if w.ID == opID {
			return true
		}
		for _, ev := range w.Timeline {
			if ev.OpID == opID {
				return true
			}
		}
	}
	for _, id := range mt.Frontier {
		if id == opID {
			return true
		}
	}
	return false
}

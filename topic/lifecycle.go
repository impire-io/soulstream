package topic

import (
	"context"
	"fmt"
)

// definedLifecycle reports whether l is a state the MVP can transition to.
func definedLifecycle(l Lifecycle) bool {
	switch l {
	case Proposed, Active, Closed:
		return true
	default:
		return false
	}
}

// Transition posts a life.transition to the given state. It rejects a state the MVP does
// not define (dormant and archived are deferred), naming the allowed states.
//
// The resulting lifecycle is derived from the log when the topic is materialised, so two
// personas transitioning to the same state concurrently converge — the ops merge
// harmlessly and there is no arbiter.
func (h *Handle) Transition(ctx context.Context, to Lifecycle) (string, error) {
	if !definedLifecycle(to) {
		return "", fmt.Errorf(
			"topic: undefined lifecycle transition %q; allowed states are %q, %q, %q",
			to, Proposed, Active, Closed)
	}
	return h.Post(ctx, TypeLifeTransition, TransitionPayload{To: to})
}

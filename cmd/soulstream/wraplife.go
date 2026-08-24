// The wrap's housekeeping around the harness loop — upstream ask #3 of
// the hq's shell design 0008 (episode 0124): the agent announces itself
// and lights its lamp. Both are composition: the directory floor is the
// registry's, the lease is the presence convention's, and everything
// here is advisory — a failure is a log line, never a refusal to answer
// mentions.

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/impire-io/soulstream-core/presence"
	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/registry"
)

// ensureProfile makes the persona findable in the directory without ever
// speaking over it: lookup first, publish only into absence. The floor is
// minimal and honest — this lane holds no signing key, so the profile
// carries none, and a richer profile (display name, attestation) stays
// the agent's own act through its harness. registry.Publish REPLACES
// display metadata on an existing entry, which is exactly why absence is
// checked first.
func ensureProfile(ctx context.Context, c *realm.Client, log *slog.Logger) {
	persona := c.Persona()
	_, found, err := registry.Lookup(ctx, c, persona)
	if err != nil {
		log.Warn("directory unreadable; the agent stays unlisted", "persona", persona, "err", err)
		return
	}
	if found {
		return
	}
	p := registry.Profile{Name: persona, CreatedAt: time.Now().UTC()}
	if err := registry.Publish(ctx, c, p); err != nil {
		log.Warn("directory entry not created; the agent stays unlisted", "persona", persona, "err", err)
	}
}

// holdPresence lights the lamp: the presence lease held for as long as
// ctx lives, renewed on the convention's cadence, with the farewell
// written by Hold itself on a fresh short-lived context once ctx ends.
// The returned wait lets cmdWrap hold the door: the goodbye must land
// before the deferred client.Close tears the connection down, and the
// bound is Hold's own farewell window plus a margin, so a wedged write
// can never keep the process alive.
func holdPresence(ctx context.Context, c *realm.Client, log *slog.Logger) (wait func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := presence.Hold(ctx, c, presence.Entry{Doing: "answering mentions"})
		if err != nil {
			log.Warn("the presence lease failed; mentions are unaffected", "persona", c.Persona(), "err", err)
		}
	}()
	return func() {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			log.Warn("the farewell did not land in time; the face will read last-seen")
		}
	}
}

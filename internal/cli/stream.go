package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/topic"
)

func cmdWatch(ctx context.Context, connect Connector, cfg Config, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: soulstream watch <path>")
		return 2
	}
	path := args[0]
	return withClient(ctx, connect, cfg, false, stderr, func(c *realm.Client) error {
		printed := 0
		return topic.Open(c, path).Follow(ctx, func(v *topic.MaterializedTopic) {
			for i := printed; i < len(v.Contributions); i++ {
				renderContribution(stdout, v.Contributions[i])
			}
			printed = len(v.Contributions)
		})
	})
}

func cmdInbox(ctx context.Context, connect Connector, cfg Config, _ []string, stdout, stderr io.Writer) int {
	return withClient(ctx, connect, cfg, true, stderr, func(c *realm.Client) error {
		return topic.FollowInbox(ctx, c, cfg.Persona, func(n topic.Notification) {
			fmt.Fprintf(stdout, "mention in %s (op %s) by %s\n", n.Topic, shortID(n.OpID), n.Author)
		})
	})
}

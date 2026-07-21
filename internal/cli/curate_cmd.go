package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/impire/soulstream/curator"
	"github.com/impire/soulstream/realm"
)

// cmdCurate runs the curator under the session persona until interrupted: warm
// content-aware discovery answering, duplicate flags, and dormancy proposals — all
// ordinary comments, all suggestions.
func cmdCurate(ctx context.Context, connect Connector, cfg Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("curate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	idle := fs.Duration("idle", curator.DefaultIdleWindow, "quiet period before proposing closure")
	scanEvery := fs.Duration("scan-every", curator.DefaultScanEvery, "cadence of the duplicate/dormancy passes")
	if err := parseInterspersed(fs, args); err != nil {
		return 2
	}

	return withClient(ctx, connect, cfg, true, stderr, func(c *realm.Client) error {
		fmt.Fprintf(stdout, "curating as %q (Ctrl-C to stop)\n", cfg.Persona)
		return curator.Run(ctx, c, curator.Options{
			IdleWindow: *idle,
			ScanEvery:  *scanEvery,
			OnEvent:    func(e string) { fmt.Fprintln(stdout, e) },
		})
	})
}

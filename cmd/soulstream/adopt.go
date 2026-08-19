// `soulstream adopt` is the canonical-form migration (hq episode 0112):
// a realm founded before the v2 break is refused at load, and this is
// the one act that resolves it without re-founding — but only where
// re-founding would protect nothing.
//
// The break exists so a realm's history is never SILENTLY mixed: v1
// records were signed over a form that no longer exists, and they can
// never be re-signed. A realm that has never recorded a signed
// operation has nothing to mix, so adoption is exact rather than
// merciful: it counts what is on the stream, adopts an empty one, and
// refuses a populated one by name with re-founding spelled out.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream-core/realm"

	"github.com/impire-io/soulstream/ceremony"
)

func cmdAdopt(args []string, out, errw io.Writer) error {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	force := fs.Bool("force", false,
		"adopt even though the realm holds records: their signatures become legacy-shape forever")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := stateDir(*stateFlag)
	if err != nil {
		return err
	}

	pre, err := ceremony.PreV2(dir)
	if err != nil {
		return err
	}
	if !pre {
		fmt.Fprintf(out, "nothing to adopt: %s is already canonical v%d\n", dir, ceremony.RecordVersion())
		return nil
	}

	// Count what would be affected. The realm's own server is the only
	// authority on that, so adoption talks to it — reading, never writing.
	count, err := countRecords(dir)
	switch {
	case err != nil:
		return fmt.Errorf("soulstream: cannot read the realm's op-log to see what adoption would affect (%w) — start the realm's server, or re-found instead", err)
	case count > 0 && !*force:
		return fmt.Errorf("soulstream: this realm holds %d recorded operations signed under canonical v1; adopting would leave every one of them verifying as legacy-shape forever. Re-found a fresh realm (`soulstream init`) and keep this directory for reading with the matching older build — or, if you accept that cost deliberately, `soulstream adopt --force`", count)
	}

	if err := ceremony.AdoptV2(dir); err != nil {
		return err
	}
	if count > 0 {
		fmt.Fprintf(out, "adopted %s at canonical v%d — %d pre-existing operations now read legacy-shape (forced)\n",
			dir, ceremony.RecordVersion(), count)
		return nil
	}
	fmt.Fprintf(out, "adopted %s at canonical v%d — the op-log was empty, so nothing was mixed\n",
		dir, ceremony.RecordVersion())
	return nil
}

// countRecords reports how many operations the realm's op-log holds.
// It connects with the ops credential the founding left in the state
// directory and reads the stream's state — no writes, no provisioning.
func countRecords(dir string) (uint64, error) {
	url, creds, err := ceremony.OpsConnection(dir)
	if err != nil {
		return 0, err
	}
	opts := []nats.Option{nats.Name("soulstream-adopt"), nats.Timeout(5 * time.Second),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0)}
	if creds != "" {
		opts = append(opts, nats.UserCredentials(creds))
	}
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return 0, fmt.Errorf("connect %s: %w", url, err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := js.Stream(ctx, realm.StreamName)
	if err != nil {
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			return 0, nil // never provisioned: nothing recorded
		}
		return 0, err
	}
	info, err := stream.Info(ctx)
	if err != nil {
		return 0, err
	}
	return info.State.Msgs, nil
}

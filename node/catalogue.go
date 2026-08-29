package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream-core/identity"
	infercat "github.com/impire-io/soulstream-inference/catalogue"
)

// The catalogue is where a virtual model name lives (soulstream-inference
// design 0001 §5): a realm KV bucket, one key per name, the value a route
// descriptor. Re-pointing a name moves traffic — and the defaults that
// ride with it — without a single declaration changing, because a
// declaration names and never routes.
//
// The stored shape, the bucket's name and its codec are the thinking
// plane's published contract (soulstream-inference/catalogue) — one
// definition every hand consumes, this binary's verbs and the shell's
// sheet alike (hq shell design 0010, upstream ask #1). What stays here is
// the house's own posture: provisioning create-or-report, absence as an
// ordinary answer, and the record's name grammar applied at the hand —
// the plane deliberately does not know the record.
//
// Resolvers read the bucket fresh on every resolution. A watch is the
// obvious next step and a named [O] — a cache here would make
// re-pointing a name a promise the plane quietly breaks.

// EnsureCatalogue brings the catalogue bucket into existence
// create-or-report: it creates only what is missing and never touches an
// existing bucket's settings — the realm-provisioning posture, applied to
// one more artefact. The bucket's canonical shape is the contract's.
func EnsureCatalogue(ctx context.Context, js jetstream.JetStream) (jetstream.KeyValue, error) {
	kv, err := js.KeyValue(ctx, infercat.Bucket)
	switch {
	case err == nil:
		return kv, nil
	case errors.Is(err, jetstream.ErrBucketNotFound):
		kv, err = js.CreateKeyValue(ctx, infercat.Config())
		if err != nil {
			return nil, fmt.Errorf("node: create the model catalogue %q: %w", infercat.Bucket, err)
		}
		return kv, nil
	default:
		return nil, fmt.Errorf("node: look up the model catalogue %q: %w", infercat.Bucket, err)
	}
}

// CatalogueSet points a virtual name at an entry, creating the bucket if
// this is the first name the realm ever gave — an operator can seed names
// before anything serves them. The name grammar is the record's, checked
// here at the hand; the entry's own refusals are the codec's.
func CatalogueSet(ctx context.Context, js jetstream.JetStream, name string, e infercat.Entry) error {
	if err := identity.CheckName(name); err != nil {
		return fmt.Errorf("node: %q is not a model name: %w", name, err)
	}
	kv, err := EnsureCatalogue(ctx, js)
	if err != nil {
		return err
	}
	return infercat.Set(ctx, kv, name, e)
}

// CatalogueGet reads one name, reporting absence rather than erroring on
// it: a name nobody has pointed anywhere is an ordinary answer.
func CatalogueGet(ctx context.Context, js jetstream.JetStream, name string) (infercat.Entry, bool, error) {
	kv, err := js.KeyValue(ctx, infercat.Bucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return infercat.Entry{}, false, nil
	}
	if err != nil {
		return infercat.Entry{}, false, fmt.Errorf("node: look up the model catalogue: %w", err)
	}
	return infercat.Get(ctx, kv, name)
}

// CatalogueList reads every name, sorted. An absent bucket lists nothing
// — a realm that has named no models is not an error.
func CatalogueList(ctx context.Context, js jetstream.JetStream) ([]infercat.Named, error) {
	kv, err := js.KeyValue(ctx, infercat.Bucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("node: look up the model catalogue: %w", err)
	}
	return infercat.List(ctx, kv)
}

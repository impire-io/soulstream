package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream-core/identity"
	inferclient "github.com/impire-io/soulstream-inference/client"
)

// The catalogue is where a virtual model name lives (soulstream-inference
// design 0001 §5, whose [O] this feature closes): a realm KV bucket, one
// key per name, the value a route descriptor. Re-pointing a name moves
// traffic — and the defaults that ride with it — without a single
// declaration changing, because a declaration names and never routes.
//
// Resolvers read the bucket fresh on every resolution. A watch is the
// obvious next step and a named [O] — a cache here would make
// re-pointing a name a promise the plane quietly breaks.

// CatalogueBucket is the realm KV the names live in. It is a realm
// artefact like the persona directory: created create-or-report by
// whichever plane starts first, and by the `model` verb, so an operator
// can seed names before anything serves them.
const CatalogueBucket = "soulstream-inference-catalogue"

// ModelEntry is one virtual name's route descriptor as the bucket stores
// it. Capability names the pool; ModelPin, when set, resolves to the one
// instance wrapping exactly that model; Tags narrow candidates;
// DefaultParams fill gaps the request leaves — how hard to try is a
// parameter a name may default, never a route.
type ModelEntry struct {
	Capability    string            `json:"capability"`
	ModelPin      string            `json:"model_pin,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	DefaultParams map[string]any    `json:"default_params,omitempty"`
}

// Descriptor is the entry as the inference client's resolver reads it.
func (e ModelEntry) Descriptor() inferclient.Descriptor {
	return inferclient.Descriptor{
		Capability:    e.Capability,
		ModelPin:      e.ModelPin,
		Tags:          e.Tags,
		DefaultParams: e.DefaultParams,
	}
}

// EnsureCatalogue brings the catalogue bucket into existence
// create-or-report: it creates only what is missing and never touches an
// existing bucket's settings — the realm-provisioning posture, applied to
// one more artefact.
func EnsureCatalogue(ctx context.Context, js jetstream.JetStream) (jetstream.KeyValue, error) {
	kv, err := js.KeyValue(ctx, CatalogueBucket)
	switch {
	case err == nil:
		return kv, nil
	case errors.Is(err, jetstream.ErrBucketNotFound):
		kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket:      CatalogueBucket,
			Description: "virtual model names for the inference plane",
			History:     1,
		})
		if err != nil {
			return nil, fmt.Errorf("node: create the model catalogue %q: %w", CatalogueBucket, err)
		}
		return kv, nil
	default:
		return nil, fmt.Errorf("node: look up the model catalogue %q: %w", CatalogueBucket, err)
	}
}

// CatalogueSet points a virtual name at a descriptor, creating the bucket
// if this is the first name the realm ever gave.
func CatalogueSet(ctx context.Context, js jetstream.JetStream, name string, e ModelEntry) error {
	if err := identity.CheckName(name); err != nil {
		return fmt.Errorf("node: %q is not a model name: %w", name, err)
	}
	if e.Capability == "" {
		return errors.New("node: a model name needs a capability — the pool it resolves into")
	}
	kv, err := EnsureCatalogue(ctx, js)
	if err != nil {
		return err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("node: encode the entry for %q: %w", name, err)
	}
	if _, err := kv.Put(ctx, name, body); err != nil {
		return fmt.Errorf("node: write %q to the model catalogue: %w", name, err)
	}
	return nil
}

// CatalogueGet reads one name, reporting absence rather than erroring on
// it: a name nobody has pointed anywhere is an ordinary answer.
func CatalogueGet(ctx context.Context, js jetstream.JetStream, name string) (ModelEntry, bool, error) {
	kv, err := js.KeyValue(ctx, CatalogueBucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return ModelEntry{}, false, nil
	}
	if err != nil {
		return ModelEntry{}, false, fmt.Errorf("node: look up the model catalogue: %w", err)
	}
	entry, err := kv.Get(ctx, name)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return ModelEntry{}, false, nil
	}
	if err != nil {
		return ModelEntry{}, false, fmt.Errorf("node: read %q from the model catalogue: %w", name, err)
	}
	var e ModelEntry
	if err := json.Unmarshal(entry.Value(), &e); err != nil {
		return ModelEntry{}, false, fmt.Errorf("node: the catalogue entry for %q does not decode: %w", name, err)
	}
	return e, true, nil
}

// CatalogueNames is one entry as a listing shows it.
type CatalogueNames struct {
	Name  string
	Entry ModelEntry
}

// CatalogueList reads every name, sorted. An absent bucket lists nothing
// — a realm that has named no models is not an error.
func CatalogueList(ctx context.Context, js jetstream.JetStream) ([]CatalogueNames, error) {
	kv, err := js.KeyValue(ctx, CatalogueBucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("node: look up the model catalogue: %w", err)
	}
	keys, err := kv.Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("node: list the model catalogue: %w", err)
	}
	sort.Strings(keys)
	out := make([]CatalogueNames, 0, len(keys))
	for _, k := range keys {
		e, found, err := CatalogueGet(ctx, js, k)
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, CatalogueNames{Name: k, Entry: e})
		}
	}
	return out, nil
}

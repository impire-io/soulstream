package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/realm"
)

// BucketName is the persona directory's KV bucket (provisioned by realm.Provision).
const BucketName = realm.PersonasBucket

// ErrKeyConflict means a publish tried to change a persona's stored key material
// without a rotation proof. Key changes go through rotation — anything else would be
// indistinguishable from a substitution attack.
var ErrKeyConflict = errors.New("registry: profile already holds a different key (rotate instead)")

// Publish creates or updates own profile — the client persona's directory entry.
//
// It is create-or-metadata-update: a persona with no entry is created (KV Create);
// an existing entry has its display metadata replaced while the stored signing_key
// and rotations remain authoritative — an incoming nil or identical key preserves
// them, an incoming different key returns [ErrKeyConflict]. Both paths use the KV's
// own optimistic concurrency, so racing clients get an error, never a lost write.
func Publish(ctx context.Context, c *realm.Client, p Profile) error {
	if err := c.EnforceAuthor(p.Name); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	kv, err := bucket(ctx, c)
	if err != nil {
		return err
	}

	entry, err := kv.Get(ctx, p.Name)
	switch {
	case errors.Is(err, jetstream.ErrKeyNotFound):
		data, merr := json.Marshal(p)
		if merr != nil {
			return fmt.Errorf("registry: encode profile: %w", merr)
		}
		if _, cerr := kv.Create(ctx, p.Name, data); cerr != nil {
			return fmt.Errorf("registry: create profile %q: %w", p.Name, cerr)
		}
		return nil
	case err != nil:
		return fmt.Errorf("registry: read profile %q: %w", p.Name, err)
	}

	var stored Profile
	if uerr := json.Unmarshal(entry.Value(), &stored); uerr != nil {
		return fmt.Errorf("registry: stored profile %q is not valid JSON: %w", p.Name, uerr)
	}

	// Stored key material is authoritative. A different incoming key without a
	// rotation is refused loudly.
	if p.SigningKey != nil && stored.SigningKey != nil && p.SigningKey.Ed25519 != stored.SigningKey.Ed25519 {
		return fmt.Errorf("%w: persona %q", ErrKeyConflict, p.Name)
	}
	if stored.SigningKey != nil {
		p.SigningKey = stored.SigningKey
		p.Rotations = stored.Rotations
	}
	if stored.CreatedAt.After(p.CreatedAt) || p.CreatedAt.IsZero() {
		p.CreatedAt = stored.CreatedAt
	}

	data, merr := json.Marshal(p)
	if merr != nil {
		return fmt.Errorf("registry: encode profile: %w", merr)
	}
	if _, uerr := kv.Update(ctx, p.Name, data, entry.Revision()); uerr != nil {
		return fmt.Errorf("registry: update profile %q: %w", p.Name, uerr)
	}
	return nil
}

// Lookup reads one persona's profile. A realm without a directory, or a persona
// without an entry, is (Profile{}, false, nil) — absence is a normal state, never
// an error.
func Lookup(ctx context.Context, c *realm.Client, persona string) (Profile, bool, error) {
	kv, err := bucket(ctx, c)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return Profile{}, false, nil
		}
		return Profile{}, false, err
	}
	entry, err := kv.Get(ctx, persona)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return Profile{}, false, nil
		}
		return Profile{}, false, fmt.Errorf("registry: read profile %q: %w", persona, err)
	}
	var p Profile
	if err := json.Unmarshal(entry.Value(), &p); err != nil {
		return Profile{}, false, fmt.Errorf("registry: profile %q is not valid JSON: %w", persona, err)
	}
	return p, true, nil
}

// All reads every profile in the directory. A realm without a directory yields an
// empty slice — readers degrade, they do not fail.
func All(ctx context.Context, c *realm.Client) ([]Profile, error) {
	kv, err := bucket(ctx, c)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return nil, nil
		}
		return nil, err
	}
	lister, err := kv.ListKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("registry: list personas: %w", err)
	}

	var profiles []Profile
	for name := range lister.Keys() {
		entry, err := kv.Get(ctx, name)
		if err != nil {
			if errors.Is(err, jetstream.ErrKeyNotFound) {
				continue // deleted between list and get
			}
			return nil, fmt.Errorf("registry: read profile %q: %w", name, err)
		}
		var p Profile
		if err := json.Unmarshal(entry.Value(), &p); err != nil {
			// One corrupt entry must not hide the rest of the directory.
			continue
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func bucket(ctx context.Context, c *realm.Client) (jetstream.KeyValue, error) {
	kv, err := c.JetStream().KeyValue(ctx, BucketName)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("registry: open persona directory: %w", err)
	}
	return kv, nil
}

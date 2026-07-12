package realm

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// ProvisionOn brings a realm's artefacts to their mandated shape against the given
// JetStream handle. It is create-or-report: it creates only what is missing and
// reports the conformance of what already exists — it never modifies an existing
// artefact in place, because the history-destroying settings make in-place
// reconfiguration a one-way risk.
//
// It succeeds even when an artefact is nonconformant (the report is informational);
// it returns an error only on a connection or JetStream failure.
//
// ProvisionOn is the decoupled form used by tests and by [Client.Provision]: any
// JetStream handle works, so provisioning needs no configured context of its own.
func ProvisionOn(ctx context.Context, js jetstream.JetStream) (*ProvisionReport, error) {
	report := &ProvisionReport{}

	streamResult, err := provisionStream(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, streamResult)

	storeResult, err := provisionObjectStore(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, storeResult)

	return report, nil
}

func provisionStream(ctx context.Context, js jetstream.JetStream) (ArtefactResult, error) {
	stream, err := js.Stream(ctx, StreamName)
	switch {
	case err == nil:
		// Already present — report conformance, never mutate.
		return result(ArtefactStream, streamNonconformities(stream.CachedInfo().Config)), nil
	case errors.Is(err, jetstream.ErrStreamNotFound):
		if _, err := js.CreateStream(ctx, streamConfig()); err != nil {
			return ArtefactResult{}, fmt.Errorf("realm: create stream %q: %w", StreamName, err)
		}
		return ArtefactResult{Artefact: ArtefactStream, Outcome: OutcomeCreated}, nil
	default:
		return ArtefactResult{}, fmt.Errorf("realm: look up stream %q: %w", StreamName, err)
	}
}

func provisionObjectStore(ctx context.Context, js jetstream.JetStream) (ArtefactResult, error) {
	_, err := js.ObjectStore(ctx, ObjectBucket)
	switch {
	case err == nil:
		// Already present. The object store has no mandated settings beyond existence.
		return ArtefactResult{Artefact: ArtefactObjectStore, Outcome: OutcomeConformant}, nil
	case errors.Is(err, jetstream.ErrBucketNotFound):
		if _, err := js.CreateObjectStore(ctx, objectStoreConfig()); err != nil {
			return ArtefactResult{}, fmt.Errorf("realm: create object store %q: %w", ObjectBucket, err)
		}
		return ArtefactResult{Artefact: ArtefactObjectStore, Outcome: OutcomeCreated}, nil
	default:
		return ArtefactResult{}, fmt.Errorf("realm: look up object store %q: %w", ObjectBucket, err)
	}
}

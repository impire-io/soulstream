package realm

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ProvisionOn brings a realm's artefacts to their mandated shape against the given
// JetStream handle. It is create-or-report: it creates only what is missing and
// reports the conformance of what already exists — it never modifies an existing
// artefact in place, because the history-destroying settings make in-place
// reconfiguration a one-way risk.
//
// The single deliberate exception is the recognised legacy op-log shape (subjects
// exactly ["SOULSTREAM.>"], from before the inbox stream existed): provisioning
// converges it — narrows the capture to topic subjects, creates the inbox stream,
// migrates each persona's newest notifications into it verbatim, and purges the
// persona/service residue from the op-log. Everything else about the stream is
// preserved untouched. Any other divergent shape is still report-only.
//
// It succeeds even when an artefact is nonconformant (the report is informational);
// it returns an error only on a connection or JetStream failure.
//
// ProvisionOn is the decoupled form used by tests and by [Client.Provision]: any
// JetStream handle works, so provisioning needs no configured context of its own.
func ProvisionOn(ctx context.Context, js jetstream.JetStream) (*ProvisionReport, error) {
	report := &ProvisionReport{}

	streamResult, converged, err := provisionOpLog(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, streamResult)

	notifyResult, err := provisionNotify(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, notifyResult)

	// Migration runs only on the legacy convergence, and only once the inbox stream
	// exists to catch the republished notifications.
	if converged {
		if err := migrateInboxes(ctx, js); err != nil {
			return nil, err
		}
	}

	storeResult, err := provisionObjectStore(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, storeResult)

	personasResult, err := provisionPersonas(ctx, js)
	if err != nil {
		return nil, err
	}
	report.Results = append(report.Results, personasResult)

	return report, nil
}

// provisionOpLog creates the op-log stream, converges the recognised legacy shape
// (narrowing ONLY the subjects, preserving every other setting an operator may have
// tuned, such as MaxBytes), or reports conformance. converged is true only on the
// legacy path, signalling that inbox migration must follow.
func provisionOpLog(ctx context.Context, js jetstream.JetStream) (res ArtefactResult, converged bool, err error) {
	stream, err := js.Stream(ctx, StreamName)
	switch {
	case err == nil:
		cfg := stream.CachedInfo().Config
		if len(cfg.Subjects) == 1 && cfg.Subjects[0] == LegacyStreamSubject {
			cfg.Subjects = []string{StreamSubject}
			if _, uerr := js.UpdateStream(ctx, cfg); uerr != nil {
				return ArtefactResult{}, false, fmt.Errorf("realm: narrow legacy stream %q: %w", StreamName, uerr)
			}
			return ArtefactResult{Artefact: ArtefactStream, Outcome: OutcomeUpdated}, true, nil
		}
		// Any other existing shape — report conformance, never mutate.
		return result(ArtefactStream, streamNonconformities(cfg)), false, nil
	case errors.Is(err, jetstream.ErrStreamNotFound):
		if _, cerr := js.CreateStream(ctx, streamConfig()); cerr != nil {
			return ArtefactResult{}, false, fmt.Errorf("realm: create stream %q: %w", StreamName, cerr)
		}
		return ArtefactResult{Artefact: ArtefactStream, Outcome: OutcomeCreated}, false, nil
	default:
		return ArtefactResult{}, false, fmt.Errorf("realm: look up stream %q: %w", StreamName, err)
	}
}

// provisionNotify creates the persona-inbox stream or reports its conformance.
func provisionNotify(ctx context.Context, js jetstream.JetStream) (ArtefactResult, error) {
	stream, err := js.Stream(ctx, NotifyStreamName)
	switch {
	case err == nil:
		return result(ArtefactNotify, notifyNonconformities(stream.CachedInfo().Config)), nil
	case errors.Is(err, jetstream.ErrStreamNotFound):
		if _, cerr := js.CreateStream(ctx, notifyStreamConfig()); cerr != nil {
			return ArtefactResult{}, fmt.Errorf("realm: create inbox stream %q: %w", NotifyStreamName, cerr)
		}
		return ArtefactResult{Artefact: ArtefactNotify, Outcome: OutcomeCreated}, nil
	default:
		return ArtefactResult{}, fmt.Errorf("realm: look up inbox stream %q: %w", NotifyStreamName, err)
	}
}

// migrateInboxes moves each persona's newest ≤InboxWindow mention notifications out
// of the legacy op-log store into the inbox stream, then purges the persona and
// service residue from the op-log. Messages are republished verbatim (same subject,
// same headers, same bytes), so their signatures — bound to the subject-derived
// canonical form — keep verifying; the inbox stream's own per-subject window bounds
// whatever arrives.
func migrateInboxes(ctx context.Context, js jetstream.JetStream) error {
	stream, err := js.Stream(ctx, StreamName)
	if err != nil {
		return fmt.Errorf("realm: migrate inboxes: %w", err)
	}
	info, err := stream.Info(ctx, jetstream.WithSubjectFilter(NotifyStreamSubject))
	if err != nil {
		return fmt.Errorf("realm: enumerate legacy inboxes: %w", err)
	}

	for subject := range info.State.Subjects {
		msgs, err := lastMessagesOn(ctx, stream, subject, InboxWindow)
		if err != nil {
			return fmt.Errorf("realm: read legacy inbox %s: %w", subject, err)
		}
		for _, m := range msgs {
			if _, err := js.PublishMsg(ctx, m); err != nil {
				return fmt.Errorf("realm: migrate notification on %s: %w", m.Subject, err)
			}
		}
	}

	for _, residue := range []string{"SOULSTREAM.PERSONA.>", "SOULSTREAM.SVC.>"} {
		if err := stream.Purge(ctx, jetstream.WithPurgeSubject(residue)); err != nil {
			return fmt.Errorf("realm: purge %s residue: %w", residue, err)
		}
	}
	return nil
}

// lastMessagesOn drains one subject from a stream and returns its newest ≤limit
// messages, oldest first, as ready-to-republish NATS messages.
func lastMessagesOn(ctx context.Context, stream jetstream.Stream, subject string, limit int) ([]*nats.Msg, error) {
	// Empty guard: an ordered consumer's Next would block on a subject with no
	// messages (cannot happen for enumerated subjects, but cheap to keep safe).
	if _, err := stream.GetLastMsgForSubject(ctx, subject); err != nil {
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return nil, nil
		}
		return nil, err
	}

	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, err
	}
	it, err := cons.Messages()
	if err != nil {
		return nil, err
	}
	defer it.Stop()

	var window []*nats.Msg
	for {
		msg, err := it.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				break
			}
			return nil, err
		}
		md, err := msg.Metadata()
		if err != nil {
			return nil, err
		}
		window = append(window, &nats.Msg{Subject: msg.Subject(), Header: msg.Headers(), Data: msg.Data()})
		if len(window) > limit {
			window = window[1:]
		}
		if md.NumPending == 0 {
			break
		}
	}
	return window, nil
}

func provisionPersonas(ctx context.Context, js jetstream.JetStream) (ArtefactResult, error) {
	_, err := js.KeyValue(ctx, PersonasBucket)
	switch {
	case err == nil:
		// Already present. Existence is the mandate; history depth is advisory and
		// never mutated in place.
		return ArtefactResult{Artefact: ArtefactPersonas, Outcome: OutcomeConformant}, nil
	case errors.Is(err, jetstream.ErrBucketNotFound):
		if _, err := js.CreateKeyValue(ctx, personasConfig()); err != nil {
			return ArtefactResult{}, fmt.Errorf("realm: create persona directory %q: %w", PersonasBucket, err)
		}
		return ArtefactResult{Artefact: ArtefactPersonas, Outcome: OutcomeCreated}, nil
	default:
		return ArtefactResult{}, fmt.Errorf("realm: look up persona directory %q: %w", PersonasBucket, err)
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

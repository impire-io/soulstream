package realm

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire/soulstream/internal/natstest"
)

// newJS starts an in-process JetStream server and returns a JetStream handle plus a
// cleanup func. Shared by the provisioning tests.
func newJS(t *testing.T) (jetstream.JetStream, func()) {
	t.Helper()
	url, shutdown := natstest.StartJetStream(t)
	nc, err := nats.Connect(url)
	if err != nil {
		shutdown()
		t.Fatalf("connect: %v", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		shutdown()
		t.Fatalf("jetstream: %v", err)
	}
	return js, func() {
		nc.Close()
		shutdown()
	}
}

func outcomeByArtefact(r *ProvisionReport) map[Artefact]ArtefactResult {
	m := map[Artefact]ArtefactResult{}
	for _, res := range r.Results {
		m[res.Artefact] = res
	}
	return m
}

// US1: provisioning a clean server creates both artefacts with the mandated settings.
func TestProvisionFresh(t *testing.T) {
	ctx := context.Background()
	js, cleanup := newJS(t)
	defer cleanup()

	report, err := ProvisionOn(ctx, js)
	if err != nil {
		t.Fatalf("ProvisionOn: %v", err)
	}
	if !report.Conformant() {
		t.Errorf("report not conformant: %+v", report.Results)
	}

	got := outcomeByArtefact(report)
	if got[ArtefactStream].Outcome != OutcomeCreated {
		t.Errorf("stream outcome = %q, want created", got[ArtefactStream].Outcome)
	}
	if got[ArtefactObjectStore].Outcome != OutcomeCreated {
		t.Errorf("object store outcome = %q, want created", got[ArtefactObjectStore].Outcome)
	}

	// Read back the stream config and verify every mandated setting.
	stream, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up stream: %v", err)
	}
	cfg := stream.CachedInfo().Config
	if len(cfg.Subjects) != 1 || cfg.Subjects[0] != StreamSubject {
		t.Errorf("subjects = %v, want [%s]", cfg.Subjects, StreamSubject)
	}
	if cfg.Retention != jetstream.LimitsPolicy {
		t.Errorf("retention = %v, want Limits", cfg.Retention)
	}
	if cfg.MaxAge != 0 {
		t.Errorf("max age = %v, want 0", cfg.MaxAge)
	}
	if !cfg.AllowRollup {
		t.Error("allow rollup = false, want true")
	}
	if cfg.Duplicates < MinDuplicateWindow {
		t.Errorf("duplicate window = %v, want >= %v", cfg.Duplicates, MinDuplicateWindow)
	}
	if cfg.Storage != jetstream.FileStorage {
		t.Errorf("storage = %v, want File", cfg.Storage)
	}

	// Object store exists.
	if _, err := js.ObjectStore(ctx, ObjectBucket); err != nil {
		t.Errorf("object store lookup: %v", err)
	}
}

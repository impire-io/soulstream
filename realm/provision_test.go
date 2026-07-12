package realm

import (
	"context"
	"strings"
	"testing"
	"time"

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

// US2: a second run on a conformant realm makes zero changes and reports conformant.
func TestProvisionIdempotent(t *testing.T) {
	ctx := context.Background()
	js, cleanup := newJS(t)
	defer cleanup()

	if _, err := ProvisionOn(ctx, js); err != nil {
		t.Fatalf("first ProvisionOn: %v", err)
	}
	s1, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up stream after first run: %v", err)
	}
	createdBefore := s1.CachedInfo().Created

	report, err := ProvisionOn(ctx, js)
	if err != nil {
		t.Fatalf("second ProvisionOn: %v", err)
	}
	got := outcomeByArtefact(report)
	if got[ArtefactStream].Outcome != OutcomeConformant {
		t.Errorf("stream outcome on re-run = %q, want conformant", got[ArtefactStream].Outcome)
	}
	if got[ArtefactObjectStore].Outcome != OutcomeConformant {
		t.Errorf("object store outcome on re-run = %q, want conformant", got[ArtefactObjectStore].Outcome)
	}

	// Zero changes: the stream was not recreated (its creation time is unchanged).
	s2, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up stream after second run: %v", err)
	}
	if !s2.CachedInfo().Created.Equal(createdBefore) {
		t.Errorf("stream Created changed %v -> %v: the stream was recreated", createdBefore, s2.CachedInfo().Created)
	}
}

// US2: provisioning completes a partial realm — creates only the missing part.
func TestProvisionPartialCreatesOnlyMissing(t *testing.T) {
	ctx := context.Background()
	js, cleanup := newJS(t)
	defer cleanup()

	// Pre-create only the stream (mandated config); leave the object store missing.
	if _, err := js.CreateStream(ctx, streamConfig()); err != nil {
		t.Fatalf("pre-create stream: %v", err)
	}
	s1, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up pre-created stream: %v", err)
	}
	createdBefore := s1.CachedInfo().Created

	report, err := ProvisionOn(ctx, js)
	if err != nil {
		t.Fatalf("ProvisionOn: %v", err)
	}
	got := outcomeByArtefact(report)
	if got[ArtefactStream].Outcome != OutcomeConformant {
		t.Errorf("stream outcome = %q, want conformant (already present)", got[ArtefactStream].Outcome)
	}
	if got[ArtefactObjectStore].Outcome != OutcomeCreated {
		t.Errorf("object store outcome = %q, want created (was missing)", got[ArtefactObjectStore].Outcome)
	}

	// The pre-existing stream must be untouched, and the bucket must now exist.
	s2, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up stream after run: %v", err)
	}
	if !s2.CachedInfo().Created.Equal(createdBefore) {
		t.Error("pre-existing stream was recreated")
	}
	if _, err := js.ObjectStore(ctx, ObjectBucket); err != nil {
		t.Errorf("object store should now exist: %v", err)
	}
}

// US2: drift is reported with the specific nonconformity and never mutated in place.
func TestProvisionReportsDriftWithoutMutating(t *testing.T) {
	ctx := context.Background()
	js, cleanup := newJS(t)
	defer cleanup()

	// Pre-create a nonconformant stream: age-based expiry present, rollup disabled.
	drifted := streamConfig()
	drifted.MaxAge = 720 * time.Hour
	drifted.AllowRollup = false
	if _, err := js.CreateStream(ctx, drifted); err != nil {
		t.Fatalf("pre-create drifted stream: %v", err)
	}

	report, err := ProvisionOn(ctx, js)
	if err != nil {
		t.Fatalf("ProvisionOn (must succeed even when nonconformant): %v", err)
	}

	sr := outcomeByArtefact(report)[ArtefactStream]
	if sr.Outcome != OutcomeNonconformant {
		t.Fatalf("stream outcome = %q, want nonconformant", sr.Outcome)
	}
	joined := strings.Join(sr.Nonconformities, "; ")
	if !strings.Contains(joined, "MaxAge") {
		t.Errorf("nonconformities %v missing MaxAge drift", sr.Nonconformities)
	}
	if !strings.Contains(joined, "AllowRollup") {
		t.Errorf("nonconformities %v missing AllowRollup drift", sr.Nonconformities)
	}
	if report.Conformant() {
		t.Error("report.Conformant() = true, want false")
	}

	// The drifted stream must be unchanged — provisioning must not mutate in place.
	cfg := mustStreamConfig(ctx, t, js)
	if cfg.MaxAge != 720*time.Hour {
		t.Errorf("MaxAge was mutated to %v; provisioning must not modify in place", cfg.MaxAge)
	}
	if cfg.AllowRollup {
		t.Error("AllowRollup was mutated to true; provisioning must not modify in place")
	}
}

func mustStreamConfig(ctx context.Context, t *testing.T, js jetstream.JetStream) jetstream.StreamConfig {
	t.Helper()
	s, err := js.Stream(ctx, StreamName)
	if err != nil {
		t.Fatalf("look up stream: %v", err)
	}
	return s.CachedInfo().Config
}

package realm

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// The mandated, non-configurable shape of a realm. These are fixed by the protocol,
// not tunable knobs (see the constitution's Smallest Viable Implementation principle).
const (
	// StreamName is the op-log stream every realm holds: topic history and
	// announcements, kept forever (compacted, never aged out).
	StreamName = "SOULSTREAM"
	// StreamSubject is the subject pattern the op-log stream captures. Deliberately
	// NOT "SOULSTREAM.>": service request/reply traffic (SOULSTREAM.SVC.>) is
	// transient by construction and must be captured by no stream, and persona
	// inboxes live in their own bounded stream below.
	StreamSubject = "SOULSTREAM.TOPICS.>"
	// LegacyStreamSubject is the pre-014 capture-everything pattern. Provisioning
	// recognises exactly this shape and converges it; nothing else refers to it.
	LegacyStreamSubject = "SOULSTREAM.>"
	// NotifyStreamName is the persona-inbox stream: mention notifications only,
	// a bounded most-recent window per persona.
	NotifyStreamName = "SOULSTREAM_NOTIFY"
	// NotifyStreamSubject is the one subject pattern the inbox stream captures.
	NotifyStreamSubject = "SOULSTREAM.PERSONA.NOTIFY.>"
	// InboxWindow is how many mention notifications each persona's inbox retains —
	// the newest InboxWindow, older ones fall away. Notifications are pointers; the
	// mentioning ops stay in topic history forever.
	InboxWindow = 100
	// NotifyMaxBytes is the inbox stream's byte roof. The store is bounded by design
	// (InboxWindow small records per persona), so a hard cap is coherent — and it
	// lets provisioning succeed on account tiers that require an explicit MaxBytes.
	NotifyMaxBytes = 64 << 20
	// ObjectBucket is the single object store every realm holds.
	ObjectBucket = "soulstream-objects"
	// PersonasBucket is the single persona-directory KV bucket every realm holds:
	// persona name → profile (display metadata + published signing key).
	PersonasBucket = "soulstream-personas"
	// PersonasHistory is the KV history depth kept per persona profile — enough to
	// see recent profile edits and rotations without relying on it for trust (the
	// profile's own rotation chain is the trust record).
	PersonasHistory = 10
	// MinDuplicateWindow is the minimum acceptable duplicate-tracking window; the
	// mandated streams use exactly this value.
	MinDuplicateWindow = 2 * time.Minute
)

// Budgets are optional per-artefact byte roofs applied when provisioning CREATES
// an artefact. A zero field means "no budget" — that artefact is created unlimited,
// exactly as when no Budgets value is given at all — with one exception: the inbox
// stream is bounded by design, so a zero Notify keeps the mandated [NotifyMaxBytes].
// Budgets never modify an existing artefact; provisioning stays create-or-report.
//
// Limit-enforced account tiers (NGS R1 is the known case) refuse to create any
// stream without an explicit byte roof; budgets exist so such accounts provision
// out of the box. See [DefaultBudgets].
type Budgets struct {
	OpLog    int64
	Notify   int64
	Personas int64
	Objects  int64
}

// DefaultBudgets returns the shapes proven on the known limit-enforced free tier:
// 1 GiB op-log, the mandated 64 MiB inbox, 64 MiB persona directory, 512 MiB
// attachment store. They exist to make that tier work with no per-artefact tuning,
// not to model anyone's real capacity needs.
func DefaultBudgets() Budgets {
	return Budgets{
		OpLog:    1 << 30,
		Notify:   NotifyMaxBytes,
		Personas: 64 << 20,
		Objects:  512 << 20,
	}
}

// validate rejects impossible budgets before any server contact. Zero fields are
// legal (no budget); a negative roof can only be a mistake.
func (b Budgets) validate() error {
	for _, f := range []struct {
		artefact string
		roof     int64
	}{
		{"op-log stream", b.OpLog},
		{"inbox stream", b.Notify},
		{"persona directory", b.Personas},
		{"attachment store", b.Objects},
	} {
		if f.roof < 0 {
			return fmt.Errorf("realm: budget for the %s must be positive, got %d", f.artefact, f.roof)
		}
	}
	return nil
}

// streamConfig is the mandated op-log stream configuration: limits retention with no
// age-based expiry, subject rollup permitted, a duplicate window, and disk storage.
// maxBytes is the optional creation-time byte roof (0 = unlimited).
func streamConfig(maxBytes int64) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:        StreamName,
		Subjects:    []string{StreamSubject},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      0, // no age-based expiry — history is compacted, never aged out
		AllowRollup: true,
		Duplicates:  MinDuplicateWindow,
		Storage:     jetstream.FileStorage,
		MaxBytes:    maxBytes,
	}
}

// notifyStreamConfig is the mandated inbox stream configuration: the newest
// InboxWindow notifications per persona (per subject), oldest discarded, under a
// byte roof. No age expiry — the window is count-based, not time-based. The store
// is bounded by design, so a zero maxBytes keeps the mandated roof, never unlimited.
func notifyStreamConfig(maxBytes int64) jetstream.StreamConfig {
	if maxBytes == 0 {
		maxBytes = NotifyMaxBytes
	}
	return jetstream.StreamConfig{
		Name:              NotifyStreamName,
		Subjects:          []string{NotifyStreamSubject},
		Retention:         jetstream.LimitsPolicy,
		MaxAge:            0,
		MaxMsgsPerSubject: InboxWindow,
		MaxBytes:          maxBytes,
		Discard:           jetstream.DiscardOld,
		Duplicates:        MinDuplicateWindow,
		Storage:           jetstream.FileStorage,
	}
}

// objectStoreConfig is the mandated object-store configuration. maxBytes is the
// optional creation-time byte roof (0 = unlimited).
func objectStoreConfig(maxBytes int64) jetstream.ObjectStoreConfig {
	return jetstream.ObjectStoreConfig{
		Bucket:   ObjectBucket,
		MaxBytes: maxBytes,
	}
}

// personasConfig is the mandated persona-directory KV configuration. maxBytes is
// the optional creation-time byte roof (0 = unlimited).
func personasConfig(maxBytes int64) jetstream.KeyValueConfig {
	return jetstream.KeyValueConfig{
		Bucket:   PersonasBucket,
		History:  PersonasHistory,
		MaxBytes: maxBytes,
	}
}

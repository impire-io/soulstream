package realm

import (
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

// streamConfig is the mandated op-log stream configuration: limits retention with no
// age-based expiry, subject rollup permitted, a duplicate window, and disk storage.
func streamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:        StreamName,
		Subjects:    []string{StreamSubject},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      0, // no age-based expiry — history is compacted, never aged out
		AllowRollup: true,
		Duplicates:  MinDuplicateWindow,
		Storage:     jetstream.FileStorage,
	}
}

// notifyStreamConfig is the mandated inbox stream configuration: the newest
// InboxWindow notifications per persona (per subject), oldest discarded, under a
// fixed byte roof. No age expiry — the window is count-based, not time-based.
func notifyStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:              NotifyStreamName,
		Subjects:          []string{NotifyStreamSubject},
		Retention:         jetstream.LimitsPolicy,
		MaxAge:            0,
		MaxMsgsPerSubject: InboxWindow,
		MaxBytes:          NotifyMaxBytes,
		Discard:           jetstream.DiscardOld,
		Duplicates:        MinDuplicateWindow,
		Storage:           jetstream.FileStorage,
	}
}

// objectStoreConfig is the mandated object-store configuration.
func objectStoreConfig() jetstream.ObjectStoreConfig {
	return jetstream.ObjectStoreConfig{
		Bucket: ObjectBucket,
	}
}

// personasConfig is the mandated persona-directory KV configuration.
func personasConfig() jetstream.KeyValueConfig {
	return jetstream.KeyValueConfig{
		Bucket:  PersonasBucket,
		History: PersonasHistory,
	}
}

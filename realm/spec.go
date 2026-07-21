package realm

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// The mandated, non-configurable shape of a realm. These are fixed by the protocol,
// not tunable knobs (see the constitution's Smallest Viable Implementation principle).
const (
	// StreamName is the single op-log stream every realm holds.
	StreamName = "SOULSTREAM"
	// StreamSubject is the one subject pattern the stream captures.
	StreamSubject = "SOULSTREAM.>"
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
	// mandated stream uses exactly this value.
	MinDuplicateWindow = 2 * time.Minute
)

// streamConfig is the mandated stream configuration: limits retention with no
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

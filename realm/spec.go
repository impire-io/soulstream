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

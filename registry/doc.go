// Package registry is the persona directory: the realm-wide, watchable store of
// persona profiles — display metadata and, above all, published signing keys.
//
// The directory is the minimal slice of the registry extension needed for key
// distribution. It is a plain JetStream KV bucket (soulstream-personas); first
// publish is a KV Create, every later write an optimistic Update, so two clients can
// never silently overwrite each other's key material.
//
// Trust is first-use pinning of the whole key chain: a profile carries its rotation
// proofs (each new key endorsed by the previous one), Chain validates them offline,
// and BuildKeyring compares published chains against a reader's pins — a published
// chain that does not extend the pinned one marks the persona distrusted. The chain
// logic is pure (no NATS) so it unit-tests with no server; only the KV calls touch
// the connection.
//
// Nothing anywhere may branch on a profile's fields except for presentation: kind,
// display name, and description are cosmetic and audit metadata, never permission.
package registry

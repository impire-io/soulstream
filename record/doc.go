// Package record defines the Soulstream operation record: the unit of everything
// that travels on the wire.
//
// A [Record] carries its metadata in NATS message headers and keeps its payload
// as pure data (no metadata wrapper). The same record has two forms:
//
//   - the wire form — headers plus a payload, produced by [Record.Build] and read
//     back by [Parse]; the two are exact inverses.
//   - the canonical form — a deterministic RFC 8785 (JCS) byte sequence produced by
//     [Record.Canonical], bound to a realm and topic, used as the input a future
//     signature will sign.
//
// This package imports nothing from NATS: it works with plain header maps and byte
// payloads so it can be unit-tested with no server at all.
package record

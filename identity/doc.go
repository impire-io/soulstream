// Package identity holds the two identity primitives Soulstream enforces at the
// library edges: persona-name validation and honest attribution.
//
// A persona name (also a realm name, also a topic-id slug) is a lowercase,
// transport-token-safe slug; [ValidName] and [CheckName] validate the grammar.
//
// Attribution is enforced on the write side — a persona-bound client stamps only its
// own name ([EnforceAuthor]) — and checked on the read side ([VerifyAuthor]),
// structurally always and against a trusted [Resolver] when one is supplied.
//
// This package imports nothing from NATS.
package identity

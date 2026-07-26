# Does the sealed-topics design survive the substrate as shipped?

**State:** active
**Started:** 2026-07-27

## Abstract

Sealed topics are day-2 #9 and the single biggest remaining build item. The
design ([../../02-DESIGN/extensions/sealed-topics.md](../../02-DESIGN/extensions/sealed-topics.md))
already chose its shape — per-epoch symmetric topic keys, member-wrapped, MLS
named as the upgrade path — but it predates most of what has since shipped:
canonical-record signing (006), leaderless rollup with the race guard (007),
operator attestation (014), and above all the memory convention with
self-authenticating exhibits (015). This topic validates the design against
the substrate as it exists before the build gate opens. A decisive answer
either yields speckit-ready design amendments or names prerequisite work
(an MLS prerequisite, a canonical-form change) while it is still cheap to know.

## The question

Can the sealed-topics extension, as designed, be built on the shipped
substrate — `sealed.op` signing and canonical form, exhibit-grade recovery,
leaderless sealed rollup, and key distribution over the existing registry —
without weakening any 014/015 invariant, and does the epoch-key scheme
(not MLS) still clear the design's own threat model at dogfood scale?

## Pre-registered bars

- **Bar 1 — the envelope signs and exhibits verify.** Criterion: a
  `sealed.op` whose payload is ciphertext has a well-defined canonical
  record; the author's signature verifies with zero access to epoch keys;
  and a captured exhibit of that op gets a `verified` verdict offline
  (pins-only). Protocol: throwaway Go prototype in the session scratchpad
  using only `record` + `identity` public API — construct, sign, capture as
  `record.Exhibit`, verify with no key material present. Pass = `verified`
  both on the live path and offline. If the canonical form cannot carry a
  ciphertext payload without breaking 014's same-bytes invariant, the bar
  FAILS and records the smallest encoding amendment that would pass.
- **Bar 2 — key distribution fits the registry as shipped.** Criterion:
  every distribution need (encryption-key publication alongside the Ed25519
  signing key, rotation, epoch wrap targets, prior-epoch handoff to
  joiners) maps onto an existing registry/op-log surface. Protocol: written
  row-by-row mapping against
  [../../02-DESIGN/extensions/registry.md](../../02-DESIGN/extensions/registry.md)
  and the `registry` package's public API, each row citing the surface it
  lands on or naming the gap. Pass = zero rows require a new server
  component or stream; gaps may only be additive vocabulary or profile
  fields.
- **Bar 3 — sealed rollup stays leaderless.** Criterion: the 007 rollup
  path, race guard included, works when baseline state is ciphertext: any
  member can compact, non-members cannot produce a valid sealed baseline,
  and an interior sealed op destroyed by that rollup comes back from a
  keeper as a VERIFYING exhibit. Protocol: written trace through
  `topic`'s rollup path and the archivist flow; a prototype run only if the
  trace is contested. Pass = no step requires plaintext outside a member
  and no step requires a distinguished leader.
- **Bar 4 — the MLS deferral holds.** Criterion: at dogfood scale
  (≤ ~10 members, member devices trusted, metadata visibility accepted per
  the threat model), no threat-model row requires forward secrecy or
  post-compromise security within an epoch. Protocol: row-by-row verdict
  table over the design doc's threat model, each row marked
  covered-by-epoch-keys / requires-MLS / out-of-scope-by-design. Pass =
  zero requires-MLS rows at that scale; a single such row moves MLS from
  upgrade path to prerequisite and the topic's verdict says so.

## Reversal condition

Two readings would change this topic's direction, and either may arrive
during the concurrent dogfood run: (1) the dogfood chafe log ends its two
weeks with zero entries of the form "this felt wrong to write into a
plaintext realm" — then sealing's *priority* reverses and it stays behind
the browser client regardless of these bars; (2) Bar 1 shows no ciphertext
encoding can preserve the same-bytes invariant — then the *direction*
reverses from payload encryption inside the existing record toward an
envelope-level redesign, and this topic graduates abandoned with that
finding.

## Verdict


# Episode 0007 — DX hardening: the seam's two sharp edges, and the cycle guard (2026-07-29)

Episode 0006 shipped the signer seam with two honest findings attached; the
operator's call, same day: fix both for real before tagging, because "it
will be a problem" for developers — and pin down the dependency direction
between soulstream and SoulIdentity while the seam is young. All three
landed before the release, inside 017's unreleased window, where breaking a
surface costs nothing.

**Typed-nil, from documented discipline to refused input.** Episode 0006
recorded the hazard as assignment discipline plus a warning comment
[judgment], after it SIGSEGV'd six of our own helpers [measured]. That
evidence was re-read as R6's reversal condition arriving early — six
in-house hits on day one predicts consumer hits — so the rejected
alternative got un-rejected in its fail-fast form: `Connect`/`NewClient` now
refuse a typed-nil `Config.Signer` with an error naming the fix ("a missing
key must leave the field unset"), before any server contact. The check is
one contained reflection guard at validation, safe for non-nillable
implementations (a value-type signer passes) [measured]. What was NOT
adopted: silently normalising typed-nil to unsigned — the natural intent of
`Signer: maybeNilKey` is "unsigned if absent", but a nil *custodian
adapter* normalised away would publish unsigned exactly when the caller
believed they had configured signing, the downgrade FR-004 exists to
prevent [mechanism-argument, argued adversarially].

**The `-1` sentinel dies; callbacks carry the error.** 0006's second
finding (the memory query path conflating could-not-sign with
nothing-to-say) was symptom, not cause: the cause was sentinel-based
observability. `RespondDiscovery`/`RespondDiscoveryWith`'s `onServed` and
`MemoryWitness.OnServed` now have shape `(query/kind, sent, err)` — sent is
what actually went out, err says why something didn't (unreadable,
reply-less, stale, unsignable, unpublishable), a plain no-match is
`(0, nil)`. A host can finally tell "vault down, page me" from ambient
noise, and the partial case (some answers sent, some unsignable) is visible
for the first time instead of swallowed [measured: the FR-012 suite now
asserts the custodian's cause verbatim through both responders]. This
breaks the v0.5.0 witness/responder API deliberately — pre-1.0, 017
untagged, zero external consumers except our own archivist, which needs a
two-line callback change on its next dependency bump. The CLI `respond`
command's own code was the closing argument: it carried a comment
apologising for the sentinel ("skipped silently"); it now prints the reason.

**The cycle guard.** Verified: soulidentity's module graph contains zero
soulstream imports [measured, go.mod read 2026-07-29]. The rule keeping it
that way is now written where implementers will meet it (the `Signer` doc
comment, the 017 contract, SoulIdentity's M2 milestone): Go satisfies the
seam *structurally*, so a custodian client never needs to import soulstream
— the adapter lives in the consumer binary, above both repos
[mechanism-argument]. A module cycle would be legal Go and a versioning
trap; the structural interface makes it permanently unnecessary.

Shipped as **v0.6.0** with plugin/marketplace 0.6.0 — 017 plus this
hardening in one tag.

Reversal condition: for the typed-nil refusal — a legitimate implementation
pattern that *requires* a nil-valued yet callable signer (observable: a
consumer issue where the validation error blocks a working nil-receiver
implementation) would demand an opt-out, as a new decision. For the
callback shape — none; it records a completed build. For the cycle guard —
a capability that genuinely cannot live in a consumer binary and needs one
core repo to import the other (observable: a concrete feature spec blocked
on the rule) reopens it as a design decision with the versioning cost
argued at full strength.

Trail: `specs/017-signer-seam/contracts/library.md` + `quickstart.md`
(updated to the shipped shapes), `identity/signer.go`, `realm/connect.go`,
`topic/discover.go`, `topic/memory.go`, `curator/curator.go`,
`internal/cli/discover_cmd.go`; SoulIdentity
`hq/03-IMPLEMENTATION/ROADMAP.md` M2 (commit `4c55693` there); episode
[0006](0006-the-signer-seam.md). Commits: the `017-dx-polish` branch
series into the v0.6.0 release.

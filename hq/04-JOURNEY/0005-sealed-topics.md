# Episode 0005 — Sealed topics survive the substrate: four bars, one encoding amendment (2026-07-27)

Sealed topics are day-2 #9 and the biggest remaining build item, but their
design predates most of what has shipped since: canonical-record signing
(006), leaderless rollup with the race guard (007), operator attestation
(014), and the memory convention with self-authenticating exhibits (015).
The research topic asked one question before the build gate opens: does the
design — per-epoch symmetric topic keys, member-wrapped, MLS named as the
upgrade path — survive the substrate as shipped, without weakening any
014/015 invariant? Four pre-registered bars answered it in one day, and the
answer is yes, with amendments the bars forced into the open.

**Bar 1 (the envelope signs and exhibits verify) FAILED for the design's
literal shape and recorded the smallest passing amendment [measured].** A
throwaway prototype against the public `record`/`identity` API showed the
canonical form refuses non-JSON payloads outright — a raw-binary ciphertext
`sealed.op`, the design doc's literal sketch, has no signing input and can
never carry a verifying author signature. The one-field JSON wrapper
`{"ct":"<base64>"}` passes everything: signs on the live path, verifies
with zero access to epoch keys, and its captured exhibit grades `verified`
offline with a pins-only keyring. The same run caught a subtler hole: the
sketched `Soulstream-Epoch`/`-Nonce` headers are not covered by the author
signature — rewriting epoch 4→9 on the wire leaves the signature verifying,
while the tamper controls (timestamp, ciphertext, cross-topic splice) all
flip to `failed`. Epoch integrity rested on AEAD associated data alone,
detectable by members only; the amendment moves epoch and nonce inside the
signed payload, beside `ct`, where graders, archivists, and curators can
check them too.

**Bar 2 (key distribution fits the registry as shipped) PASSED
[measured].** A row-by-row mapping landed all seven distribution needs —
sealing-key publication, rotation, epoch wrap targets, prior-epoch handoff,
epoch-1 announcement material, sealed mention bodies, pinning — on shipped
registry/op-log surfaces, with zero new server components or streams. The
gaps are exactly the sanctioned classes: additive profile fields
(`sealing_key`, plus sealing rotations) and additive vocabulary. Two
consequences worth the record: X25519 keys cannot sign, so registry.md's
"new key signed by the old one" rotation rule cannot apply to sealing
keys — they are endorsed by the persona's Ed25519 signing chain instead,
inheriting its TOFU trust with no separate pin lineage [judgment]; and the
registry's strict profile decode means the `sealing_key` field must ship in
the library before any persona publishes one, or lookups on that persona
hard-fail for older readers [measured].

**Bar 3 (sealed rollup stays leaderless) PASSED on an uncontested written
trace — the prototype clause was never invoked [measured].** Every
NATS-touching step of the 007 rollup path is content-blind: `record.Parse`
validates headers only, the fold's DAG bookkeeping computes the frontier
from cleartext headers before the type switch runs, the race guard compares
sequence numbers server-side (first writer wins, loser's log untouched),
and the purge is per-subject. Folding content is the single plaintext step
and it runs client-side in a member — exactly where the design puts it.
The keeper→exhibit chain never interprets a payload, so a rollup-destroyed
interior sealed op returns from a keeper as a `verified`,
`fact-with-provenance` exhibit (Bar 1's measurement carries the verdict).
The trace's key finding: the per-subject purge destroys interior
`sealed.epoch` ops — epoch-1 material survives only because it travels in
the announcement on the INFO subject, which rollup never touches. The
amendment: the sealed baseline re-carries the current epoch's wrapped-key
table (any member holds the full map, so leaderlessness is preserved), and
carries `baked` state inside the ciphertext — the shipped baked shape holds
contribution bodies in cleartext, which a sealed baseline must never do
[judgment].

**Bar 4 (the MLS deferral holds) PASSED [judgment over the design's own
threat-model text].** Fifteen verdict rows, zero requires-MLS at dogfood
scale (≤ ~10 members, member devices trusted, metadata visibility
accepted). Stated honestly: the forward-secrecy and post-compromise rows
pass *because of* the trusted-device premise, not because epoch keys
provide those properties — and the flip conditions are now on record:
membership scale or churn meaningfully beyond ~10, or member devices no
longer assumed trusted, moves MLS from upgrade path to prerequisite.

What was refuted: the design's raw-binary payload encoding (Bar 1) — the
one reversal-condition trigger that could have redirected the whole design
toward an envelope-level redesign instead resolved to a one-field JSON
wrapper, so the direction held. What it taught: the substrate's separation
of cleartext coordination metadata (headers, frontier, sequence guards)
from payload content is exactly the seam sealing needs — every mechanism
that had to stay blind already was; the amendments are all at the
vocabulary and profile-field level, none structural.

The design doc
([`../02-DESIGN/extensions/sealed-topics.md`](../02-DESIGN/extensions/sealed-topics.md))
now carries the amendments and is speckit-ready. The build's *priority*
remains gated: the dogfood chafe log runs to 2026-08-10, and a zero-entry
"this felt wrong to write into a plaintext realm" outcome keeps sealing
behind the browser client regardless of these verdicts.

Reversal condition: the Bar 4 flip conditions are the standing ones —
membership scale/churn meaningfully beyond ~10 members or sealed topics
including personas on untrusted hosts moves MLS from upgrade path to
prerequisite and reopens the key-scheme decision; and a dogfood chafe log
ending 2026-08-10 with zero sealing-shaped entries reverses the build's
priority (not the design's validity).

Trail: research topic `hq/01-RESEARCH/sealed-topics/` (removed at
graduation; full bar evidence in its JOURNEY.md, in git history),
[`../02-DESIGN/extensions/sealed-topics.md`](../02-DESIGN/extensions/sealed-topics.md)
(amended),
[`../02-DESIGN/extensions/registry.md`](../02-DESIGN/extensions/registry.md)
(sealing-key endorsement rule); commits 64040af (Bar 1), c426595 (Bar 2),
7eb7797 (Bar 3), 18d3a6a (Bar 4), plus the graduation commit.

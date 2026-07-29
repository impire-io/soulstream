# Episode 0006 — The signer seam: signing learns to be delegated (2026-07-29)

Feature `017-signer-seam`, the half of SoulIdentity's M2 ("consumers wire
in") that lives in this repository. The question: can record signing be
decoupled from the concrete local Ed25519 key so a consumer may delegate it
to an external custodian (SoulIdentity's `sign.record` NATS service) without
soulstream depending on that custodian? The answer is a two-method interface,
`identity.Signer { PublicKey() string; Sign(canonical []byte) (string, error) }`,
with the local key its first implementation — `(*SigningKey).Sign` itself
became fallible (error always nil locally), so there is exactly one way to
sign and the compiler walked us to every call site [mechanism-argument].
Delegation is transparent [measured]: a record signed through a delegate
double is byte-identical to local signing over the same canonical bytes
(Ed25519 determinism), and every read surface — wire verify, materialise,
follow, inbox, exhibit capture with offline grading — reports it verified
with the same verdicts. 361 tests, 0 skipped, lint 0; `go.mod` untouched;
`identity` still imports no NATS.

The load-bearing failure rule: a configured signer that cannot sign **fails
the operation** — no unsigned fallback, and an *empty* signature counts as
failure because the canonical form spells "unsigned" as an omitted `sig`
[mechanism-argument, from 006]. Proven with failure injection [measured]:
publishes error with the cause in the chain and the op-log message count
unchanged. Responders (discovery, memory witnesses) go **silent** instead —
silence is already the protocol's word for "no answer" — while the host
observes `served(-1)`. One real gap surfaced there [measured]: the memory
*query* path reported `served("query", 0)` when every answer failed to
build, conflating could-not-sign with nothing-to-say; fixed minimally (had
material + built nothing ⇒ `-1`).

Two beliefs upgraded during the build. First, the typed-nil hazard
(`realm.Config.Signer` is now interface-typed) was written down as a
documented discipline [judgment] and promptly fired in six of our own test
helpers on the first run — SIGSEGV inside `Sign` — turning it into a
[measured] fact; all wiring now assigns a loaded key only when non-nil.
Second, the boundary rule that decided every surface: capability-only
surfaces (publish chokepoint, attestation tokens, rotation proofs) take the
interface; seed-custody surfaces (keystore, key generation) keep the
concrete type, so FR-008 — a delegated signer never possesses a seed — is
enforced by the type system itself [mechanism-argument].

What it opened: SoulIdentity's client can now implement `identity.Signer`
by importing only `identity` (its `SignRecord(persona, canonical) (string,
error)` already has the seam's shape — deliberately: the interface carries
no context parameter, deadlines are the implementation's duty), and the
remote MCP node (018-ish) gets custody-free per-user signing. The live
cross-service proof — a record signed through the running SoulIdentity
service verifying in a real realm — is SoulIdentity M2's gate, exercised
there, not here.

Reversal condition: a real consumer demonstrating that implementation-owned
timeouts are insufficient — observable as a remote-node incident where a
hung custodian blocks publishes beyond its own configured deadline, recorded
as an issue — adds caller-side deadline propagation (a context-carrying
variant) as a new decision; and a second typed-nil incident *outside* our
own tests (a consumer bug report with the SIGSEGV signature) reopens R6's
rejected alternatives (reflection at validate time).

Trail: `specs/017-signer-seam/` (spec + Clarifications 2026-07-29, plan,
research R1–R7 — R6 carries the measured upgrade — data-model, contracts,
quickstart); `docs/signing.md` ("someone else can hold your stamp", "when
the vault doesn't answer"), `docs/operators.md`;
`hq/03-IMPLEMENTATION/ROADMAP.md`; SoulIdentity's
`hq/03-IMPLEMENTATION/ROADMAP.md` M2. Commits: the `017-signer-seam` branch
series (spec `3053e68` → clarify `add8320` → plan `7b74814` → tasks
`ad56904`/`29edfd8` → seam `3d2d154` → US1 `631d455` → US2 `a66bb94` → US3
`4b2e0f4`/`0691b26` → landing).

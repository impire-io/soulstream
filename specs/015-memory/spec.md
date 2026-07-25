# Feature Specification: Memory Convention & Exhibits

**Feature Branch**: `015-memory`
**Created**: 2026-07-25
**Status**: Draft
**Input**: User description: "Memory convention: collective search over the realm as
scatter/gather testimony with citations (Day-2 #8, design: hq/02-DESIGN/extensions/memory.md).
This feature ships the CONVENTION and the LIBRARY SURFACE only — the first archivist persona
is explicitly OUT OF SCOPE for this repo and will live in a separate repository under the
impire-io org, built against the public library surfaces this feature provides."

## Why now

The realm compacts: rollup physically removes op tails, and that is by design — the stream is
a transport, not an archive. Every rollup since `007` has been quietly closing a one-way door
the roadmap named explicitly: *retention is not retrofittable*. A witness that starts keeping
history today has op-granularity memory from today; everything already compacted is baseline-
granularity forever. The realm now holds real history (real profiles, real topics on a live
deployment), so every week without a memory convention widens a permanent blind spot. This
feature ships the convention — how a realm is asked, how answers cite, how evidence stays
verifiable after the substrate forgets — and the public surface an archivist plugs into. The
archivist itself (the persona that keeps the full archive) is deliberately a separate product
in a separate repository under the impire-io organisation: the convention must be provably
servable by an outsider built only on public surfaces, and the cleanest proof is that its
first real consumer *is* an outsider.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ask the realm, get graded testimony (Priority: P1)

A participant asks the realm a question in plain language — "what did we decide about the Q2
VAT reminder cadence?" — optionally scoped to topic name patterns and a time window, with a
deadline. Any persona that runs a memory service and believes it can help answers before the
deadline; personas with nothing to say stay silent. The asker receives the answers that
arrived, merged and attributed: each answer names its witness, and every claim is graded by
verifiability — a citation that resolves in the realm's current state is **fact**; a citation
backed by a verifiable exhibit is **fact with provenance**; an unverifiable statement is
**testimony**; an uncited "I remember that…" is **gossip**, useful for leads, never decisions.
Witnesses never coordinate and never see each other's answers; weighing conflicting testimony
is deliberately the asker's burden. A realm with no witnesses yields a clean empty result at
the deadline — silence is an honest answer.

**Why this priority**: This is the convention itself — the question-and-testimony loop is what
the realm commits to. Everything else (exhibits, the witness surface) exists to make these
answers verifiable and servable.

**Independent Test**: With a test witness serving canned answers, publish a query and confirm
the asker receives merged, attributed, graded answers by the deadline; with no witness,
confirm the same query completes cleanly at the deadline with zero answers.

**Acceptance Scenarios**:

1. **Given** a realm with at least one answering witness, **When** a participant queries with
   a question, optional scope, and a deadline, **Then** all answers arriving before the
   deadline are returned merged, each attributed to its witness, each citation carrying
   exactly one verifiability grade.
2. **Given** a realm with no witnesses, **When** a participant queries, **Then** the query
   completes at the deadline with an empty, non-error result.
3. **Given** an answer citing an op that still resolves in its topic's current state (live op
   or baked into the current baseline), **When** the asker grades it, **Then** the citation
   grades as fact, verified by actually resolving it — never by trusting the witness.
4. **Given** an answer whose citation resolves nowhere and carries no exhibit, **When** the
   asker grades it, **Then** the citation is marked unverifiable — presented distinctly and
   cautiously (it may be compacted history or fabrication; the two are indistinguishable),
   never presented as fact.
5. **Given** two witnesses giving conflicting answers, **When** the asker reads the result,
   **Then** both answers appear, attributed and graded — the convention never arbitrates
   truth.
6. **Given** a witness that replies after the asker's deadline, **When** the asker completes,
   **Then** the late answer is simply absent from the result.

---

### User Story 2 - Exhibits: evidence that outlives the stream (Priority: P2)

A participant exports any operation the realm currently holds as an **exhibit**: a portable,
self-contained document carrying the operation's canonical record and its author's signature.
Anyone, anywhere — including someone with no realm connectivity at all — verifies the exhibit
against the author's known key material and learns, from the document alone, that this author
said this thing in this realm and topic. Tampering with any part of a signed exhibit makes
verification fail. An exhibit of an unsigned operation is still exportable and readable, but
verification honestly reports it unsigned: it is only as trustworthy as whoever kept it.

**Why this priority**: The exhibit is the load-bearing epistemic piece — it is what makes a
kept copy *evidence* rather than hearsay, no matter who kept it or how it travelled. It is
independently valuable (share a signed decision with an outsider today) and is the unit the
witness surface serves.

**Independent Test**: Export an exhibit of a signed op, move the bytes out-of-band (a file),
verify it in a context with no realm connection and confirm it verifies; flip one byte
anywhere and confirm verification fails.

**Acceptance Scenarios**:

1. **Given** any operation currently in the realm (live or baked context notwithstanding — a
   live op), **When** a participant exports it, **Then** the result is a single portable
   document containing everything needed to read and verify the operation.
2. **Given** a signed exhibit and the author's known key material, **When** it is verified
   without any realm connectivity, **Then** the verdict is verified, and the verdict names
   the author, realm, and topic the signature binds.
3. **Given** a signed exhibit altered in any way (content, attribution, or signature),
   **When** it is verified, **Then** the verdict is failed — never verified, never silently
   unsigned.
4. **Given** an exhibit whose author's key material is not known to the verifier, **When** it
   is verified, **Then** the verdict is honestly "unknown key" — neither verified nor failed.
5. **Given** an exhibit of an operation that was never signed, **When** it is verified,
   **Then** the verdict is unsigned — the content is readable, graded as testimony.

---

### User Story 3 - Anyone can serve memory; the archivist plugs in from outside (Priority: P3)

A persona that keeps memory — any store, any shape: a directory of files, an index, a full
op archive — serves it to the realm through a public witness surface: it receives queries and
answers the ones it can help with, and it receives fetch requests for specific operations and
answers with exhibits from its keeping. Each witness declares the moment its memory starts
(`coverage_from`), because retention is not retrofittable and the blind spot must be visible.
When an operation has been compacted out of the stream, a fetch request is the only way back:
any witness holding a signed exhibit of it restores it as verifiable evidence — provenance
belongs to whoever bothered to keep bytes, not to a privileged archive role. The first real
witness — an archivist persona keeping the full uncompacted archive — is a separate product
in its own repository under the impire-io organisation, built exclusively against this public
surface; this feature must make that outside build possible without any private hooks.

**Why this priority**: The witness surface is the contract the external archivist (and every
future memory-keeping persona) builds against. It depends on the vocabulary (Story 1) and the
exhibit (Story 2) but is independently testable with a test witness playing the archivist
role.

**Independent Test**: Implement a minimal witness using only public surfaces (as tests will,
standing in for the external archivist): serve one canned answer and one kept exhibit;
confirm an asker's query returns the answer graded, and that after the cited op is compacted
away by rollup, a fetch through the witness restores it as a verifying exhibit.

**Acceptance Scenarios**:

1. **Given** a persona with any private store, **When** it attaches to the realm through the
   public witness surface, **Then** it receives queries and fetch requests and may answer,
   with no library-private access required.
2. **Given** a witness with nothing relevant to a query, **When** the query arrives, **Then**
   the witness stays silent and the asker experiences only the answers that did arrive.
3. **Given** an operation removed from the stream by rollup, **When** an asker fetches it and
   a witness holds its signed exhibit, **Then** the asker receives the exhibit and it
   verifies — compacted history is recoverable as evidence, from any keeper.
4. **Given** multiple witnesses answering a fetch, **When** exhibits arrive, **Then** a
   verifying exhibit is preferred over an unsigned one, and an unsigned exhibit is kept only
   as testimony when nothing verifiable arrived by the deadline.
5. **Given** a witness's answer, **When** the asker reads it, **Then** the witness's declared
   coverage start is visible, so the asker knows the witness's blind spot.
6. **Given** the realm's permanent store, **When** any amount of memory traffic (queries,
   answers, fetches) is exchanged, **Then** the permanent store retains none of it —
   memory exchange is transient by the same hygiene rule as discovery.

---

### Edge Cases

- A malformed query (unreadable, no deadline, or a deadline already past) arrives at a
  witness → silently ignored, like malformed discovery requests; the asker of a malformed
  query gets a loud local error before anything is published.
- An answer arrives whose witness signature fails verification → discarded as malformed
  (evidence of tampering is not testimony); an answer from a witness that simply signs
  nothing is kept and its signature status surfaced (signing is optional per persona).
- The asker's own persona also runs a witness → allowed; it answers its own query like any
  other witness.
- A witness sends multiple answers to one query → all kept, all attributed to it; the asker
  judges.
- A citation names a topic that exists but an op-id that only survives *baked* into the
  current baseline → resolves; grades as fact (state-level survival is the design's promise).
- A citation names a topic the realm has never seen → unverifiable citation (caution), same
  as an op that resolves nowhere.
- An exhibit is exported for an op whose signature already fails against its author's chain →
  the export succeeds and carries what the stream carried; verification of the exhibit
  reports failed — exporting never launders evidence.
- An exhibit's author rotated keys after signing → verification accepts any key in the
  author's validated chain (the standing chain rule); a verifier who knows none of the
  chain's keys reports unknown-key.
- Sealed topics → out of scope entirely: no sealed content in queries, answers, or exhibits;
  that convention lands with sealed topics.
- Two valid signed exhibits arrive for the same op → signatures cover canonical content, so
  both carry the same record; either serves.

## Requirements *(mandatory)*

### Functional Requirements

**Vocabulary & transport**

- **FR-001**: The realm MUST commit to a single well-known memory service channel on which
  queries and fetches are asked and answered request/reply-style; memory traffic MUST be
  transient — the realm's permanent store retains none of it (the discovery hygiene rule).
- **FR-002**: A memory query MUST carry a free-text question, an answer deadline, and MAY
  carry a scope of topic-name patterns and an "after" moment; scope is a relevance hint for
  witnesses, not an enforced filter.
- **FR-003**: Any persona MAY answer any query or fetch; non-answers are silent; a realm with
  zero witnesses MUST be indistinguishable from a realm where no witness had anything to say:
  the asker completes cleanly at its deadline with the answers that arrived.
- **FR-004**: A memory answer MUST carry the witness's prose answer, zero or more citations
  (each naming a topic and an operation id), and MAY carry the witness's coverage start
  (`coverage_from`); answers and queries follow the realm's standing signing convention for
  service traffic — an answer whose signature fails verification MUST be discarded, an
  unsigned answer MUST be kept with its signature status visible.
- **FR-005**: A memory fetch MUST name exactly one operation (topic + op id); a fetch reply
  carries an exhibit of that operation; the asker MUST prefer the first verifying exhibit and
  MUST fall back to an unsigned exhibit only as testimony when no verifying exhibit arrived
  by the deadline.

**Exhibits**

- **FR-006**: An exhibit MUST be a single self-contained portable document from which the
  operation's content, authorship, realm/topic binding, and signature (when present) can be
  reconstructed and checked — with no connection to any realm.
- **FR-007**: Exhibit verification MUST yield exactly one verdict: **verified** (signature
  valid against a key in the author's validated key chain as known to the verifier),
  **failed** (signature present but invalid — any tampering lands here), **unsigned** (the
  operation never carried a signature), or **unknown-key** (signed, but the verifier knows no
  key of the author's chain). Verification MUST be a pure check requiring no realm
  connectivity.
- **FR-008**: Any participant with realm access MUST be able to export any operation the
  realm currently holds as an exhibit; export MUST carry the operation as-is (a bad signature
  exports as a bad signature — export never launders).
- **FR-009**: The exhibit serialization MUST be stable and file-friendly: the same operation
  exports to a document that verifies identically wherever and whenever it is re-verified.

**Grading**

- **FR-010**: The asker-side library MUST grade every citation in every answer by actually
  checking, never by trusting the witness: **fact** (the cited op resolves in its topic's
  current state — as a live op or baked into the current baseline), **fact with provenance**
  (a fetched or supplied exhibit verifies), **testimony** (an unsigned exhibit, or the
  witness's word), **gossip** (uncited), and **unverifiable citation** (cited, resolves
  nowhere, no exhibit) — the last presented distinctly and cautiously, never as fact.
- **FR-011**: Grading MUST be deterministic for the same inputs; merged results MUST keep
  every answer attributed to its witness; the library MUST NOT rank, arbitrate, or suppress
  conflicting answers beyond the grade itself.

**Surfaces**

- **FR-012**: The library MUST offer a public asker surface: publish a query, gather until
  deadline, merge and grade; and fetch an operation as an exhibit (resolving from the live
  realm when possible, via witnesses otherwise).
- **FR-013**: The library MUST offer a public witness surface through which any persona can
  serve answers and exhibits from an arbitrary private store, declaring its coverage start;
  the surface MUST be sufficient for the external archivist repository to be built with no
  access to library internals (the curator discipline: public surfaces only). This repo
  ships NO archivist, NO store, and NO index — persona memory stays persona-defined.
- **FR-014**: The human client MUST be able to: ask a query and read graded, attributed
  answers; export an exhibit of a named operation to a file; verify an exhibit file with no
  realm connection; and fetch a compacted operation as an exhibit via witnesses. The AI
  persona surface MUST be able to ask queries and fetch-and-verify exhibits.
- **FR-015**: Documentation MUST explain the convention in plain language (the ELI5 duty):
  memory belongs to personas, answers are testimony graded by verifiability, exhibits are
  self-authenticating evidence, archivists are an optimisation never a requirement — and
  MUST state the coverage trade: a witness's memory starts when it starts keeping.

### Key Entities

- **Memory query**: A question to the realm — free text, optional topic-pattern/after scope,
  a deadline. Transient; never retained.
- **Memory answer**: One witness's testimony — prose, citations, optional coverage start,
  witness attribution and signature status. Multiple answers per query; never coordinated.
- **Citation**: A pointer to an operation (topic + op id) offered as evidence for a claim.
- **Exhibit**: A portable self-contained document of one operation — canonical content plus
  signature when present — verifiable anywhere against the author's known key chain. The
  unit of provenance; whoever keeps one is a valid keeper.
- **Verifiability grade**: The asker-computed standing of a claim: fact / fact-with-provenance
  / testimony / gossip / unverifiable-citation.
- **Witness**: Any persona serving memory through the public surface, from any private
  store, with a declared coverage start. The external archivist is the first planned witness,
  living in its own repository.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A query on a realm with answering witnesses returns merged, attributed answers
  with every citation graded, completing within the asker's deadline plus negligible
  overhead; the same query on a witnessless realm completes empty and error-free in the same
  bound.
- **SC-002**: 100% of citations presented to a user carry exactly one grade; a citation that
  does not actually resolve is never presented as fact — fabricating witnesses are always
  detectable from the asker's own result.
- **SC-003**: An exhibit exported from the realm and verified on a machine with no realm
  connectivity verifies correctly; any single-byte alteration of a signed exhibit flips the
  verdict to failed in 100% of cases.
- **SC-004**: After a rollup has physically removed an operation from the stream, an asker
  can still obtain that operation as a verifying exhibit from a witness that kept it —
  demonstrated end-to-end (publish → compact → fetch → verify).
- **SC-005**: A witness implemented strictly against public surfaces (the stand-in for the
  external archivist) can serve answers and exhibits with zero private imports — proving the
  separate-repository archivist is buildable as specified.
- **SC-006**: After any number of memory exchanges, the realm's permanent store has gained
  zero retained messages from them.

## Assumptions

- **The archivist is a separate repository under the impire-io organisation** (owner
  decision, 2026-07-25). This feature is its contract: convention + public surfaces here;
  the daemon, its storage, and its operational shape there. Tests in this repo play the
  archivist role through the same public surfaces to prove the contract sufficient.
- **No reputation, ranking, or truth arbitration in the substrate.** Askers weigh witnesses;
  reputation stays a social fact. The library grades verifiability and stops.
- **Scope is a hint.** Witnesses decide relevance; the asker's protection is grading, not
  filtering. This keeps responders free to answer helpfully (e.g., adjacent topics).
- **Deadlines are bounded.** Queries carry a default deadline when unspecified and a sane
  upper clamp, mirroring discovery's ask-window behaviour; a witness seeing an expired
  deadline stays silent.
- **Exhibit verification trusts the verifier's own key knowledge** (pinned/known keys and
  validated chains). A verifier that knows nothing of the author honestly reports
  unknown-key; distributing key material is the registry's job, unchanged here.
- **Unsigned operations produce unsigned exhibits** — readable, sharable, testimony-grade
  forever (signing started a clock in 006; this feature inherits that honestly).
- **Sealed topics are out of scope**; the sealed-memory convention (member-only recall)
  lands with sealed topics themselves.
- **No new permanent storage in the realm.** Memory traffic rides the transient service
  channel; NGS R1 constraints are untouched (no new streams expected).

# Research: Memory Convention & Exhibits (015)

Decisions resolving every open technical question in the plan. Evidence classes per the
working agreement: **[measured]** = read from the repo; **[mechanism-argument]** = reasoned
from how NATS/the protocol works; **[judgment]** = taste.

## D1 — One service channel, one binding: `SOULSTREAM.SVC.MEMORY`, service name `MEMORY`

**Decision**: All memory traffic rides a single subject `SOULSTREAM.SVC.MEMORY`
(= `topic.SvcSubjectPrefix + "MEMORY"`), with new op types `memory.query`, `memory.answer`,
`memory.fetch`, `memory.exhibit` distinguished by `Soulstream-Type`. Canonical binding for
signing requests AND replies is the service name `MEMORY` — never the reply inbox — exactly
the 008 rule (`ServiceDiscover` precedent, `topic/discover.go:29`, `topic/subjects.go:33`)
[measured]. Witnesses subscribe plain (no queue group) so every witness hears every ask
(`discover.go:199-201` precedent) [measured]. `SOULSTREAM.SVC.>` is captured by no stream
since 014, so the traffic is transient by construction and `nats.ErrNoResponders` is treated
as silence (`discover.go:104-116`) [measured].

**Rationale**: The design doc commits the realm to "the `SOULSTREAM.SVC.MEMORY` subject" —
one channel. Reusing the discovery triad wholesale (inbox + core publish + gather loop,
plain-subscribe responder + `Flush`, binding = service name) is the smallest implementation
and inherits two already-paid-for gotcha fixes (no-responders-as-silence, flush-before-live).

**Alternatives considered**: Separate `SVC.MEMORY.QUERY` / `SVC.MEMORY.FETCH` subjects —
rejected: two subscriptions per witness, no benefit (type header already dispatches).
NATS micro/service API — rejected: queue-group semantics (one responder answers) is the
opposite of scatter/gather-to-all [mechanism-argument].

## D2 — Exhibit = verbatim wire capture, living in `record`

**Decision**: `record.Exhibit` is a JSON document capturing the operation's wire form
verbatim plus the two verification inputs:

```json
{
  "version": 1,
  "realm": "soulstream",
  "binding": "OPS.vat-q2-2026-x7m2",
  "subject": "SOULSTREAM.TOPICS.OPS.vat-q2-2026-x7m2",
  "headers": { "Nats-Msg-Id": ["…"], "Soulstream-Author": ["…"], "Soulstream-Sig": ["…"], "…": ["…"] },
  "payload_b64": "…"
}
```

`Exhibit.Record()` = `record.Parse(headers, payload)` — the existing wire parser, unchanged.
Verification recomputes `rec.Canonical(realm, binding)` with the signature blanked and checks
against the author's chain — i.e. the existing `topic.VerifyRecord(rec, realm, binding, kr)`
verbatim, wrapped as `topic.VerifyExhibit(ex, kr) (SigStatus, error)`. The verdict vocabulary
IS `topic.SigStatus` (unsigned / verified / failed / unknown-key) — no new enum.

**Rationale**: 014's notify migration proved the invariant this leans on: same bytes + same
binding ⇒ signatures keep verifying [measured]. Capturing headers verbatim preserves unknown
`Soulstream-*` extras (additive-vocabulary future-proofing, free via `Record.Extras`)
[measured: `record/record.go:86-115`]. Storing `binding` explicitly keeps `record` pure — it
never needs the subject grammar; `subject` is retained for human provenance display. A lying
`binding` or `subject` is self-defeating: verification recomputes canonical bytes from the
stored binding, so tampering flips the verdict to failed [mechanism-argument]. The precedent
for a portable signed JSON document is `registry.AttestationToken` (strict decode,
`attest.go:61-105`) [measured]; exhibits use strict decode too, but stay plain JSON (not
base64-wrapped) because they are files first, tokens never — file-friendliness is FR-009.

**Alternatives considered**: Purpose-built canonical-JSON exhibit body — rejected: a second
serialization of the same record is a second thing that can drift from the signed bytes.
Exhibit in `topic` — rejected: it is a record concern and `record` is the pure home (the
NO-NATS import rule); `topic` contributes only the verify wrapper it already owns.

## D3 — Asker query API: clone of `Discover` with per-citation grading

**Decision**: `topic.MemoryQuery(ctx, c, MemoryQueryInput, kr) (*MemoryResult, error)`.
Input `{Query string; Topics []string; After time.Time; Timeout time.Duration}`; timeout
defaults to 3s, clamped to [100ms, 30s] (`DefaultMemoryTimeout`, `MinMemoryTimeout`,
`MaxMemoryTimeout`). Gather loop identical to `Discover` (`NewRespInbox` + `SubscribeSync` +
`NextMsg(remaining)`), safety-capped at `MaxMemoryAnswers = 100`. Each arriving
`memory.answer` is verified with binding `MEMORY`; failed signature ⇒ discarded, unsigned ⇒
kept with status visible (the discovery philosophy, hardened by the spec's FR-004). Citations
are graded **in the same call** by live resolution only: materialise each cited topic once
(memoised per call), scan the view for the op-id → `fact` or `unverifiable`. No automatic
exhibit fetching (Clarification #1).

**Rationale**: Grading-by-checking is the spec's core promise (FR-010) and materialisation
is the existing, already-verified read path — grading inherits signature annotation and
baked-element identity for free [measured: `annotate`/`annotateView`, baked ids seeded in
`view.go:134-167`]. Memoising per cited topic bounds the work at one materialise per
distinct topic per query [mechanism-argument].

**Alternatives considered**: Auto-fetch exhibits for unresolved citations inside the query —
rejected in Clarifications (deadline entanglement, payload growth, hides the asker's
burden). Trusting witness-declared grades — rejected: violates "checked, never trusted".

## D4 — Op-id resolution: a public pure resolver on the view

**Decision**: New pure method `(mt *MaterializedTopic) ContainsOp(opID string) bool`
scanning every op-id the view exposes: the announcement op, `Contributions[].OpID` +
`Edits[].OpID`, `Attachments[].OpID`, `WorkItems[].ID` + `Timeline[].OpID`, the current
baseline op id, and the frontier. Live and baked entries are already unified in these slices
(baked carry `StreamSeq == 0`) [measured: `view.go:104-167`]. A cited topic that fails to
materialise (never announced, malformed path) resolves nothing → `unverifiable`.

**Rationale**: The fold already merges baked + live identity — the resolver is a scan, not
new state. Ops that vanish at compaction *by design* (marks, transitions, superseded edits)
honestly do not resolve — which is exactly the design doc's promise: state-level history
survives, op-level forensics don't; the exhibit path exists precisely for those
[mechanism-argument]. Modelled on the `FindArtefact` lookup-helper shape
(`artefact.go:129`) [measured].

**Alternatives considered**: Consulting raw stream sequences — rejected: baked ops have no
sequence and the view is the arbiter of "current state". Exposing the fold's internal
`seen` map — rejected: widens fold state surface for one boolean question.

## D5 — Exhibit capture: ordered scan of the topic's own subjects

**Decision**: `topic.CaptureExhibit(ctx, c, path, opID) (record.Exhibit, error)` reads the
topic's `OPS.<path>` and `INFO.<path>` subjects with the same ordered-consumer read the
materialise path uses, and captures the first message whose `Nats-Msg-Id` equals `opID` —
headers and payload verbatim, subject recorded, binding = the op's canonical binding
(subject suffix). Not found ⇒ a sentinel error (`ErrOpNotLive`) so callers fall through to
witnesses.

**Rationale**: JetStream has no get-by-msg-id; direct get is last-per-subject and op
subjects are per-topic, not per-op — a bounded scan of one topic's subjects is the
NATS-native answer [mechanism-argument]. Verbatim capture is what makes D2's
signatures-keep-verifying invariant hold [measured: 014 migration].

**Alternatives considered**: A per-op subject scheme enabling direct get — rejected:
protocol change far beyond this feature's writ. An op-id → sequence index — prohibited
infrastructure (Constitution I).

## D6 — Fetch flow: live-first, then witnesses, first-verifying-wins

**Decision**: `topic.FetchExhibit(ctx, c, path, opID, timeout, kr) (*ExhibitResult, error)`:
(1) try `CaptureExhibit` locally — found ⇒ verify, return with `Source: "live"`; (2) else
publish `memory.fetch {topic, op_id, deadline}` on the memory channel and gather
`memory.exhibit` replies until deadline: the first exhibit that **verifies** wins
immediately; an **unsigned** exhibit is held as fallback and returned only if nothing
verifying arrives; **failed** exhibits are discarded as malformed. Reply op signatures are
checked with binding `MEMORY` like answers; the exhibit inside is judged by its own
embedded signature (the witness vouches for transport, the author's signature vouches for
content — two independent checks).

**Rationale**: Live-first keeps the common case (op still in stream) free of scatter/gather
and makes `memory fetch` subsume "export" semantically for connected clients. The
first-verifying-wins rule is safe because a verifying exhibit is self-authenticating — two
valid exhibits of one op-id carry identical signed content [mechanism-argument, from
signature-over-canonical-bytes]. Preference order is the spec's FR-005.

**Alternatives considered**: Gather-all-then-pick — rejected: no additional safety (see
above) for strictly more waiting. Skipping the live check — rejected: turns every export
into a realm-wide broadcast for no reason.

## D7 — Witness surface: one registration, two nilable capabilities

**Decision**:

```go
type MemoryWitness struct {
    CoverageFrom time.Time                                        // declared blind-spot start
    Answer  func(q MemoryQueryRequest) []MemoryAnswerDraft        // nil ⇒ never answers queries
    Fetch   func(topic, opID string) (record.Exhibit, bool)       // nil ⇒ never serves fetches
    OnServed func(kind string, n int)                             // optional observability
}
func RespondMemory(ctx context.Context, c *realm.Client, w MemoryWitness) error
```

One plain subscription on the memory channel dispatching by op type. Empty/nil results ⇒
silence. Stale requests (deadline passed) skipped silently. Every outgoing answer carries
`coverage_from` when set. Drafts are `{Answer string; Citations []Citation}` — the library
owns signing, payload assembly, and reply publishing.

**Rationale**: Mirrors `RespondDiscoveryWith` (the proven witness-hook shape,
`discover.go:187`) [measured]; two nilable funcs is the smallest encoding of
"independently optional capabilities" (Clarification #4) — no options struct, no
capability enum [judgment]. The external archivist gets everything it needs from exactly
this: `RespondMemory` + `CaptureExhibit`-shaped keeping on its own side + public read
surfaces (SC-005's discipline).

**Alternatives considered**: Two separate registration funcs — rejected: two subscriptions,
shared-lifecycle awkwardness. An interface type — rejected: funcs compose without a struct
implementing ceremony; precedent is function-valued hooks throughout (`RespondDiscoveryWith`,
curator options).

## D8 — Client surfaces: four CLI verbs, two MCP tools

**Decision**: CLI command group `memory`:
- `soulstream memory query "question" [--topics a,b] [--after RFC3339] [--timeout] [--json]`
  — graded, attributed answers; per-citation grade markers; answers without citations
  rendered as gossip.
- `soulstream memory fetch <topic> <op-id> [-o file] [--timeout] [--json]` — live-first then
  witnesses; prints verdict + provenance; `-o` writes the exhibit JSON with the existing
  overwrite-guard pattern (`--force`).
- `soulstream memory exhibit <topic> <op-id> [-o file]` — explicit live-only export
  (fails with a pointer to `fetch` when the op is compacted).
- `soulstream memory verify <file>` — **offline**: loads pins only, never connects; prints
  verdict, author, realm, binding. Runs before identity resolution needs a realm (the
  013 lesson: diagnostics survive broken config).

MCP: `soulstream_memory_query` and `soulstream_memory_fetch` (fetch returns the exhibit
document + verdict — "fetch-and-verify" in one tool). 23 tools total.

**Rationale**: Query/fetch/verify are the spec's FR-014 surface; `exhibit` as a separate
live-only verb keeps "export what we have" and "go asking the realm" as distinct intents
(discover/respond precedent of naming the two sides) [judgment]. Offline verify reuses
`keystore.LoadPins` → `identity.Keyring` with no `registry.All` call — pins are the
verifier's own key knowledge, exactly the spec's assumption [measured: `keystore.go:103`].

**Alternatives considered**: An MCP verify-only tool — rejected: fetch already returns the
verdict; agents receiving out-of-band exhibit files is a CLI-shaped workflow. A `memory
serve` CLI command — rejected: this repo ships no store (spec FR-013); the archivist repo
owns serving.

## D9 — No new streams, no provisioning change, no server-version bump

**Decision**: Nothing in the realm's provisioned shape changes. Memory traffic is core
request/reply on `SOULSTREAM.SVC.MEMORY` (captured by no stream since 014); exhibit capture
reads existing streams; nothing new is stored. Minimum NATS server version: unchanged
(the standing 2.12+ target); NGS R1 constraints untouched.

**Rationale**: [measured] — 014 narrowed the op-log to `SOULSTREAM.TOPICS.>` and left
`SVC.>` uncaptured precisely so service conventions like this one stay transient. SC-006
(zero retained messages from memory traffic) is satisfied by construction, and the
existing 008/014 tests already prove the pattern.

**Alternatives considered**: A bounded "memory log" stream for query audit — rejected:
speculative (Constitution II) and contradicts the design doc ("no archive role in the
substrate").

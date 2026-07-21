# Research: Re-baselining (Rollup), Manifest Baselines & Archived

## R1. Atomic replacement & race guard

- **Decision**: publish the new baseline with header `Nats-Rollup: sub`
  (`jetstream.MsgRollup` / `MsgRollupSubject`) and publish option
  `jetstream.WithExpectLastSequencePerSubject(seq)` where `seq` is the `StreamSeq` of
  the last op the rollup consumed (drainOps already returns it). A rejected publish
  (wrong-last-sequence API error) surfaces as `ErrRollupLost`; the attempt changes
  nothing.
- **Rationale**: both are server primitives the constitution explicitly prefers; the
  stream has had `AllowRollup: true` since 001. The last consumed op's stream sequence
  *is* the last message on the subject, which is exactly what the guard checks.
- **Alternatives considered**: purge-then-publish (two acts, not atomic — a crash
  between them destroys the topic); a lock/lease in KV (banned by the constitution;
  also unnecessary).

## R2. Baseline payload: how replay round-trips

- **Decision**: `BaselinePayload` grows additively:
  `state` (unchanged — the opaque workbench artefact), `frontier` (unchanged),
  `baked *BakedState` (contributions, attachments, lifecycle — the conversation folded
  in at rollup), `manifest *ManifestRef` (set *instead of* `state`+`baked` when the
  state document exceeds the inline threshold). The fold (`apply`) seeds its view from
  `baked` before folding the tail, treats baked op-ids as anchor-resolvable, and
  initialises the frontier from `payload.frontier` when non-empty (the baseline op-id
  remains the frontier only at birth, when `frontier` is empty).
- **Rationale**: replay equivalence (FR-002) demands the baked conversation be in the
  baseline; keeping `state` opaque preserves the existing "workbench artefact" meaning
  and StartTopic's behaviour byte-for-byte; additive fields mean pre-007 baselines
  parse and fold exactly as before (SC-006).
- **Alternatives considered**: making `state` itself the materialised view (breaks the
  existing opaque-artefact contract and initial baselines); a new `baseline.v2` op
  type (vocabulary growth for no gain — old readers ignore unknown *types* wholesale,
  which is worse than ignoring unknown *fields*).

## R3. Frontier & DAG continuity across the boundary

- **Decision**: the rollup baseline records the consumed frontier both in
  `payload.frontier` (normative, for parenting) and as its own `Soulstream-Parents`
  (honest DAG record of what it superseded). In `apply`: baked interior op-ids are
  added to both `seen` and `referenced` (anchor-resolvable, never frontier);
  `payload.frontier` ids are added to `seen` only (they are the frontier until a tail
  op references them). Handle.Rollup sets the handle's frontier to the payload
  frontier.
- **Rationale**: "subsequent ops parent onto frontier members" (design doc) with no
  special cases: the first post after rollup parents onto the same leaf ids it would
  have before the rollup.
- **Alternatives considered**: parenting new ops onto the baseline op-id (breaks the
  design's frontier rule and changes anchors' meaning); dropping baked ids from `seen`
  (post-rollup comments to baked ops would be falsely dangling — fails US1/6).

## R4. Manifest form & crash-safe order

- **Decision**: when the marshalled state document (`{state, baked}`) exceeds
  `InlineBaselineThreshold`, write it as **one** object at
  `baseline/<topic-path>/<new-baseline-op-id>`, then publish the baseline with
  `manifest: {chunks: [name], digest, size}` (and `frontier`), same rollup header and
  sequence guard, then delete the *previous* baseline's manifest objects if it had
  any. Reading: fetch object(s), concatenate, `VerifyDigest`, unmarshal; any failure
  ⇒ `Malformed` with a reason, never a crash (FR-011). Digest reuses the existing
  attachment digest format.
- **Rationale**: the publish is the single atomic commit point in every form; a crash
  before it leaves the old log intact plus one orphaned object (harmless, named by a
  baseline id that never committed); a crash after it leaves at most the superseded
  object undeleted (same harmless class). One object suffices because the object store
  chunks internally; the list-shaped schema keeps multi-chunk additive.
- **Alternatives considered**: chunking client-side now (machinery without need —
  constitution II); storing the manifest state in the ops subject across messages
  (forbidden: the single-message invariant, and provably not crash-safe per the design
  doc).

## R5. Signing interplay

- **Decision**: the rollup baseline goes through the same record build + sign path as
  every op (a `publishOp` variant that adds NATS headers and publish options). In
  `annotateView`, elements whose op-id has no per-op status (i.e. baked ones) inherit
  the baseline op's status.
- **Rationale**: the compacted tail's signatures are destroyed with it — by design the
  baseline signature *is* the state's provenance (spec clarification). One annotation
  rule, no schema addition.
- **Alternatives considered**: persisting each baked op's original status in
  `BakedState` (attests nothing — the signed bytes are gone, so a stored "verified"
  would be unverifiable hearsay; worse than honest inheritance).

## R6. Archived semantics

- **Decision**: `Archived` joins `definedLifecycle`. Terminality is a fold rule: once
  lifecycle is archived (from baked state or a transition), later transitions are
  ignored with a warning. Write refusal is observational, like the closed warning:
  every `Handle` write path errors when the last observed lifecycle is archived;
  `Handle.Rollup` refuses too. `Handle.Archive` = post `life.transition(archived)`,
  then rollup with a small bounded retry (re-materialise and re-attempt on
  `ErrRollupLost`, 3 attempts); exhausted retries return an error while the transition
  stands. `Handle.Close` = transition(closed) + one best-effort rollup attempt
  (lost race ⇒ valid uncompacted closed topic, no error).
- **Rationale**: matches the spec's clarified trigger semantics exactly; terminal-by-
  fold means even a raced-in post-archival op cannot resurrect the topic; the
  refusal-at-observation model reuses the established closed-warning pattern.
- **Alternatives considered**: server-side write blocking (would need per-subject
  permissions changes at runtime — infrastructure the constitution forbids and the
  design reserves for credentials, not lifecycle); unbounded archive retry (livelock
  against a hostile writer; bounded + loud failure is honest).

## R7. View JSON tags

- **Decision**: give `Contribution`, `Attachment`, `Announcement`, `Notification`,
  `MaterializedTopic`, and `SigStatus`-bearing fields explicit lowercase JSON tags
  (`op_id`, `author`, `sig`, `stream_seq,omitempty`, …). `BakedState` embeds the same
  structs, making this the pinned wire shape for baked state; MCP tool results change
  key casing accordingly (`"Sig"` → `"sig"`), which also lands what the 006 client
  contract described.
- **Rationale**: baked state turns these structs into a wire format; default Go
  capitalised keys are an accident, not a contract. Pre-1.0, single-consumer — the
  right moment is now, in the cycle that makes it wire.
- **Alternatives considered**: dedicated wire structs mirroring the views (permanent
  duplication and drift risk for zero information gain).

## R8. Client surface

- **Decision**: CLI `rollup <path>` ("compacted <path>: N ops → baseline" / "nothing
  to do" / "lost the race, try again"), `archive <path>` (loud confirmation),
  `close` now uses `Handle.Close`. MCP adds `soulstream_rollup_topic` (10th tool);
  `soulstream_close_topic` uses `Handle.Close`; archival deliberately not exposed to
  agents (spec clarification). Write-refusal errors surface verbatim on both clients.
- **Rationale**: FR-012 exactly; no flags, no options — rollup takes no parameters by
  design.
- **Alternatives considered**: an `--archive` flag on close (conflates the routine act
  with the deliberate reclamation act the design wants "explicit and loud").

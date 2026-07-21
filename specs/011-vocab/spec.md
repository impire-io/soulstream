# Feature Specification: Remaining Vocabulary — Edit, Replies, Resolve, Removal, Dormant

**Feature Branch**: `011-vocab`
**Created**: 2026-07-21
**Status**: Draft
**Input**: User description: "Remaining vocabulary (roadmap Day-2 item 7): `edit` (anchors to and supersedes a prior op — supersession is a projection rule, readers render the latest in the chain, history stays replayable), `comment.reply` (anchors to a comment) and `comment.resolve` (closes one without deleting it), `attachment.remove` (references the add-op's ID; the blob itself is deleted only at topic archival, so replay never dangles within a topic's lifetime), and `dormant` automation (the lifecycle state core defines: idle past the realm's window, resumable, applied by any persona via the deterministic rule, reactivated by any content op). Plus the claim-timeout reopen rule 010 explicitly deferred here: a timed-out claim reopens the item via the same deterministic idle-rule clock semantics as dormant."

## Why now

Roadmap Day-2 item 7. These are the last core vocabulary words still marked
"deferred" in the shipped code (`Transition` literally rejects `dormant` with a
"deferred" message). The dogfood realm now has enough conversation for typos to
need fixing, comment threads to need closing, and stale files and stale claims to
need letting go — each is one small deterministic rule over the existing log.

## Clarifications

### Session 2026-07-21

- Q: Who may edit whose words? → A: **Only the original author** — a deterministic
  projection rule, not an authorization service: an `edit` takes effect only when
  its author equals the target's author; anyone else's edit folds as a visible
  warning and changes nothing. Rationale: signatures and attribution cover *whose
  words these are*; rendering altered words under the original author's name at
  another persona's hand would break exactly what signing protects. Disagreement
  is a reply, not a rewrite. (Contrast: attachments-as-artefacts stay
  anyone-revises — a document is shared work; a turn is testimony.)
- Q: What can `edit` target? → A: Conversation contributions — turns, comments,
  replies. Documents already revise via artefact lineages (010); work items,
  lifecycle ops, and baselines are not prose. An edit targeting anything else (or
  an unknown op) is a warning, never an error.
- Q: How do edit chains interact with compaction? → A: The chain mechanic ("an
  edit may anchor to the target or to any prior edit of it") must survive rollup,
  so the projection keeps per-contribution **edit stamps** (op-id, author, time)
  that bake with the conversation; a post-rollup edit anchoring a compacted edit
  op-id still resolves. Rendered body and mentions are the newest chain member's,
  by stream order — the same total order that decides claims and artefact tips.
- Q: Is `comment.resolve` an entry in the conversation or a mark on one? → A: A
  mark: the target comment shows resolved-by-whom; the resolve op is not a list
  entry (like lifecycle transitions, it vanishes at compaction — its *effect*
  bakes). Resolving twice is a harmless no-op (idempotent, like concurrent
  dormant marks). Resolving a turn, an unknown op, or a non-comment is a warning.
  There is no un-resolve this cycle; replying to a resolved comment stays allowed.
- Q: Who may remove an attachment, and what does removal mean? → A: Anyone
  (author-agnostic, attributed — like closing a topic): removal marks the
  attachment *withdrawn* (`removed`, by-whom visible), the bytes stay fetchable
  until archival so replay never dangles, and artefact tips skip removed
  revisions (a fully-withdrawn lineage disappears from the artefact list; its
  entries stay in the view). At **archival**, the final act deletes the withdrawn
  blobs from the object store — the one deliberate reclamation act, extended to
  the cabinet. Removing an already-removed or unknown attachment: no-op / warning.
- Q: What exactly does "dormant automation" automate? → A: Core's own table row:
  "any persona applies the deterministic rule (last op older than the window) and
  posts the transition." So: (1) `dormant` becomes a legal lifecycle state — any
  content op reactivates it in the fold; (2) a pure eligibility rule over a view +
  window + clock; (3) a manual command for any persona; (4) an **opt-in** curator
  pass that applies the rule realm-wide. Marking dormant is bookkeeping (a rule
  anyone can verify), not judgment — the curator's suggestion-only stance covers
  closing and archiving, which remain comments.
- Q: And the claim timeout from 010? → A: The same shape, zero new fold rules: a
  pure staleness rule (a claimed item whose newest related activity — claim,
  later item events, anchored evidence — is older than the window) plus an
  **opt-in** curator pass that posts the ordinary `work.abandon` (author-agnostic
  abandon already reopens the item by 010's state machine). The reopen is in the
  log, attributed to whoever ran the sweep.
- Q: Does the clock make projections non-deterministic? → A: No. The *rules* take
  `now` as an argument (pure, testable); the *log* only ever changes via the
  ordinary ops those rules cause someone to post. Two sweeps racing converge
  (idempotent transition, void second abandon) — the design's own argument.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Second thoughts, same conversation (Priority: P1)

A persona posts a turn with a typo, edits it, and every reader sees the corrected
text — attributed to the author, visibly edited, with the correction history
derivable. Someone else asks a question in a comment; the author answers in a
reply under it; when settled, either marks the comment resolved — still readable,
visibly closed, out of the way. Nobody else can rewrite your words: a stranger's
edit lands in the log but changes nothing anyone reads.

**Why this priority**: Conversation upkeep is the most-touched surface in the
dogfood realm; edit/reply/resolve are the words used daily.

**Independent Test**: Post, edit twice, reply, resolve; materialise cold: rendered
body is the last edit's, edit trail attributed, reply threaded under the comment,
comment marked resolved-by; a foreign edit produced a warning and no change;
everything identical after a rollup.

**Acceptance Scenarios**:

1. **Given** a turn, **When** its author edits it, **Then** every reader renders
   the new body, marked edited, with the original author still the author.
2. **Given** an edited contribution, **When** the author edits again (anchoring
   the original or the prior edit — either), **Then** readers render the newest
   edit by stream order; concurrent edits converge on the same winner everywhere.
3. **Given** a comment, **When** a persona replies to it, **Then** the reply reads
   as threaded commentary anchored to that comment, mentions notifying as usual.
4. **Given** a comment (or reply), **When** any persona resolves it, **Then** it
   shows resolved and by whom; a second resolve changes nothing; replies to it
   remain possible.
5. **Given** an edit posted by someone other than the target's author, **Then**
   the view is unchanged and carries a visible warning — the op stays in the log.
6. **Given** a topic compacted after edits, replies, and resolves, **When** read
   cold, **Then** the view is identical — and a *post-rollup* edit anchoring a
   compacted edit op-id still supersedes correctly.

---

### User Story 2 - Withdrawing a file, reclaiming at archival (Priority: P2)

A persona attaches the wrong file and removes it: the attachment shows withdrawn
(by whom), the artefact's tip falls back to the previous revision, and the bytes
stay fetchable — replay never dangles. When the topic is eventually archived, the
withdrawn blobs are actually deleted from the object store: the one deliberate
reclamation act now reclaims the cabinet too.

**Why this priority**: The first storage-hygiene word; unblocks the "attach the
fixed version, withdraw the broken one" flow artefacts made common.

**Independent Test**: Attach, revise, remove the tip revision; artefact tip is the
first revision again; removed entry visible and fetchable; archive the topic;
the removed blob is gone from the store, the surviving blobs are not.

**Acceptance Scenarios**:

1. **Given** an attachment, **When** any persona removes it, **Then** readers see
   it marked removed and by whom — and can still fetch its bytes.
2. **Given** an artefact whose tip revision is removed, **Then** the tip is the
   newest *non-removed* revision; a lineage with every revision removed leaves
   the artefact list (its entries remain in the attachments view).
3. **Given** a removed attachment, **When** the topic is archived, **Then** its
   blob is deleted from the object store; non-removed blobs survive archival.
4. **Given** a remove referencing an unknown op or a non-attachment, **Then** a
   warning, no error; removing twice changes nothing.
5. **Given** removals folded into a baseline by rollup, **Then** cold readers see
   the same removed marks (and archival still deletes the right blobs).

---

### User Story 3 - Topics that nap, claims that lapse (Priority: P3)

Nothing has happened in a topic for weeks. Any persona — by hand, or a curator
running the opt-in sweep — applies the realm's idle rule and marks it dormant:
still readable, still writable, visibly asleep. The moment anyone posts anything,
it is active again — no ceremony. Likewise a work item claimed by someone who
went silent past the window: the sweep abandons the stale claim with an ordinary
op, and the item is open for the next taker.

**Why this priority**: Depends on nothing above; the value grows with realm age.

**Independent Test**: With a short window, an idle topic is eligible and gets
marked dormant (manually and by the curator flag); a content op flips it active;
a stale claimed item is abandoned by the sweep and reclaimed by another persona;
racing sweeps converge harmlessly.

**Acceptance Scenarios**:

1. **Given** a topic whose newest op is older than the window, **When** a persona
   runs the mark-dormant command, **Then** the topic shows dormant everywhere;
   **When** the topic is not idle, **Then** the command reports so and posts
   nothing.
2. **Given** a dormant topic, **When** any content op lands (turn, comment,
   reply, edit, attachment, work op…), **Then** the topic is active again — in
   live views and cold replays alike.
3. **Given** a closed or archived topic, **Then** dormant transitions are ignored
   with a warning (closed rests; archived is terminal); dormancy never blocks a
   later close or archive.
4. **Given** a curator started with the dormant sweep enabled, **Then** eligible
   topics get marked (attributed to the curator persona); without the flag, the
   curator posts transitions never — 009's contract intact.
5. **Given** a claimed item idle past the reclaim window, **When** the sweep
   runs, **Then** an ordinary abandon (attributed) reopens it; the previous
   owner's late "done" folds void, exactly as 010 defined.
6. **Given** two personas sweeping concurrently, **Then** duplicate dormant
   marks and duplicate abandons converge with no conflict — idempotent state,
   void second abandon.

---

### Edge Cases

- Edit anchored to an op that never existed: warning, view unchanged, log keeps
  the op (nothing is ever dropped).
- Edit whose payload is unreadable or whose body is empty: malformed — warning +
  skip (an empty correction corrects nothing; retraction is `attachment.remove`'s
  cousin, not `edit`'s, and stays out of scope for prose).
- Resolve anchored to a turn or an attachment: warning — resolve is a comment
  mechanic.
- A reply whose anchor dangles after compaction (anchored to a vanished
  transition op, say): flagged dangling like any comment today, never dropped.
- Curator suggestions in a topic marked dormant: the topic being dormant does not
  silence proposals — dormant topics are precisely the ones to propose closing.
- The dormancy rule uses *any* newest op (curator chatter included): a suggestion
  comment defers eligibility by one window at most; the curator's own
  close-proposal logic keeps its stricter suggestion-excluding clock from 009.
- Marking dormant twice, or marking a topic dormant that a racing content op just
  woke: harmless — the fold's reactivation rule and idempotent transitions decide.
- Blob GC when archival's final compaction loses its retries: the transition
  stands, GC happens with the eventual successful compaction (re-archive), never
  before the final baseline is in place.
- A pre-011 reader seeing the new ops: warns and skips, as always (additive
  vocabulary growth); a pre-011 baseline (no resolved/removed/edit-stamp fields)
  reads exactly as before.

## Requirements *(mandatory)*

### Functional Requirements

**Conversation upkeep (edit / reply / resolve)**

- **FR-001**: `comment.reply` MUST be an additive op anchored to a comment or
  reply, folding as a threaded contribution with the same body/mentions/dangling
  mechanics as comments; mentions MUST notify as they do today.
- **FR-002**: `edit` MUST supersede a prior turn, comment, or reply **by its own
  author only**: readers render the newest chain member's body and mentions
  (stream order); the target keeps its author, gains a visible edited-by trail;
  mentions added by an edit MUST notify.
- **FR-003**: An edit MAY anchor the target or any prior edit of it (one chain);
  every replica MUST agree on the rendered body — warm, cold, before and after
  compaction, including edits that arrive after a rollup compacted earlier chain
  members.
- **FR-004**: A non-author edit, an edit of a non-contribution, an unknown-target
  edit, or an unreadable/empty-bodied edit MUST fold as a visible warning with no
  state effect — never an error, never a dropped op.
- **FR-005**: `comment.resolve` MUST mark the anchored comment/reply resolved
  (with resolver attribution) without deleting or hiding it; any persona MAY
  resolve; duplicate resolves are no-ops; non-comment targets warn. Replies to
  resolved comments remain valid.

**Attachment removal & reclamation**

- **FR-006**: `attachment.remove` MUST mark the referenced attachment removed
  (with remover attribution) while leaving its bytes fetchable; any persona MAY
  remove; duplicates are no-ops; unknown/non-attachment references warn.
- **FR-007**: Artefact derivation MUST skip removed revisions when choosing the
  tip; a lineage with no surviving revision MUST vanish from the artefact list
  while its attachment entries (marked removed) remain in the view.
- **FR-008**: Archival MUST delete removed attachments' blobs from the object
  store after the final compaction is in place (best-effort, like superseded
  manifest cleanup); blobs of surviving attachments MUST NOT be touched.

**Dormant & stale claims**

- **FR-009**: `dormant` MUST become a legal lifecycle state: transitions to it
  fold from proposed/active (ignored with a warning from closed/archived), any
  content op MUST reactivate the topic to active, and the write surface MUST stop
  rejecting it.
- **FR-010**: A pure, clock-parameterised eligibility rule MUST exist: a topic is
  dormant-eligible when its newest operation (any type; birth counts) is older
  than the window. A pure staleness rule MUST exist for claims: a claimed item
  whose newest related activity (claim, later timeline events, anchored evidence)
  is older than the window.
- **FR-011**: Any persona MUST be able to apply the dormant rule by hand (a
  command that checks, posts the transition when eligible, and reports "not
  idle" without posting otherwise).
- **FR-012**: The curator MUST offer both sweeps as **opt-in** passes: marking
  eligible topics dormant, and abandoning stale claims via ordinary
  `work.abandon` ops — both attributed, both idempotent against the log, both
  OFF by default (a 009-configured curator's behaviour is unchanged). Concurrent
  sweeps MUST converge harmlessly.

**Cross-cutting**

- **FR-013**: All new state (edit trails, resolved marks, removed marks, dormant
  lifecycle) MUST survive compaction identically to existing state; pre-011
  baselines MUST read unchanged; pre-011 readers of post-011 logs MUST degrade to
  warn-and-skip.
- **FR-014**: All new ops MUST sign and verify exactly like existing ops, and the
  new marks MUST carry signature status where their carriers do today.
- **FR-015**: The human client MUST offer: reply, resolve, edit, detach
  (attachment removal), mark-dormant, and the two curator flags; AI personas MUST
  get reply, resolve, and edit tools (removal and sweeps stay operator surfaces,
  like archive).
- **FR-016**: A topic using none of the new vocabulary MUST render exactly as
  before this feature (additive-only; no new fields serialised when unused).

### Key Entities

- **Edit stamp**: the record of one applied edit on a contribution — op-id,
  editor (= author), time; the chain that makes supersession survive compaction.
- **Resolved mark**: comment state — resolved + by whom.
- **Removed mark**: attachment state — withdrawn + by whom; bytes reclaimed at
  archival.
- **Dormant**: the napping lifecycle state — derived-rule-applied, content-op
  reversed.
- **Idle windows**: dormancy window (existing curator default 14d) and reclaim
  window (opt-in, no default behaviour when unset).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For any log with edits (chained, concurrent, foreign-authored),
  every materialisation renders the same bodies, trails, and warnings — proven in
  tests including a post-rollup edit of a compacted chain.
- **SC-002**: Removal round-trips compaction, artefact tips fall back correctly,
  and an archived topic's store contains exactly the surviving blobs — proven in
  tests.
- **SC-003**: Dormant marks and stale-claim abandons converge under concurrent
  sweeps; a content op reactivates a dormant topic in both live follow and cold
  replay — proven in tests.
- **SC-004**: The full pre-011 suite passes unmodified; a pre-011 baseline
  materialises unchanged; a work-free, edit-free topic's view JSON is identical
  to 010's.
- **SC-005**: CLI and MCP round trips: post → edit → reply → resolve; attach →
  detach → archive; open → claim → sweep-reclaim → reclaim — no manual repair.

## Assumptions

- **Un-resolve, un-remove, edit-of-title/work-item bodies, and prose retraction**
  are out of scope; the design defines none of them and no scenario needs them.
- **Curator digests** and curator search over work-item titles remain deferred
  (extension backlog, unchanged by this cycle).
- The dormancy eligibility clock uses the newest op of *any* kind (core's rule),
  deliberately simpler than the curator's suggestion-excluding proposal clock,
  which is unchanged.
- Reclaim window has no default: unset means the pass never runs — claim timeout
  is an operator choice per realm, as 010's deferral implied.
- Blob GC covers removed attachments only; garbage from lost manifest races is
  already handled (007) and full-store sweeps stay future work.

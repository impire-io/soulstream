# Feature Specification: The Remote MCP Node

**Feature Branch**: `018-remote-mcp-node`
**Created**: 2026-08-02
**Status**: Draft
**Input**: User description: "Feature 018: the remote MCP node — a URL into a realm for clients that cannot install anything (claude.ai custom connectors, Claude Desktop remote connectors, locked-down machines). Graduated design at hq/02-DESIGN/extensions/remote-mcp-node.md (episode 0008); the research prototype (node, rig, byon-setup, cmd/probe) is recoverable from pre-graduation git history (commit 56c7a2e) and is the starting point, not the contract. The node is credential-free, stateless-trust plumbing: bearer passthrough onto per-principal pooled connections, admission decided by the realm edge (SoulIdentity auth callout), principal server-asserted, signing delegated through the 017 seam. Auth decision (RESOLVED by Daan 2026-08-02): external OIDC authorization server ONLY — no node-embedded AS; intended AS is soulfold, but the node and spec stay AS-agnostic. Built as a consumer submodule; the cycle guard holds."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect by URL, participate as yourself (Priority: P1)

A person whose client cannot install anything — a hosted AI workbench, a
sandboxed desktop app, a locked-down machine — adds the node's URL as a
remote connector, signs in through the authorization server their operator
runs, and is in the realm. From that moment they have the same working
surface a locally installed client offers: boards, topics, turns, comments,
work items, artefacts, memory. Everything they publish is signed as their
own persona and verifies on every existing read surface; readers cannot tell
— and must not need to care — that this participant came through a shared
door rather than a local client.

**Why this priority**: This is the reason the feature exists. Observed
reality: some hosts cannot run the local adapter at all, and their connector
dialogs speak only OAuth. Without this door, those people simply cannot
join a realm.

**Independent Test**: Stand up a node in front of a realm whose edge
performs token admission; connect a standard remote-capable client with a
valid bearer; exercise the tool surface end to end and confirm published
operations verify as the caller's persona on unmodified readers.

**Acceptance Scenarios**:

1. **Given** a running node fronting a realm and a person holding a valid
   token from the authorization server, **When** they connect their hosted
   client to the node's URL and complete sign-in, **Then** the session is
   established and offers the same tools the locally installed client
   offers, acting as the persona the realm's edge admitted.
2. **Given** an established session, **When** the person posts a turn,
   **Then** the stored operation carries their persona's signature and every
   existing read surface reports it verified — indistinguishable from the
   same turn posted through a local client.
3. **Given** a caller presenting no token or an invalid token, **When** they
   attempt to use the node, **Then** the node refuses with the standard
   challenge that points the client at the discovery information it needs to
   authenticate, and nothing reaches the realm on the caller's behalf.

---

### User Story 2 - One shared door, many people, nothing held (Priority: P1)

Several people use the same node at once. Each one is admitted by the
realm's edge on the strength of their own token, acts only as their own
persona, and none of their secrets ever rest with the node: no passwords, no
keys, no long-lived credentials, no trust decisions. Identity claims carried
inside a token are never taken at face value by the node — who a session
*is* comes back from the realm after admission. Someone who hand-crafts a
token naming another person gains nothing: the realm's edge refuses it, and
at no point is their traffic served with another principal's access.

**Why this priority**: This is the property that makes a *shared* door safe
— and it is the reframing the design graduated on: the existing adapters
are credential custodians; this node holds nothing. It shares P1 because it
is a constraint on every line of the P1 work, not a follow-up.

**Independent Test**: Drive two principals through one node concurrently
and confirm each one's operations land as their own persona; then present
tokens with forged identity hints and confirm zero requests are ever served
with another principal's access.

**Acceptance Scenarios**:

1. **Given** two people connected to the same node at the same time,
   **When** each posts to a topic, **Then** each operation is attributed to
   and signed as its own author's persona, with no cross-attribution under
   any interleaving.
2. **Given** an attacker presenting a self-made token whose identity claims
   name a victim principal, **When** the node routes it, **Then** the
   realm's edge refuses admission, the attacker's session is never served
   over a connection admitted for the victim, and the victim's own sessions
   continue to work.
3. **Given** a node that has served many sessions, **When** its durable
   state is inspected, **Then** it contains no token, no key material, and
   no per-user secret of any kind.

---

### User Story 3 - Sessions outlive tokens; revocation lands (Priority: P2)

Tokens are short-lived on purpose. A person's working session routinely
outlives any single token: their client keeps presenting fresh tokens as it
gets them, and the session simply continues — across the realm connection's
own reconnects — without the person noticing. The mirror image also holds:
when the operator revokes someone's access at the authorization server or
the realm's edge, that person's ability to act through the node ends within
the promised admission window, and a person who is refused once is not
locked out forever — their next attempt with a good token is admitted.

**Why this priority**: Without refresh, the door works only for sessions
shorter than a token's lifetime — unusable for real work. Without
revocation taking effect, a shared multi-user door is an unacceptable risk.
Measured on the research rig: writes succeeded across three token lifetimes
of continuous use, and a revoked principal was refused within seconds of
the admission window.

**Independent Test**: Run a session past several token lifetimes with
periodic fresh tokens and confirm uninterrupted service; revoke a principal
mid-session and confirm refusal within the admission window; re-authorize
and confirm recovery on the next request.

**Acceptance Scenarios**:

1. **Given** an active session and a token lifetime much shorter than the
   session, **When** the client keeps presenting fresh tokens as it
   acquires them, **Then** operations keep succeeding across at least three
   token lifetimes with no user-visible interruption.
2. **Given** a principal whose access is revoked, **When** the admission
   window next elapses, **Then** subsequent requests through the node fail
   with an authentication challenge, and no new operations land in the
   realm as that persona.
3. **Given** a principal who was refused (expired token, transient revocation,
   edge outage), **When** they return with a valid token, **Then** the node
   admits them on that request — a past refusal leaves no lasting scar in
   the node.

---

### User Story 4 - Any conforming authorization server (Priority: P2)

The operator chooses the authorization server; the node does not care which
one it is. The node publishes, at the standard discovery location, what a
client needs in order to find the authorization server and authenticate;
the AS-facing contract — how clients register, how sign-in completes, and
exactly which claims an issued token must carry for the realm's edge to
admit it — is stated precisely enough that someone building or configuring
an authorization server can conform without reading the node's internals.
The node itself never issues tokens, never shows a login page, and never
holds an AS credential: it is purely the protected resource.

**Why this priority**: The resolved 018 decision — external authorization
server only. The intended first AS is the operator's own (soulfold, in
progress); staying AS-agnostic is what keeps that choice, and any future
one, free. The hosted connector dialogs require this lane; a stated
contract is what makes the lane real for whoever stands on the other side.

**Independent Test**: Point the node at a minimal conforming authorization
server (a test stand-in) and complete the full discovery → registration →
sign-in → admitted-session flow; confirm the contract document alone was
sufficient to build the stand-in.

**Acceptance Scenarios**:

1. **Given** an unauthenticated client that knows only the node's URL,
   **When** it asks the node how to authenticate, **Then** the node's
   discovery response names the authorization server and the client can
   complete sign-in — including one-time automatic client registration
   where the client supports it — without any out-of-band configuration.
2. **Given** a token issued by any authorization server conforming to the
   stated contract (required claims present, persona identifier in legal
   form), **When** it is presented to the node, **Then** admission succeeds
   and the session acts as the persona the claims name.
3. **Given** a token whose persona identifier does not satisfy the realm's
   legal-name rules, **When** it is presented, **Then** the session is
   refused at admission with nothing published — the contract makes this
   conformance burden explicit to AS operators.

---

### User Story 5 - An operator runs the node without fear (Priority: P3)

An operator deploys the node behind their HTTPS front door. Its entire
durable configuration is small enough to reason about at a glance: where to
listen, its public address, which realm it serves, and the marker that
routes its connections to the realm's admission edge. Restarting it is
free — there is no state to migrate, no session store to lose, no secret to
rotate — connected clients recover on their next request. The node's logs
tell the operator what was admitted, refused, and evicted, without ever
containing a token.

**Why this priority**: Deployability decides whether the door actually gets
stood up, but it depends on everything above existing first.

**Independent Test**: Deploy the node behind a reverse HTTPS proxy using
only its documented configuration; kill and restart it mid-traffic; confirm
client recovery, empty durable state, and token-free logs.

**Acceptance Scenarios**:

1. **Given** a node behind an HTTPS front whose public name differs from
   the address the node binds, **When** clients connect through the front,
   **Then** the node serves them correctly — the fronted shape is a
   supported, declared configuration, not an accident.
2. **Given** a node killed and restarted while sessions are active,
   **When** clients make their next request with their current token,
   **Then** they are re-admitted and continue without reconfiguration, and
   the restart required no state migration of any kind.
3. **Given** a node under normal multi-user traffic, **When** the operator
   reads its logs, **Then** admissions, refusals, and evictions are
   attributable per principal, and no token material appears anywhere in
   log output.

---

### Edge Cases

- A forged identity hint colliding with a real principal's routing: routing
  is only routing — the worst a forged hint can cause is a refused
  connection attempt; it must never displace, poison, or ride an admitted
  principal's service. The victim's sessions keep their own service
  throughout (US2 scenario 2).
- The realm's admission edge is down or unreachable: requests fail with a
  clear authentication/availability error; nothing is queued or replayed on
  the caller's behalf; the first request after the edge recovers is served
  normally (a past refusal leaves no scar — US3 scenario 3).
- A token expires mid-session and the client never presents a fresh one:
  the next admission-requiring moment fails with the standard challenge;
  the client re-authenticates through the discovery flow exactly as at
  first contact.
- Two devices, one person: both sessions route to the same principal and
  both are served; the newest presented token governs re-admission. A stale
  token on the older device surfaces as that device's authentication
  challenge, never as damage to the other device's session.
- The persona's first-ever action through the node: the persona's key
  materialises with its custodian on first touch (an identity-service
  behavior the node neither performs nor observes); the first publish signs
  and verifies like every later one.
- A principal admitted to the realm but lacking the signing grant: their
  publish fails loudly with the signing failure as the cause — never an
  unsigned operation — exactly the seam semantics shipped in 017.
- An MCP session held open across an admission refusal: tool calls return
  errors naming the authentication cause; the MCP session itself is not the
  trust boundary — admission is — so recovery needs no session teardown.
- Node overload or realm connection churn: per-principal service degrades
  to errors, not to another principal's capacity; eviction of a dead
  realm connection is per-principal and re-admission is retried on that
  principal's next authenticated request.

## Clarifications

### Session 2026-08-02

- Q: Does the feature's primary success criterion require a live hosted
  client and a live authorization server, or is a scripted conforming
  client plus an AS stand-in the gate? → A: The scripted flow is the gate
  (SC-001, SC-005); the live hosted-client-with-live-AS run is a documented
  follow-up measurement, mirroring where 017 placed its live cross-service
  proof. (Folded into SC-001 and Assumptions.)
- Q: Is the realm-side deployment wiring (admission-edge scope template,
  managed-cloud account plumbing) a productized command of this feature or
  documentation plus carried tooling? → A: Documentation plus the
  research-era script carried as operator tooling — explicitly not a
  product surface of this feature; cloud-specific parts are best-effort.
  (Folded into Assumptions.)
- Q: May a forged routing hint degrade the imitated principal's service
  (weak reading of "a forged hint just builds a connection the edge
  refuses"), or must the node guarantee non-interference? → A:
  Non-interference: only a token that has actually been admitted as
  principal P may govern P's shared realm access; a token that never
  admitted must never displace any principal's freshest token or evict
  their access. (Folded into FR-005 and Edge Cases.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The node MUST be reachable at a single URL and offer, to an
  admitted session, the same tool surface the locally installed MCP client
  offers for the realm it fronts — the node adds reach, never capability,
  and nothing about it is protocol-normative.
- **FR-002**: The node MUST hold no credentials, no key material, and no
  per-user secrets, durably or beyond a session's lifetime: the only
  durable configuration is its addresses (bind and public), the realm it
  serves, and the marker that routes its realm connections to the
  admission edge. Restart MUST be free: no state migration, and clients
  recover by re-presenting their current token.
- **FR-003**: Every session's realm access MUST be authenticated by the
  caller's own bearer token passed through unmodified to the realm's
  admission edge. The node MUST NOT mint, transform, or persist tokens; it
  MAY retain, in memory only, the newest token a session has presented, and
  that token MUST be what re-admission uses.
- **FR-004**: Who a session acts as MUST be asserted by the realm after
  admission, never taken from client-supplied claims. The persona is the
  admitted principal's user identity. Any identity material the node reads
  from a token before admission MUST be used for routing only and MUST
  have no bearing on trust, attribution, or access.
- **FR-005**: A request MUST only ever be served over realm access admitted
  for the principal of the token that request's session presented. Sessions
  of the same principal MAY share realm access; sessions of different
  principals MUST NOT. A forged or colliding routing hint MUST NOT grant
  access, displace an admitted principal's service, or deny service to the
  principal it imitates: only a token that has been admitted as a principal
  may govern that principal's shared realm access — a token that never
  admitted MUST NOT become any principal's freshest token or cause eviction
  of their access.
- **FR-006**: Presenting a fresh valid token MUST be sufficient to keep a
  session serviceable across token lifetimes and across the node's internal
  reconnections to the realm, with no other refresh mechanism required of
  the client beyond re-authenticating when challenged.
- **FR-007**: Revocation MUST take effect through re-admission: once a
  principal's access is revoked upstream, requests MUST begin failing no
  later than the realm edge's admission window allows, and the node MUST
  NOT extend service beyond what its current admitted access permits.
  Refusals MUST be non-sticky: a subsequent request with a valid token MUST
  be re-admitted, with failed realm access evicted rather than reused.
- **FR-008**: To unauthenticated or invalidly authenticated requests, the
  node MUST respond with the standard protected-resource challenge and
  MUST publish standard discovery metadata sufficient for a conforming
  client to locate the authorization server and complete sign-in —
  including clients that register themselves automatically — with no
  out-of-band configuration.
- **FR-009**: The node MUST NOT be an authorization server: it issues no
  tokens, renders no login or consent surface, and holds no authorization-
  server credential. Exactly one authorization lane exists: an external
  authorization server conforming to the stated contract.
- **FR-010**: The feature MUST state the AS-facing contract precisely and
  AS-agnostically: the discovery relationship, how clients register
  (automatic registration or pre-registered), the sign-in flow class
  (authorization code with proof-of-possession challenge), the claims an
  issued token must carry for admission — including that the persona
  identifier must be a legal persona name (lowercase slug, at most 64
  characters) — and what happens on each conformance failure. A minimal
  conforming stand-in built from the contract alone MUST be exercised in
  the feature's tests; no part of the node may special-case any particular
  authorization server.
- **FR-011**: Operations published through the node MUST be signed as the
  acting persona via delegated signing — key custody stays with the
  identity service; keys never reach the node — and MUST verify identically
  on every existing read surface. Signing failure MUST fail the publish
  loudly with its cause; the 017 seam semantics (no unsigned fallback) are
  invariants this feature must not move.
- **FR-012**: The node MUST operate correctly behind an HTTPS front whose
  public name differs from its bound address, as an explicitly declared
  deployment shape; fronted deployment is the documented norm.
- **FR-013**: Token material MUST NOT appear in the node's logs, error
  messages, durable state, or diagnostics; admissions, refusals, and
  evictions MUST be observable per principal by the operator.
- **FR-014**: The node MUST be built as a separate consumer module: the
  soulstream library and the identity service's client are both imported by
  the node, and neither core library gains a dependency on the other or on
  the node — the cycle guard holds.

### Key Entities

- **The node**: a shared, stateless door into one realm — an MCP server
  reached by URL, holding configuration but no trust. Everything it serves
  rides access the realm's edge admitted for the caller.
- **Bearer token (the badge)**: the caller's proof, issued by the external
  authorization server, presented over HTTP, passed through to the realm's
  edge unchanged. Short-lived by design; the newest one presented is the
  one that counts.
- **Principal**: the identity the realm's edge asserts after admitting a
  token; the persona is the principal's user identity. The only identity
  the node ever acts on.
- **Routing hint**: identity material peeked from a token *before*
  admission, used solely to group sessions likely to be the same principal.
  Explicitly untrusted; correctness never depends on it.
- **Admission edge**: the realm-side decision point (the identity service's
  auth callout) that validates tokens and asserts principals. Owns trust,
  admission windows, and revocation; the node owns none of them.
- **Authorization server (external)**: the operator-chosen issuer the
  hosted client authenticates against. Conforms to the stated contract;
  the intended first one is the operator's own, but nothing in the node
  knows which one it is.
- **AS-facing contract**: the published statement of what a conforming
  authorization server must provide — discovery, registration, flow, and
  claim requirements — precise enough to build against without reading
  node internals.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a client with nothing installed, a person can go from
  knowing only the node's URL to a working realm session — discovery,
  sign-in, admitted — and their first published operation verifies as
  their persona on every existing read surface. The gate is this flow
  driven by a scripted conforming client end to end; the same flow through
  a live hosted client against a live authorization server is a documented
  follow-up measurement, not a gate.
- **SC-002**: With multiple principals active concurrently through one
  node, 100% of operations are attributed to their true author, and an
  adversarial run presenting forged identity hints yields zero requests
  served with another principal's access and zero disruption to the
  imitated principal's sessions.
- **SC-003**: A session presenting periodic fresh tokens remains fully
  serviceable across at least three token lifetimes with zero failed
  operations attributable to expiry; a revoked principal's requests begin
  failing within the admission window after revocation; a re-authorized
  principal is served on their next request.
- **SC-004**: After serving a multi-principal workload and being killed
  and restarted, the node's durable footprint contains zero tokens, zero
  keys, and zero per-user secrets, and every client recovers with at most
  a re-presented token.
- **SC-005**: A minimal authorization-server stand-in written from the
  AS-facing contract document alone completes the full flow (discovery,
  automatic client registration, sign-in, admitted session) against the
  node with no node-side changes — demonstrating the contract, not any
  particular server, is the interface.
- **SC-006**: An audit of logs, errors, and diagnostics from a full test
  run finds zero occurrences of token material.

## Assumptions

- The identity-service side is shipped and out of scope: the admission
  edge (auth callout, token validation, admission windows), first-touch
  persona key materialisation, the public-key directory, and the delegated
  signer adapter all exist and were proven cross-service; this feature
  consumes them. Changes there (e.g. an issuer claim-name profile, a
  persona display-name surface) are that project's decisions.
- The intended authorization server (the operator's own, in progress) is
  built against this feature's AS-facing contract; live pairing with it is
  a follow-up measurement, not a gate here. The gate is the conforming
  stand-in (SC-005) — chosen deliberately so the contract stays
  AS-agnostic and the in-progress server does not block the door.
- Realm-side deployment wiring (the admission edge's scope template, the
  managed-cloud account plumbing) is an operator prerequisite documented
  by this feature, with the research-era setup script carried as operator
  tooling; it is explicitly not a product surface of this feature, and its
  cloud-specific parts are best-effort.
- One node instance fronts one realm. Fronting several realms is running
  several nodes — acceptable at this maturity and consistent with
  "configuration you can reason about at a glance".
- "Same tool surface as the locally installed client" means parity with
  the MCP tool set as it exists when this feature lands; the node does not
  fork the tool surface.
- Hosted-client behavior (OAuth-only connector dialogs, discovery
  expectations, automatic registration) is as measured 2026-08-01; if a
  hosted client later grows other lanes, that is new information for a
  future decision, not this one.
- Display names for OIDC-born personas are out of scope (an identity-
  service decision noted in the design); boards may render bare persona
  identifiers meanwhile.

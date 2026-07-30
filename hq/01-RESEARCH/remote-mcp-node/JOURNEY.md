# remote-mcp-node — investigation journal (started 2026-07-30)

## 2026-07-30 — the prototype and Bars 1–3, in one sitting

**Setup.** Built the prototype node and its rig in [experiment/](experiment/)
(a consumer-position module importing both repos; soulstream at the working
tree ≈ v0.6.0, soulidentity at the working tree, post-journey-0016). The rig
is the one combination neither repo had ever run: **admission through auth
callout** (the Entra e2e's front door) **with the both-subject-space scope
template** (the m2 gate's shape) — SoulIdentity user ops beside
`SOULSTREAM.>`/JetStream, plus `$SYS.REQ.USER.INFO` — on one team account
(ACME) hosting both the SoulIdentity service and the realm. The node itself
is ~250 lines: `mcp.NewStreamableHTTPHandler` with a per-request hook, a pool
keyed by an *unverified* routing hint (OIDC `iss`+`oid` peek, else the token
string), `nats.TokenHandler` reading the freshest bearer stored by the HTTP
layer, principal → `PersonaSigner` → `realm.NewClient` per user.

**Bar 1 — PASS.** Both lanes admit through the node with attribution read
back from the realm, not the node's word: the API-token session's turn is
authored `daan-ext`, the Entra-shaped session's turn is authored by the oid.
Garbage and revoked tokens refuse (MCP session never forms, `callout REFUSED`
in the audit) and the topic's contribution count is unchanged — no realm
write without admission.

- **Finding (Bar 1 wording amended openly): the sentinel is required in
  operator mode.** The bare token option is refused
  (`nats: Authorization Violation` with no callout decision); the sentinel
  creds file (soulidentity journey 0008: a public, deny-all bearer artifact)
  is the rung that routes a connection to the callout. So the node's honest
  config is **URL + realm + sentinel** — the sentinel is credential-*shaped*
  but grants nothing by itself. Bar 1's "nothing credential-shaped" is
  amended to "nothing that grants access by itself"; measured basis recorded
  here.
- **Finding: OIDC-lane personas are oids.** The issuer names the minted user
  `sub.OID`, so realm attribution reads as a UUID (a legal soulstream slug);
  the human display (`preferred_username`) survives only in the audit line.
  Fine for machines, poor for humans reading a board — makes the
  persona-presentation op on SoulIdentity's surface (display metadata beside
  `keys.public`) load-bearing, exactly the gap named in the pre-registration.
- **Finding: the principal is server-asserted, discoverable by the node.**
  `$SYS.REQ.USER.INFO` returns the *resolved* permission set; the expanded
  `{{account-subject()}}.{{name()}}` in the `sign.record` grant names
  (account, user) — a whoami with no client-claimed identity anywhere. It is
  a parse of a permission subject, though; an explicit whoami surface is an
  018 niceness.

**Bar 2 — PASS.** The negative control first: with no keyring the node's
announcement reads `unknown-key`. Then the reader builds its keyring from
**one** `keys.public` answer and announce + turns read `SigVerified`. Zero
per-user acts anywhere; the persona key materialised in the vault at
`PersonaSigner` construction; the node's process held URL + realm + sentinel
and no key material.

**Bar 3 — PASS, after the investigation's best scare.**

- *Phase 1 (expiry, refreshed badges):* on a 5 s-TTL rig, **33/33 writes over
  17 s with zero failures** across three `authentication expired`
  disconnects. The client refreshed its bearer per request; the HTTP layer
  stored it; `TokenHandler` presented it on each reconnect; nats.go's
  reconnect buffering absorbed even the in-flight window. Better than the
  registered allowance (≤1 in-flight failure per re-proof).
- *Phase 2 (revocation at the IdP):* after stripping the role from fresh
  tokens, **last successful write +3.1 s, persistent failure from +3.9 s**
  (bound: 2×TTL = 10 s). Mechanism, from the audit: at the first TTL
  boundary the reconnect presents the stripped token, callout refuses
  (`no roles claim`), nats.go refuses twice and **closes the connection
  permanently** — signing stops dead.
- *The scare:* two earlier runs showed writes "succeeding" 20–30 s
  post-strip. The full audit refuted the test, not the design: `sign.record`
  entries stopped at the refusal, while the test loop counted `err == nil`
  as success — **MCP tool errors are in-band** (`result.IsError`), and 54
  tool-level failures on a dead connection had been logged as "ok". Lesson
  for every MCP harness we write: a transport-level success is not a call
  success.
- *Design note for 018:* after nats.go's two-strikes close, the prototype's
  pool entry is a permanent corpse. A real node must evict dead entries and
  rebuild on the next authenticated request (and answer 401, not 400, while
  refused).

**Where the bars stand:** 1–3 PASS [measured, rig in experiment/]. **Bar 4
(a genuinely no-install client) remains** — it needs the node exposed over
HTTPS and a claude.ai / Claude Desktop connector pointed at it, which is an
operator act. The reversal-condition question (does the deployment class the
realms actually live on — NGS/Synadia Cloud — host auth callout?) also
remains open; nothing measured today touches it.

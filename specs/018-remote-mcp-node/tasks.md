# Tasks: The Remote MCP Node

**Input**: Design documents from `/specs/018-remote-mcp-node/`
**Prerequisites**: plan.md, spec.md (5 stories, Clarifications 2026-08-02),
research.md R1–R11, data-model.md, contracts/{library,authorization-server,http}.md

**Tests**: included — the spec's success criteria are explicitly
test-asserted (SC-002/004/005/006 name their audits), and the constitution's
gate is `make check` green with none skipped.

**Organization**: by user story. US1 is the MVP slice; US2 is a P1
constraint on US1's mechanics and lands immediately after. The prototype at
`git show 56c7a2e:hq/01-RESEARCH/remote-mcp-node/experiment/` is reference
material for T004, T010–T012, T034–T035 — carry mechanics, re-derive nothing
(research R3), EXCEPT the trust model, which is new (R4).

## Phase 1: Setup

**Purpose**: the nested module exists and the repo's gates cover it

- [ ] T001 Create `node/` module skeleton: `node/go.mod` (module
      `github.com/impire-io/soulstream/node`, Go 1.26, committed
      `replace github.com/impire-io/soulstream => ../`, soulidentity by
      pseudo-version @ ≥5eaf52c, go-sdk v1.6.1, nats.go v1.52; test deps
      nats-server/v2, jwt/v2, nkeys), `node/doc.go` (package doc naming the
      custody stance + cycle-guard rule), run `go mod tidy` in `node/`
- [ ] T002 Extend root `Makefile` so `make fmt`/`test`/`lint`/`build` also
      run in `node/` (second-module loop; `bin/soulstream-node` from
      `make build`), keeping single-command `make check` green locally
- [ ] T003 [P] Confirm `.golangci.yml` applies in `node/` (or add
      `node/.golangci.yml` mirroring root) and `.gitignore` covers node
      build artifacts

## Phase 2: Foundational (blocking all stories)

**Purpose**: the public surface exists and a callout-admitting rig runs in-process

- [ ] T004 Promote `internal/mcpserver` → public `mcpserver/`: `git mv`,
      fix the one import in `cmd/soulstream-mcp/main.go`, move in-package
      tests, keep `internal/natstest` import (legal: same module); core
      `make check` green, zero behavior change
- [ ] T005 Add options to `mcpserver.NewServer(c *realm.Client, opts
      ...Option)`: `WithKeyring(func(context.Context) (*identity.Keyring,
      error))` per contracts/library.md §1 — default (no option) keeps the
      file-backed pins path byte-for-byte; `keyring()` consults the
      provider; PLUS the one deliberate surface addition (analysis I1):
      `soulstream_whoami` tool (persona, realm, signer public key — lets a
      remote user see who the edge admitted them as); option +
      default-parity + whoami tests in `mcpserver/server_test.go`
- [ ] T006 Build the node test rig `node/rigtest/rig.go`: embedded
      operator-mode nats-server with auth callout (AUTH + app account
      ceremony via jwt/nkeys — reference `rig_test.go` at 56c7a2e, rebuilt
      on PUBLIC surfaces), the represented-user scope template from
      research R7, soulidentity `embed.Run` in-process (vault + token KV,
      callout issuer, declared role), helpers: `MintStaticToken`,
      `MintOIDCToken(claims)` (RS256 + JWKS served for the validator),
      `StopCallout`, TTL knob
- [ ] T007 Node config: `node/config.go` — `Config` struct + validation
      with teaching errors per contracts/library.md §2 (PublicURL ⇒
      AuthIssuer; Realm/NATSURL required; Prefix default), unit tests
      `node/config_test.go`

**Checkpoint**: `mcpserver` public, rig admits a static token end to end

## Phase 3: User Story 1 — Connect by URL, participate as yourself (P1) 🎯 MVP

**Goal**: a scripted no-install client goes URL → admitted session → full
tool surface → signed, verified first operation (SC-001 gate half)

**Independent Test**: rig + streamable-HTTP MCP client with a valid token:
initialize → tools/list (== stdio surface) → board → start_topic →
post_turn; independent reader verifies `SigVerified`, `Author == persona`

- [ ] T008 [US1] `node/principal.go`: derive `(account, persona)` from
      `$SYS.REQ.USER.INFO` resolved publish allows (5-part
      `<prefix>.<account>.<persona>.sign.record`); pure parse fn + unit
      tests in `node/principal_test.go` (multi-grant, no-grant, malformed)
- [ ] T009 [US1] `node/pool.go` build path: pool map + entry lifecycle
      (data-model states), `nats.Connect` with `TokenHandler` reading
      `latest`, `ReconnectWait(200ms)`, `MaxReconnects(-1)`, optional
      sentinel creds; then principal → `siclient.New(...).PersonaSigner` →
      `realm.NewClient` (assign Signer only when non-nil — 017 guard) →
      `mcpserver.NewServer(rc, WithKeyring(in-memory per-realm keyring))`;
      corpse eviction on build error / `IsClosed()`
- [ ] T010 [US1] `node/node.go`: `New(cfg)`, `getServer(r)` factory
      (bearer extraction, hint peek `oidc:<iss>:<oid|sub>` else
      `tok:<token>`, pool resolve), `mcp.NewStreamableHTTPHandler(...,
      DisableLocalhostProtection: PublicURL != "")`, `Handler()`, `Close()`
- [ ] T011 [US1] `node/oauth.go`: RFC 9728 metadata route + 401
      `WWW-Authenticate` challenge exactly per contracts/http.md (public
      mode only; local mode = SDK 400); unit tests `node/oauth_test.go`
- [ ] T012 [US1] `node/cmd/soulstream-node/main.go`: flags/env mirroring
      Config, `-version`, startup teaching checks (realm stream exists —
      read-only, R9; never provisions), serve `Handler()`
- [ ] T013 [US1] Integration test `node/node_test.go`: the Independent
      Test above on the rig (static token lane), including tool-surface
      parity assert (node `tools/list` == `mcpserver.NewServer` over
      stdio-built list — FR-001), whoami returning the server-asserted
      persona, realm-side verification through an independent reader
      (negative control → keyring → `SigVerified`), and the loud-signing
      surface (analysis C1): a rig user admitted WITHOUT the sign.record
      grant gets a teaching failure — never an unsigned publish (FR-011)
- [ ] T014 [US1] No/invalid bearer tests: public mode → 401 + challenge
      steering (scenario 3); local mode → 400; nothing reaches the realm
- [ ] T015 [P] [US1] Docs: create `docs/remote-node.md` ELI5 core ("a
      front desk that holds no keys — your badge does the talking"): what
      the door is, badge passthrough, who decides identity, first-touch key

**Checkpoint**: MVP — a scripted client joins the realm through the node

## Phase 4: User Story 2 — One shared door, many people, nothing held (P1)

**Goal**: multi-principal isolation + the R4 trust model (non-interference)

**Independent Test**: two principals concurrently, correct attribution;
adversarial forged-hint run yields zero adoption/eviction/displacement

- [ ] T016 [US2] Session binding in `node/pool.go` + `node/node.go`: bind
      MCP session ↔ entry at admission; only bound-session requests update
      that entry's `latest` (the refresh path); unbound request with
      `bearer == latest` binds to the live entry
- [ ] T017 [US2] Candidate probe in `node/pool.go`: unbound request with a
      differing bearer on a live entry → short-lived probe connect;
      admitted + same principal ⇒ adopt+bind; admitted + different
      principal ⇒ serve via that principal's own entry (rekey), victim
      untouched; refused ⇒ 401, entry untouched — full R4/data-model state
      machine
- [ ] T018 [US2] Structured observability in `node/pool.go`: log
      build/refuse/evict/probe events carrying principal + hint CLASS +
      cause; never token material (FR-013)
- [ ] T019 [US2] Multi-principal test in `node/pool_test.go`: two
      personas through one node, interleaved posts, every op attributed +
      signed correctly (SC-002 first half)
- [ ] T020 [US2] Adversarial suite in `node/adversarial_test.go`: forged
      JWT carrying victim's iss/oid (garbage sig, wrong sig, expired),
      attacker's own valid token under victim's hint → assert zero
      `latest` adoption, zero eviction, victim session uninterrupted, all
      forged requests 401 (SC-002 second half)
- [ ] T021 [US2] Custody audit test in `node/custody_test.go`: run a
      multi-principal workload with HOME/XDG redirected to a temp dir —
      assert the node wrote NO files; `Close()` + rebuild recovers with
      re-presented bearers only (SC-004 first half)
- [ ] T022 [US2] Token-material log audit in `node/logaudit_test.go`:
      capture all node logs across T019+T020 runs, grep for every token
      value minted by the rig → zero hits (SC-006)
- [ ] T023 [P] [US2] Docs: `docs/remote-node.md` custody section (what
      the node never holds; forged-badge story in plain words)

**Checkpoint**: the shared door is provably safe for many people

## Phase 5: User Story 3 — Sessions outlive tokens; revocation lands (P2)

**Goal**: refresh across TTLs, revocation within the window, non-sticky refusals

**Independent Test**: short-TTL rig: session survives 3× TTL with fresh
bearers; revoked principal refused within window; re-authorized on return

- [ ] T024 [US3] Refresh test in `node/refresh_test.go`: TTL ~2s, client
      re-presents fresh tokens via its bound session, continuous writes
      across ≥3 TTLs, zero expiry-attributable failures (SC-003, Bar-3
      re-run through the full tool surface)
- [ ] T025 [US3] Revocation tests in `node/revoke_test.go`: delete token
      record / strip declared role → next admission refuses (≤ window);
      live pooled conn dies by TTL and is evicted as a corpse; returning
      with a valid token re-admits on that request (non-sticky, US3
      scenario 3)
- [ ] T026 [US3] Edge-outage test in `node/refresh_test.go`: `StopCallout`
      → requests fail with the auth challenge, nothing queued; restart →
      first request served (edge case list)
- [ ] T027 [P] [US3] Docs: `docs/remote-node.md` badge-lifetime section
      (TTL sizing = revocation responsiveness, in plain words)

## Phase 6: User Story 4 — Any conforming authorization server (P2)

**Goal**: the AS contract is the interface — proven by a stand-in built from it

**Independent Test**: scripted client completes 9728 discovery → AS
discovery → DCR → PKCE sign-in → admitted session against the stand-in;
conformance violations refuse exactly as the contract states

- [ ] T028 [US4] AS stand-in `node/rigtest/asstub.go` written FROM
      `contracts/authorization-server.md` ALONE (note provenance in the
      file doc — SC-005's condition): OIDC discovery doc, JWKS (RS256),
      DCR (RFC 7591), authorization-code + PKCE (S256) endpoints, token
      endpoint stamping the fixed audience
- [ ] T029 [US4] Full-flow test `node/oidc_flow_test.go`: scripted client
      knows only the node URL → challenge → metadata → AS → DCR → PKCE →
      token → admitted MCP session → signed post verified by independent
      reader (SC-001 full + SC-005); JWKS rotation mid-run needs no
      restarts (contract §3)
- [ ] T030 [US4] Conformance-refusal tests in `node/oidc_flow_test.go`:
      illegal `oid` slug, missing `oid`, zero-match / ambiguous `roles`,
      wrong `aud`, wrong issuer, non-RS256 alg → all refused at admission,
      401 at the node, nothing published (contract §3 table)
- [ ] T031 [P] [US4] Docs: `docs/remote-node.md` sign-in section + the
      operator binding checklist (issuer/audience/role knobs, contract §6)
      cross-linked from `docs/operators.md`

## Phase 7: User Story 5 — An operator runs the node without fear (P3)

**Goal**: fronted deployment, free restarts, carried tooling

**Independent Test**: node behind a public-name front serves correctly;
kill/restart mid-traffic recovers with only a re-presented bearer

- [ ] T032 [US5] Proxy-fronted test in `node/node_test.go`: requests with
      the public Host against a loopback bind succeed when `PublicURL`
      declared, are guarded when not (FR-012)
- [ ] T033 [US5] Restart test in `node/custody_test.go`: kill mid-traffic,
      restart, client's next request with its current bearer is re-admitted
      and continues; durable footprint still empty (SC-004 second half)
- [ ] T034 [P] [US5] Carry `node/cmd/byon-setup` from 56c7a2e (Synadia
      Cloud callout wiring; dry-run default, `--apply`, XKey surfaced
      loudly; secrets 0600) — compile + dry-run plan test; mark
      best-effort in its README header (spec Q2)
- [ ] T035 [P] [US5] Carry `node/cmd/probe` from 56c7a2e (pass protocol +
      independent realm verification) — the follow-up measurement driver
      (spec Q1); compile test
- [ ] T036 [P] [US5] Docs: `docs/remote-node.md` operator half (front
      with HTTPS, sentinel, scope template prerequisite, restart-is-free)
      + `quickstart.md` cross-check

## Phase 8: Polish & Cross-Cutting

- [ ] T037 Cycle-guard + custody asserts: test that core `go.mod` gained
      no dependencies (esp. no soulidentity), `node/go.mod` imports both,
      `mcpserver` package imports no `node/` code — mirror of 017's
      measured rule (FR-014)
- [ ] T038 [P] Update `docs/mcp.md` (the two doors: local adapter vs
      remote node) and `README.md` component list; docs index links
- [ ] T039 [P] Extend `.goreleaser.yaml` + release workflow: build
      `soulstream-node` from `node/` (goreleaser `dir` builds work from
      checkout despite the replace), snapshot-verify locally; extend
      `.github/workflows/ci.yml` with the node module job (`GOPRIVATE`,
      documented org-credential secret — flag for Daan if secret absent)
- [ ] T040 Full gate: `make check` green (both modules, all tests, none
      skipped, lint 0), quickstart walked end to end against the rig

## Dependencies

```
Setup (T001–T003) → Foundational (T004–T007)
  → US1 (T008–T015)                    🎯 MVP
      → US2 (T016–T023)               (constrains US1 mechanics; same files)
          → US3 (T024–T027)           (needs binding/probe for refresh tests)
          → US4 (T028–T031)           (needs 401/metadata + pool; parallel with US3)
              → US5 (T032–T036)       (fronted/restart tests ride full flows)
                  → Polish (T037–T040)
```

US3 and US4 are independent of each other and can proceed in parallel once
US2 lands. Within phases, [P]-marked tasks touch disjoint files.

## Parallel opportunities

- T004/T005 (core module) alongside T006/T007 (node module) — different modules
- US1: T008 ∥ T011 ∥ T015; T012 after T007
- US2: T019/T020/T021/T022 are separate test files once T016–T017 land
- US3 ∥ US4 wholesale; T034/T035/T036 within US5

## Implementation strategy

MVP = Phase 1–3 (US1 on the static lane): a scripted client joins a realm
through the node with a valid token — demonstrable value, everything real
except OAuth dressing. US2 immediately hardens the same code paths (it is a
P1 constraint, not an add-on — do not ship US1 without it). Then US3/US4 in
parallel, US5, polish. Each checkpoint keeps `make check` green; commit per
task group as usual.

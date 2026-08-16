# Tasks: BYO NATS (010)

- [x] T001 ceremony: extract the persona scope template into one source
      (`personaScope`) shared by Generate and the kit (SC-004)
- [x] T002 ceremony: `GenerateBYO` — local material per flavour; new State
      fields (flavour, url, synadia system, issuer-user seed)
- [x] T003 ceremony: `MintBYOUsers` — signing-key-signed bypass users with
      `IssuerAccount`; issuer creds from the persisted issuer-user seed
- [x] T004 ceremony: state.go — `byo` config block, BYO branches in
      files()/Save/Load/Verify, `ArtifactCount` becomes a State method,
      mutual-exclusion and validity refusals
- [x] T005 ceremony: kit.go — `Kit(st, dir)`: nsc commands (validated against
      real nsc), config fragments, push, hand-back
- [x] T006 ceremony tests: BYO round-trip, kit content, refusals
- [x] T007 node: Start's BYO branch (no server, url = substrate), Stop guard
- [x] T008 node: byo.go — `ProbeSubstrate` (named refusals incl. conf-auth)
      and `SmokeAdmission` (sentinel+token admits, garbage refuses)
- [x] T009 node tests: the rig plays the operator — external operator-mode
      server, kit publics → account JWTs, full founding, M1.1 semantics,
      custody audit, partial-kit and conf-auth refusals
- [x] T010 internal/synadia: the control-plane driver (ported from
      byon-setup, idempotent by name, plain workload group added) + stub test
- [x] T011 cmd: init two-phase flow, BYO flags, up/workload URL branches,
      usage text; cmd tests
- [x] T012 docs: getting-started BYO section replaces "deliberately not here
      yet"; quickstart.md with the manual Synadia runbook
- [x] T013 gate: make fmt && make test && make lint green; signed commits

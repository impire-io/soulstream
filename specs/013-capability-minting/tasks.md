# Tasks: Capability minting in the house

**Input**: spec.md, plan.md in this directory.

- [x] T001 ceremony.go: `AgentRole`, State fields, step-4 agent key + scoped
      signer from `client.AgentScope*`; persona scope adopted from
      `client.PersonaScope*` (FR-001, FR-007).
- [x] T002 state.go: `fileAgentSigning`, files() conditional entry,
      load-if-present in both flavours, verifyEmbedded scoped assertion
      (FR-002).
- [x] T003 byo.go: self-hosted phase 1 generates the agent key; kit.go:
      agent nsc lines + `AgentScopeAllows` (FR-005).
- [x] T004 node/minter.go `capabilityMinter`; node/workload.go routing +
      named preflight refusals (FR-003, FR-004).
- [x] T005 node/capability_test.go: TestM14CapabilityAgent (done/abandoned
      arms), legacy-realm refusal, wrong-role refusal (SC-001, SC-002).
- [x] T006 `make check` green; pin-bump standing exception recorded.

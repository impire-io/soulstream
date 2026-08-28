# Tasks: The thinking house

**Input**: spec.md, plan.md in this directory.

- [ ] T001 ceremony: State fields for both planes, `MintPlaneUser`, the two
      config blocks with their defaults and named Verify refusals
      (FR-001, FR-002, FR-003).
- [ ] T002 node/catalogue.go: the realm KV catalogue — ensure create-or-report,
      put, list, resolve to a `client.Descriptor` (FR-009).
- [ ] T003 node/inferenceplane.go: the door on its own listener, the
      per-serve key registry, `Route` through the catalogue, the configured
      instances with their custody-resolved keys (FR-007, FR-008).
- [ ] T004 node/dispatcherplane.go: the plane identity, the placement topic
      create-or-report, `ConnectAgent` (persona-scope mint), `EngineFor`
      (preset + tool door + the inference lane), drain (FR-004..FR-006, FR-011).
- [ ] T005 node/node.go: start both planes in order (inference before
      dispatcher), the accessors, the stop ceremony.
- [ ] T006 cmd/soulstream: `agent submit`, `model set|ls`, `provider set`,
      usage and endpoint lines (FR-010).
- [ ] T007 node/thinkinghouse_test.go: TestM15ThinkingHouse (answer-once,
      custody scan, keyless door, restart resume), the no-inference arm, the
      disabled arm, the configuration refusals (SC-001..SC-004).
- [ ] T008 `make check` green; the new gates at `-race -count=3`; pin bump
      recorded as the standing exception.
</content>

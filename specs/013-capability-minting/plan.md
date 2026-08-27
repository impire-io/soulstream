# Implementation Plan: Capability minting in the house

**Branch**: `013-capability-minting` | **Date**: 2026-08-27 | **Spec**: [spec.md](spec.md)

## Summary

The founding grows the agent capability key (a scoped signer on the realm
account, template = `client.AgentScope*`), the state machinery persists and
verifies it load-if-present, the self-hosted kit endorses it, and the
runtime plane routes capability-bearing declarations through workloads'
`ScopedSigningKeyMinter` — with named refusals for legacy realms and
foreign role names. No identity vault change (the RoleForAccount finding,
spec.md). The D47 persona-scope adoption rides along: the ceremony's local
scope copy becomes calls into `client`.

## Constitution Check

- **Composition of public surface — PASS**: everything wired is upstream
  public API (`minter.ScopedSigningKeyMinter`, `client.AgentScope*`).
- **Clean breaks pre-v1 — PASS**: old realms run unchanged; only the new
  capability path refuses, by name, with the migration in the refusal.
- **Secrets in the state dir — PASS**: one more 0600 seed file, same
  custody story as the plain workload key.
- **Gates all green — `make check`**, TestM14 in the hermetic suite.

## Project Structure

```text
specs/013-capability-minting/   # spec.md, plan.md, tasks.md
ceremony/
├── ceremony.go     # State.AgentSigningSeed/Pub, AgentRole, step-4 key +
│                   #   scoped signer; persona scope from client (D47 adoption)
├── state.go        # fileAgentSigning; files() conditional; load-if-present
│                   #   both flavours; verifyEmbedded scoped assertion
├── byo.go          # self-hosted phase 1 generates the agent key
├── kit.go          # agent scoped-signer nsc lines; AgentScopeAllows
node/
├── workload.go     # preflight refusals; the routed minter
├── minter.go       # NEW: capabilityMinter{scoped, plain}
└── capability_test.go  # NEW: TestM14CapabilityAgent + refusal tests
```

## Complexity Tracking

No violations. The Synadia BYON fourth signing-key group is a named [O]
(needs a live run — the 0136 discipline), not silent scope.

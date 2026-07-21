# Cross-Artifact Analysis: 008-discover

**Date**: 2026-07-21 | **Scope**: spec ↔ plan ↔ tasks ↔ constitution

## Findings

No critical/high findings. Constitution: the mechanism is core NATS request-reply
with a subject convention — no registry, broker, queue group, or persistence
(FR-008 is also a sweep item in T013); matching/merging stay pure; docs are in-story
(T008, T012).

| ID | Severity | Note | Resolution |
|----|----------|------|------------|
| S1 | MEDIUM | The realm stream (`SOULSTREAM.>`) captures discovery *requests* (replies ride `_INBOX.*`, never captured). Stored requests are unread clutter. | Documented and accepted in contracts/wire.md: trivial volume, limits retention, and excluding `SVC.>` would be an in-place stream mutation provisioning refuses (one-way door). Revisit only if volume matters. |
| C1 | LOW | Reply signatures bind to `DISCOVER`, not the reply inbox — a reply is therefore not cryptographically bound to a *specific request*. A recorded reply could in principle be replayed to a different asker. | Accepted at this trust level: replies are ephemeral advice, not durable ops; the asker's own inbox subscription scopes delivery in practice. Request-binding (op-id in the signed reply) is a curator/memory-era hardening, noted for later. |
| L1 | LOW | `Discover` returns `(nil, nil)` on silence — callers must not treat nil as an error sentinel. | Contract and tests pin the semantics (SC-003); both clients render the friendly empty-result message. |
| L2 | LOW | CLI `respond` needs a cancellable long-running test. | The watch/inbox tests already established the pattern (context-cancelled `Run`); T011 mirrors it. |

## Coverage

| Requirement | Tasks | | Requirement | Tasks |
|---|---|---|---|---|
| FR-001 | T004, T005 | | FR-006 | T009, T010 |
| FR-002 | T003, T004, T005 | | FR-007 | T001, T002, T004, T009 |
| FR-003 | T004, T005, T010 | | FR-008 | T013 (sweep) + design |
| FR-004 | T009, T010, T011 | | FR-009 | T006, T007, T011 |
| FR-005 | T003 | | FR-010 | T004, T005 |

SC-001→T005/T010 · SC-002→T005 · SC-003→T005/T010 · SC-004→T005 · SC-005→T010 ·
SC-006→suite at every checkpoint. Both stories close with docs tasks.

## Metrics

- Requirements: 10 FR + 6 SC · Tasks: 13 · Coverage: 100% · Critical: 0 · High: 0 ·
  Medium: 1 (documented/accepted).

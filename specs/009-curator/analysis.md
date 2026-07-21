# Cross-Artifact Analysis: 009-curator

**Date**: 2026-07-21 | **Scope**: spec ↔ plan ↔ tasks ↔ constitution

## Findings

No critical/high findings. Constitution: nothing new to run or keep consistent (the
whole feature is one optional client process); smallest-viable honoured by cutting
digests and any persistence; docs in-story (T008, T013). The extension's own
design test — "everything a curator does must be something the protocol already
works without" — is enforced structurally (separate package on public surfaces) and
swept in T014.

| ID | Severity | Note | Resolution |
|----|----------|------|------------|
| D1 | MEDIUM | The dirty-marking core subscription receives only messages published while the curator runs; `Board`-seeded state covers history. A message lost between seed and subscribe (startup race) or during a hiccup would leave a stale cache entry. | Accepted with a belt: the scan tick also refreshes all dirty paths, and answering refreshes before matching; staleness can delay an answer's coverage by at most one tick, never corrupt. Ordering fixed in T004/T005: subscribe first, then seed. |
| D2 | LOW | Two curators may double-flag in the same instant (both check, both post). | Spec makes this explicit: at-most-rarely-twice, harmless, self-limiting; not worth coordination machinery the design forbids. |
| D3 | LOW | Dormancy relies on author-claimed timestamps. | By design: judgment producing a suggestion, not protocol ordering — spec/research say so; a lying clock earns at worst a premature polite comment. |
| D4 | LOW | Duplicate flags skip closed topics (T009) — a nuance the spec doesn't state (it says "newer topic" without lifecycle). | Deliberate tightening consistent with US2's intent (a resting topic needs no redirect); noted here rather than spec-churned. |
| D5 | LOW | `BaselineTs` changes the rollup round-trip comparison (post-rollup baseline time ≠ birth time). | T001 handles it explicitly in the equivalence strip, alongside stream_seq/sig — a legitimate, documented divergence. |

## Coverage

| Requirement | Tasks | | Requirement | Tasks |
|---|---|---|---|---|
| FR-001 | T005, T006, T014 (sweep) | | FR-006 | T009–T012 |
| FR-002 | T004, T006 | | FR-007 | T009, T011 (comments only) + T014 |
| FR-003 | T002, T005, T006 | | FR-008 | T003, T010 |
| FR-004 | T003, T009, T010 | | FR-009 | T009, T011, T012 |
| FR-005 | T011, T012 | | FR-010 | T007, T013 |

SC-001→T006 · SC-002→T010 · SC-003→T012 · SC-004→T006 + full suite · SC-005→T010.
All three stories close with docs tasks.

## Metrics

- Requirements: 10 FR + 5 SC · Tasks: 14 · Coverage: 100% · Critical: 0 · High: 0 ·
  Medium: 1 (mitigated by design in tasks).

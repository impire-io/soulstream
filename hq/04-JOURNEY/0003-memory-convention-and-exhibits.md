# Episode 0003 — The memory convention: the realm learns to be asked (2026-07-25)

The substrate forgets by design — rollup physically removes op tails — and the
roadmap had named the standing danger for months: *retention is not retrofittable*,
and every compaction on a realm whose history matters (ours, since the NGS realm went
live) widened a permanent op-granularity blind spot. Feature `015-memory` (Day-2 #8)
closed the convention half of that gap in one autonomous spec-kit cycle: collective
search as scatter/gather testimony (`memory.query` / `memory.answer` / `memory.fetch`
/ `memory.exhibit` on `SOULSTREAM.SVC.MEMORY`, the 008 discovery triad reused
verbatim), portable **exhibits** (an op's wire form captured byte-for-byte plus realm
and canonical binding — self-authenticating anywhere, verdicts being the existing
`SigStatus` set), and asker-side **citation grading** that checks rather than trusts:
fact / fact-with-provenance / testimony / gossip / unverifiable. 33 new tests, 0
skipped, lint 0, `make check` green throughout [measured]. The end-to-end criterion
holds: an op physically destroyed by rollup came back from a witness as a verifying
exhibit (SC-004) [measured], and any amount of memory traffic adds zero retained
messages to the realm's stores, by construction since 014 left `SVC.>` uncaptured
(SC-006) [measured].

**The load-bearing direction call: the archivist lives in a separate repository**
under the impire-io organisation (owner decision, 2026-07-25). This repo ships the
convention and the public witness surface only — `RespondMemory` with independently
optional Answer/Fetch capabilities and a declared `coverage_from` — and NO store, no
index, no ranking, no archive role. The sufficiency of that contract is not asserted
but proven mechanically: SC-005's test lives in an external `topic_test` package,
compiling against exported identifiers only, and plays the archivist end to end
(keep-while-live → serve → recall-after-compaction) [measured]. The cleanest proof
that an outsider can build the archivist is that its first consumer *is* an outsider.

Two things the build refuted or sharpened. First, the plan's own wire examples had
the canonical binding wrong (`OPS.<path>`); reading `canonicalBinding` showed topic
ops bind to the bare path — caught and corrected before any code relied on it
[measured]. Second, a subtlety the tests forced into the open: after rollup, *most*
op-ids still resolve — baked contributions, attachments, work items, and even the
frontier keep their ids — so "compacted" does not mean "unciteable". What genuinely
vanishes are the interior marks (resolve/remove, transitions, superseded interiors),
and exactly those are what exhibits exist for [mechanism-argument]. The grading
distinction (fact vs unverifiable) traces that boundary precisely.

What it opened: the archivist repository is now unblocked, with a
measured-not-promised contract to build against; and the one-way door the roadmap
warned about is finally instrumented — a realm that starts a keeper today has
op-granularity memory from today.

Reversal condition: the separate-repo call reverses if the archivist turns out to
need private library surfaces after all (an SC-005-shaped build failure in the other
repo), or if operating two repos for one dogfood deployment demonstrably costs more
than it isolates — either reading would justify folding the archivist back in-tree.
The no-inline-exhibits clarification reverses if real queries routinely need a
second round-trip for provenance on most answers (observed, not anticipated), which
would justify witnesses attaching exhibits inline under a size cap.

Trail: `specs/015-memory/` (spec, research D1–D9, data-model, contracts, tasks —
all 27 tasks done), `hq/02-DESIGN/extensions/memory.md` (the settled design this
implements), `docs/memory.md` + `docs/exhibits.md` (ELI5); commits: the
`015-memory` branch series from `spec(015-memory)` through
`feat(015-memory): CLI memory command group, MCP memory tools, ELI5 docs`.

# Quickstart: Rollup & Archive

A long-running topic, compacted and eventually shelved. Assumes a provisioned realm
(006 quickstart).

## 1. A topic grows

```console
$ soulstream show design-review-x7m2 | tail -1
  [4e9910ac] ✓ architect (turn): …the 100th contribution…
```

A cold read replays 100+ messages. Time to save a version.

## 2. Compact it

```console
$ soulstream rollup design-review-x7m2
compacted design-review-x7m2: 103 ops folded into a new baseline
```

The log is now **one message**. Reading it shows the identical conversation:

```console
$ soulstream show design-review-x7m2
topic:     design-review-x7m2
name:      Design Review
lifecycle: active
contributions:
  [9f86d081] ✓ daan (turn): Ready for a look.
  …everything, exactly as before…
```

Baked contributions carry the compactor's attestation: they show the **baseline's**
verification status (the tail's individual signatures were destroyed with the tail —
that is what compaction is).

## 3. Life goes on across the boundary

```console
$ soulstream post design-review-x7m2 "continuing after the rollup"
$ soulstream comment design-review-x7m2 9f86d081 "still anchors to a baked turn"
```

New ops parent onto the baseline's frontier; anchors to baked op-ids still resolve.

## 4. Losing the race is fine

If someone posts in the instant between your read and your compaction:

```console
$ soulstream rollup design-review-x7m2
soulstream: topic: rollup lost the race (someone wrote concurrently); try again
$ soulstream rollup design-review-x7m2
compacted design-review-x7m2: 105 ops folded into a new baseline
```

Nothing was lost either time — first writer wins, the loser's attempt evaporates.

## 5. Big states still fit

When the materialised state outgrows 128 KB, the baseline becomes a manifest: the
state lives in the object store, the log still holds exactly one message, and readers
reconstruct it transparently (digest-checked).

## 6. Close tidies; archive is final

```console
$ soulstream close design-review-x7m2      # close now compacts too (best effort)

$ soulstream archive design-review-x7m2
archived design-review-x7m2: final baseline published; the topic is now read-only
$ soulstream post design-review-x7m2 "one more thing?"
soulstream: topic: design-review-x7m2 is archived — archived is terminal, writes are refused
$ soulstream show design-review-x7m2       # reading works forever
lifecycle: archived
```

Closed warns; archived refuses. That asymmetry is deliberate: closing is social,
archiving is the realm's one reclamation act.

## 7. What an agent can do

The MCP adapter exposes `soulstream_rollup_topic` — agents may tidy the workbenches
they use. Archival is not a tool: shelving a notebook for good is an operator's call.

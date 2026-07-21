# Quickstart: The Curator Persona

## 1. Start a curator

Any persona, ordinary credentials — it's a habit, not a role:

```console
$ SOULSTREAM_PERSONA=curator soulstream curate --idle 336h --scan-every 1m
curating as "curator" (Ctrl-C to stop)
projection ready: 12 topics
```

Run none and everything still works. Run two and they cooperate by reading each
other's comments.

## 2. Discovery gets better

A phrase that only ever appeared *inside* a conversation now finds its topic:

```console
$ soulstream discover "rollup gate"
active    design-review-x7m2                       Design Review
          answered by: curator ✓
```

The plain responder (`soulstream respond`) can't see inside topics; the curator can.
Both may run — the asker merges, and the curator is simply the best answerer in the
room, never the only one.

## 3. Polite sticky notes

Start a near-duplicate of an existing topic and the curator leaves one comment in
the newer one:

```console
$ soulstream show q2-vat-filing-k3f9 | grep curator
  [ab12cd34] ✓ curator (comment -> 9f86d081): [curator] this looks similar to vat-q2-x7m2 — consider continuing there
```

Let a topic sit past the idle window and it gets one nudge:

```console
  [77aa01c3] ✓ curator (comment -> 08c3fe12): [curator] no activity for 14 days — close it if it's done
```

That's the curator's entire vocabulary of action: comments. It never closes,
archives, merges, or compacts anything on its own — you can ignore every suggestion
it ever makes, and it won't repeat itself (the log is its memory: one flag per
topic, one proposal per quiet spell, across restarts and across curators).

## 4. Stop it

Ctrl-C. Nothing to deregister, nothing degrades — discovery falls back to 008
behaviour, the board keeps working, and every suggestion it made remains an
ordinary, attributed comment in the topics it touched.

# What is the curator?

A busy workshop eventually wants a **librarian** — someone who knows every shelf,
including what's written *inside* the books, answers questions fastest, and leaves
polite sticky notes when something looks off.

That's the curator: **an ordinary persona with tidy habits**. Not a boss, not a
service, not a piece of machinery. It logs in with ordinary credentials, writes
ordinary pages with its ordinary seal, and holds no special power whatsoever. Fire
the librarian and the library still works — that sentence is the whole design.

## What it actually does

**It answers the shout best.** When someone shouts "anyone seen a topic about X?"
([discovery](./discovery.md)), most answerers check the notice board. The curator has
also *read the books*: a phrase that only ever appeared inside a conversation still
finds its topic. Same shout, same kind of answer, just better — and never the only
one; anyone else may answer too, and you hear everyone.

**It leaves two kinds of sticky note.** Both are ordinary comments anyone can see,
argue with, or ignore, and both start with a visible marker so you always know a
suggestion from conversation:

- *"`[curator] this looks similar to <topic> — consider continuing there`"* — pinned
  once in the newer of two look-alike topics. The likeness rule is simple enough to
  say aloud: half their announcement words in common.
- *"`[curator] no activity for 14 days — close it if it's done`"* — pinned once per
  quiet spell in a topic that's gone silent past the workshop's idle window. Fresh
  work resets the clock; the curator's own notes never count as "activity", so its
  chatter can't keep a topic alive or nag twice.

## What it never does

It never closes, archives, merges, or compacts anything on its own. **Comments are
its entire vocabulary of action** — unless you hand it two specific chores below.
Deciding is yours: agree with a sticky note and close the topic yourself, or peel
the note off with a reply and carry on.

## Two opt-in chores (housekeeping, not judgment)

Both are **off unless you say so**, and both are rules any persona could apply by
hand — the curator just does the rounds:

- `--mark-dormant` — topics idle past the window get the ordinary "napping" page
  ([lifecycle](./lifecycle.md)). Arithmetic, not opinion: newest page older than
  the window. Any new writing wakes the topic by itself.
- `--reclaim 168h` — a claimed chore whose owner has gone silent past that window
  gets an ordinary "magnet off" op ([work items](./work-items.md)): the chore
  reopens for the next taker, and the reclaim is signed by the curator like any
  op. Closing and archiving still belong to people.

## Why it never nags

The curator keeps no private notebook of what it's said — **the topic itself is its
memory**. Before leaving a note it reads the topic; if the note is already there (its
own from before a restart, or another curator's), it stays quiet. That's also how two
curators cooperate without ever speaking: they read each other exactly the way they
read everyone.

## Running one

```sh
soulstream curate                 # under whatever persona you give it
soulstream curate --idle 72h      # a snappier workshop
soulstream curate --mark-dormant --reclaim 168h   # with the housekeeping chores
```

Run none — everything works, discovery just answers from plainer notes. Run two —
they cooperate. Ctrl-C — it's gone, nothing to unregister, and every note it left
remains an ordinary signed comment in the topics it touched.

## Related

- [Discovery](./discovery.md) — the shout the curator answers best.
- [Lifecycle](./lifecycle.md) — closing and archiving, which stay *your* decision.
- [Signing](./signing.md) — the seal on the curator's notes, like anyone's.

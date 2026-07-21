# Work items: the chore chart

Think of a fridge door with a **chore chart**: anyone can write a chore on it, and
next to each chore there's room for one name magnet. Whoever gets their magnet on
first has the chore. Not the loudest, not the boss's favourite — the first magnet.
And the chart keeps a pencil note of everyone who *tried*, so there's never an
argument about what happened.

That chart is what **work items** add to a topic: tasks with a status, an owner,
and a full history — written in the same notebook as everything else.

## The four moves

- **open** — write a chore on the chart (a title, maybe a description; `@name`
  in the description taps someone on the shoulder, like any
  [mention](./mentions.md)).
- **claim** — put your magnet on it.
- **done** — tick it off. Ticked chores stay ticked (want it redone? open a new
  chore).
- **abandon** — take your magnet off; the chore is up for grabs again. (A magnet
  whose owner has gone silent for ages can be lifted by the household sweep too —
  see the [curator's](./curator.md) opt-in `--reclaim` chore. Same op, same chart,
  signed by whoever did the lifting.)

## Who wins a race?

The notebook decides. Everything in a topic lands on one page after another, in
one order everyone sees the same way. If two people claim the same chore at the
same moment, whichever claim **landed first in the notebook wins**; the later one
is written down as **void** — visible, honest, changing nothing. The loser finds
out right away: the chart already shows the owner's name. There is no referee, no
lock, no "ask the server nicely" — the order of the pages *is* the referee.

## A task is a conversation

Evidence and discussion live where they always do: comments and
[attachments](./attachments.md) pinned to the chore. Looking at one work item
shows its story — who opened it, who tried to claim it, who did it, and what they
produced.

## Small, honest details

- Anyone can tick a chore or lift a magnet — the chart records *who did it*, in
  plain sight, the same way anyone can close a topic. Attribution, not locks, is
  the house rule.
- A claim on a ticked or already-claimed chore is simply void. A scribbled,
  unreadable chore entry is skipped with a note. Nothing ever jams the chart.
- Closing a topic doesn't touch its chart; archiving freezes the whole notebook,
  chart included.
- Tidying day ([rollup](./rollup.md)) keeps the chart exactly as it was — voids,
  owners, ticks and all.

## Try it

```sh
soulstream work open my-topic "water the plants" --body "@scribe yours?"
soulstream work claim my-topic <item-id>     # → claimed — you own it
soulstream work claim my-topic <item-id>     # (as someone else) → void — owned by …
soulstream work list my-topic
soulstream work done my-topic <item-id>
soulstream work show my-topic <item-id>      # the full story, voids included
```

## Related

- [Artefacts](./artefacts.md) (the documents chores produce) ·
  [The topic](./topic.md) (the notebook it's all written in) ·
  [Lifecycle](./lifecycle.md) (closing and archiving around the chart).

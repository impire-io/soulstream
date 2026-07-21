# A topic's life: proposed, active, dormant, closed, archived

A topic has a little **life story**, and you can tell which chapter it's in just by
reading the notebook — nobody has to keep the status in a separate place.

- **Proposed** — someone started it, but no real work has happened yet. It's just the
  cover page and a blank first page. "Here's an idea; anyone want to dig in?"
- **Active** — as soon as the first real page is written (a turn, a comment), the topic
  is being worked on.
- **Dormant** — the topic is **napping**: nothing has been written for longer than the
  realm's idle window, and somebody noted that down. Not finished, not locked — just
  visibly asleep. The moment *anyone writes anything*, it's active again, no ceremony.
- **Closed** — someone writes a "we're done here" page. The topic stays fully readable,
  but by convention people stop adding to it. Closing also tidies the notebook
  ([rollup](./rollup.md)) so it rests as one neat page.
- **Archived** — the notebook is **bound and shelved**: one final, thorough tidy glues
  everything into a single last page, and then the covers are sewn shut. Read it
  forever; write in it never.

## Nobody's in charge of the status

You don't ask a server "what state is this topic in?" — you *read the pages and work it
out*, and everyone works out the same answer. Closing is just another page with your
name on it.

That has a nice side effect: if two people both write "we're done" at the same moment,
there's **no clash to resolve**. Two "closed" pages say the same thing; the topic is
closed either way. No voting, no referee.

## Closing is a handshake, not a lock

Closing doesn't slam the notebook shut. If someone writes on a closed topic anyway, the
library gives a gentle **warning** ("hey, this one's closed") but still lets the page
through — because whether a closed topic should really re-open is a *human* call, made in
the topic, not something the plumbing should forbid.

## Archiving is a lock — the only one

Archiving is different on purpose. It's the realm's **one deliberate reclamation
act**: someone explicitly says "shelve this notebook for good"
(`soulstream archive <path>`), the final tidy runs, and from then on every attempt to
write gets a firm, clear **no** — not a warning. Reading works forever; the shelf is a
library, not a shredder.

That asymmetry is the point: closing is social, archiving is final. And because it's a
one-way act, nothing ever archives a topic on your behalf — no timer, no housekeeping
robot. A person (or an agent's operator) says so, or it doesn't happen.

## Who marks a topic dormant?

Anyone — because it isn't a judgment, it's arithmetic: "the newest page is older
than the window." Check the rule, write the page:

```sh
soulstream mark-dormant <path>          # posts only if the topic really is idle
soulstream curate --mark-dormant        # or let a curator do the rounds (opt-in)
```

Two people marking the same topic dormant at once is as harmless as two "closed"
pages: same state, no clash. And a napping topic never blocks anything — you can
still close it, archive it, or just start writing and wake it.

## Related

- [The topic](./topic.md) · [Materialisation](./materialisation.md) · [Rollup](./rollup.md)

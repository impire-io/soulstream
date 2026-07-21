# What is the persona directory?

Every workshop has a **phone book** by the door: one card per persona. A card says who
someone is — their name, a friendly display name, whether a human or an agent is behind
it, who operates the agent — and, most importantly, it shows **a picture of their wax
seal** (their public signing key, see [signing](./signing.md)).

The phone book is how everyone else learns what your seal looks like, so they can check
the letters you sealed.

## Publishing your card

You write your own card (`soulstream profile publish`), and only you can — the doorman
checks. The first time is a fresh card. Later you may re-write the friendly bits
(display name, description) whenever you like. But the **seal picture on your card
never changes silently** — replacing a seal has its own careful ritual (below).

The card also says *human or agent* — but that is purely for display, like a little
glyph next to a name. Nothing in the workshop treats anyone differently because of it.
That rule is what "peers" means here, and it is deliberate.

## Remembering seals (pinning)

Here's the clever part. Readers don't just look at the phone book every time and
believe it — the book's keeper could swap a card. Instead, the **first** time you see
someone's seal, you copy it into **your own pocket notebook** (the pin file, kept on
your machine).

From then on you trust your notebook first. It's like knowing a friend's handwriting:
after the first letter, you don't ask the post office whether it's really them — you
*recognise* them.

## When the phone book disagrees with your notebook

- The card shows the seal you remember → fine.
- The card shows the old seal **plus** a proper hand-over note to a new one → fine,
  that's a rotation; you add the new seal to your notebook.
- The card shows a **different seal with no hand-over note** → alarm bells. Someone may
  have swapped the card to impersonate your friend. Soulstream shouts about it
  (`!! possible key substitution`), stops trusting that persona's seals, and keeps your
  notebook unchanged as evidence. It never quietly starts trusting the new seal.

## The hand-over note (rotation)

When a persona replaces its seal, it presses the **old** seal on a note naming the
**new** one. The card keeps every hand-over note, so a newcomer can follow the whole
paper trail: first seal → note → second seal → note → third. If any link in that trail
is broken, the whole card is treated as suspect.

Old letters stay good: a letter sealed with any seal in the trail still counts as that
persona's.

## A workshop without a phone book

Older workshops don't have one, and that's fine — nothing breaks. Sealed letters just
show as "sealed by a seal I don't recognise" (`?`) until a directory appears. The
phone book is a helper, never a requirement.

## Related

- [Signing](./signing.md) — the wax seal itself.
- [Provisioning](./provisioning.md) — the directory is the workshop's third fixture.
- [Personas & attribution](./persona-and-attribution.md) — who the cards are about.

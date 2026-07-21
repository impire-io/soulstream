# Editing, replies & resolving: pencil edits and margin notes

A topic's conversation is written in ink — nothing is ever torn out. But three
small tools keep it tidy:

## The pencil edit (crossed out, never torn out)

Fixed a typo? Changed your mind about a sentence? Write the new version over your
old one — everyone now reads the correction, with a little "edited" mark so
nobody is fooled. Flip back through the notebook (the raw log) and the old
version is still there; the page just *shows* the newest.

Two rules keep this honest:

- **Your words are yours.** Only you can edit what you wrote. If someone else
  tries, their attempt lands in the notebook (everything lands), but nothing
  changes and a note says so. Disagreement is a reply, not a rewrite — that's
  what protects the signature on your words.
- **The newest version wins, everywhere.** Edit an edit, or two of your devices
  edit at once: everyone reads the same final version, because everyone reads
  the same notebook in the same order.

## The margin reply

A comment asks a question; the answer belongs *under it*, not at the end of the
notebook. A **reply** is a comment pinned to a comment — a little thread in the
margin. `@name` in a reply taps that person on the shoulder, like anywhere else.

## The "settled" stamp

When a margin thread is done, anyone can stamp it **resolved**: still readable,
visibly closed, out of the way. Stamping twice does nothing; the stamp shows who
pressed it. There is no un-stamp — if it turns out unsettled, reply and say so.

## Try it

```sh
soulstream post my-topic "lets ship thursdy"
soulstream edit my-topic <op-id> "let's ship Thursday"
soulstream reply my-topic <comment-op-id> "the 30th"
soulstream resolve my-topic <comment-op-id>
```

## Related

- [Mentions](./mentions.md) (edits and replies ping too) ·
  [The topic](./topic.md) · [Signing](./signing.md) (why only you can edit you).

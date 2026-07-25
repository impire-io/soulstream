# A sealed note anyone can check (Exhibits)

Imagine someone wrote a note, pressed their **wax seal** on it ([signing](./signing.md)),
and you photocopied the whole thing — seal and all. Years later, anyone holding your
copy can check the seal and know: *this author wrote exactly this, in that notebook,
on that page.* They don't have to trust you, the keeper. The seal does the work.

That's an **exhibit**: one operation, captured exactly as it travelled, in a single
portable document. It carries the op's content, its author, which realm and topic it
was said in, and — when the author signs — the signature over all of it.

## Why exhibits exist

The stream forgets by design: [rollup](./rollup.md) physically removes old ops. An
exhibit is how a moment survives the tidying — not because the realm kept it, but
because *someone* did. Whoever bothered to keep the bytes is a valid keeper; there is
no official archive to trust, and none is needed.

## Checking one

`soulstream memory verify decision.exhibit.json` — no connection, no realm, just the
file and the seals you already know (your [pinned keys](./persona-directory.md)).
The verdict is one of exactly four:

- **verified** — the seal checks out against the author's known keys. This author
  said this, here. Fact, whoever handed you the file.
- **failed** — a seal is present but wrong. The document was altered, or it never
  said this. Changing *any* part — one letter of the content, the author's name, the
  topic it claims — lands here.
- **unsigned** — the op never carried a seal. Perfectly readable, but only as
  trustworthy as whoever kept it.
- **unknown-key** — sealed, but you don't know this author's key. Not a failure;
  check on a machine that knows the realm.

## Getting one

- **While the op is live**: `soulstream memory exhibit <topic> <op-id> -o file` —
  a straight photocopy from the stream.
- **After compaction**: `soulstream memory fetch <topic> <op-id>` — first the stream
  is tried, then the realm's witnesses are asked ([memory](./memory.md)); the first
  copy whose seal verifies wins, and an unsigned copy is accepted only as a
  last-resort, clearly-labelled fallback.

## What an exhibit is not

It is not a bundle. An exhibit of a note that *points* at a file in the cupboard
(an [attachment](./attachments.md), a manifest) proves what the note said — the
pointer, the digest — not the cupboard's contents. Evidence of the record, not a
copy of everything the record mentions.

One honest limit, inherited from [signing](./signing.md): ops written before a
persona had a key are unsigned forever, so their exhibits are testimony-grade
forever. The seal can't be added after the fact — that's the whole point of seals.

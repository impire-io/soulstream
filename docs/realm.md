# What is a realm?

Imagine a **private workshop**. Only the people with a key can get in. Inside, there
is exactly one long **notebook** where everyone writes down what they did, in order,
and one **supply cupboard** for the big or bulky things that don't fit in the
notebook.

A **realm** is that workshop.

- The **key** is your NATS credential. If you have one, you're in; if you don't, the
  workshop doesn't even exist as far as you're concerned.
- The **notebook** is the `SOULSTREAM` stream. Every action anyone takes is written on
  the next empty line. Lines are never torn out one-by-one to save space; instead,
  when a page gets long, someone tidies it into a fresh summary line (that's "rollup",
  which comes later). Nothing important is ever lost to a timer.
- The **supply cupboard** is the `soulstream-objects` object store. Pictures, files,
  and anything too big for a notebook line go in the cupboard, and the notebook just
  writes down "the thing named X is in the cupboard".

One workshop, one notebook, one cupboard. That's the whole building. Everything else
Soulstream can do is just *more kinds of writing in the same notebook*.

## Why only one of each?

Because the whole idea is to keep the "what do I need to run this?" list tiny. One
notebook and one cupboard is the smallest thing that still lets people work together
and keep a real history. If you ever feel you need a second notebook, you probably
want a second workshop (a second realm) instead — they can't see into each other,
which is exactly the privacy you want.

## Related

- [Provisioning](./provisioning.md) — how the empty workshop gets set up.
- [The operation record](./operation-record.md) — what a single line in the notebook
  looks like.

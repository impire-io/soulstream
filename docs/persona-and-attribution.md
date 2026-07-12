# Who is a persona, and how do we trust "who wrote this"?

A **persona** is just a name someone works under — `daan`, `architect`,
`invoice-agent`. Here's the big idea that makes Soulstream different: **a human and an
AI are the same kind of thing.** Both hold a key, both write slips, both sign their own
name. There is no separate "bot door". One protocol, one kind of name.

A name must be a tidy lowercase label — letters, numbers, and hyphens, like
`invoice-agent`. No capitals, no spaces, no dots. (Dots mean something else, and
capitals would trip up the plumbing.) If a name breaks the rules, the library says so
and tells you why.

## Everyone signs their own name — and only their own

Think of a shared logbook where everyone writes in it.

- **When you write (the "From" line):** the library won't let you write someone else's
  name in the "From" box. If your key belongs to `daan`, you can only file slips as
  `daan`. Trying to file one as `architect` is refused on the spot. You can't put words
  in someone else's mouth.

- **When you read:** you check the "From" line. First, is it even a real, well-formed
  name? Always checkable. Second — *is it really them?* That depends on whether there's
  a **doorman** on duty:
  - **No doorman:** you take the name at face value, the way you trust a colleague's
    signature in a shared notebook. The library is honest about this — it doesn't
    pretend to catch a forger it can't see.
  - **Doorman on duty** (the realm runs a strict check at the entrance): the library
    double-checks the name on the slip against who the doorman actually let in, and
    raises a flag if they don't match.

## Why not just let the library "catch all forgers"?

Because in the simple setup, a plain slip doesn't carry proof of who really dropped it
off — so promising to detect every forgery would be a lie. Soulstream would rather be
honest: it guarantees you can't *accidentally or lazily* file under the wrong name, and
it does a real cross-check whenever a doorman (or, later, a wax-seal signature) makes
one possible. It never claims a guarantee it can't keep.

## Related

- [The operation record](./operation-record.md) — the slip with the "From" line.
- [The canonical record](./canonical-record.md) — the standard form a real wax-seal
  signature will one day stamp.

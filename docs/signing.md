# What is signing?

Anyone can photocopy a letter. Anyone can *write* a letter and put your name at the
bottom. So how do you prove, years later, that a letter really came from you?

In the old days you pressed a **wax seal** on it. Everyone could see the seal; only you
owned the stamp that makes it. A sealed letter proves itself — no witness needed, no
matter whose drawer it spent ten years in.

Signing in Soulstream is exactly that. A persona can own a **seal stamp** (a signing
key). It has two halves:

- the **stamp** itself (the secret key) — it never leaves your machine, ever;
- the **picture of your seal** (the public key) — you show that to everyone, so they
  can recognise your pressings.

## What gets sealed?

Not the envelope — the **standard form**. Every operation can be re-typed onto the
[canonical record](./canonical-record.md), the government form that always comes out
character-for-character identical. The seal is pressed on *those* bytes.

That's why the form matters: your seal on my copy and your seal on the archive's copy
match, because the pages are guaranteed identical. And because the form includes which
workshop (realm) and which workbench (topic) the slip belongs to, nobody can steam the
seal off one notebook and glue it into another — the page underneath would read
differently, and the seal would no longer fit.

## Testimony and exhibits

- An **unsigned** slip is *testimony*: "the notebook says daan wrote this." Inside the
  workshop that's fine — the doorman (the connection's credentials) checked who came in.
- A **signed** slip is an *exhibit*: it proves itself anywhere, even outside the
  workshop, even if the person holding it is a stranger you don't trust.

Slips written before a persona had a stamp stay testimony **forever** — you can't go
back and seal last year's letters. That's why Soulstream got sealing as early as
possible: everything from now on can be an exhibit.

## Sealing is optional

No stamp? Everything works exactly as before — your slips are just unsealed. A persona
starts sealing the day it makes a stamp (`soulstream key init`) and from then on every
slip it writes — turns, comments, attachments, announcements, even mention pings — is
sealed automatically. Nobody is forced; nothing breaks.

## Getting a new stamp (rotation)

Stamps wear out, or you worry someone photographed yours. You can switch: press your
**old** seal onto a note that says "this is my new seal". Anyone who knew your old seal
can now trust the new one — the trust hands over like a relay baton, and letters sealed
with the old stamp still count as yours.

A new seal that shows up **without** that hand-over note is treated as an alarm, not an
update — see [the persona directory](./persona-directory.md) for how readers spot that.

## Related

- [The canonical record](./canonical-record.md) — the standard form the seal is
  pressed on.
- [Personas & attribution](./persona-and-attribution.md) — whose name is on the slip.
- [The persona directory](./persona-directory.md) — where seals are shown, remembered,
  and checked. *(arrives with the directory feature)*

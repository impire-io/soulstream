# What is the canonical record?

Sometimes a slip needs to leave the workshop — you quote it as proof, save it in an
archive, or seal it in an envelope. Outside the building, you can't rely on the post
office to vouch for it anymore. So you **re-type the slip onto a standard government
form**.

That standard form is the **canonical record**.

The magic of a standard form is that **any two people who fill it out from the same
facts end up with the exact same page** — same words, same order, same spacing, down to
the last character. It doesn't matter who typed it or what order they thought of the
fields; the form pins all of that down.

## Why does "exactly the same page" matter so much?

Because one day you'll want to put a **wax seal** (a signature) on the slip to prove
who really wrote it. A seal only works if everyone agrees on *what* was sealed. If your
copy and my copy of the slip differ by even one space, my seal won't match your copy
and the proof falls apart.

The canonical record is that agreed-upon page. Soulstream builds it with a public
recipe (a standard called JCS / RFC 8785) that always:

- sorts the fields into the same order,
- writes numbers and text the same way,
- and even tidies the *contents* of the box (the payload) the same way.

So the same operation always canonicalises to the identical bytes — today, and in five
years when something finally signs them.

## What's written on the standard form?

The same facts as the slip, plus two that pin it to its home so it can't be
misfiled:

- **realm** — which workshop it belongs to, and
- **topic** — which workbench.

Add those, and nobody can lift a slip out of one workshop and pretend it came from
another — the sealed page itself would no longer match.

## Not yet: the actual seal

Soulstream can make the standard form today, but it does **not** stamp the wax seal
yet — signing comes later. What matters now is that the form is nailed down, so
whatever signs it later signs *these* exact bytes.

## Related

- [The operation record](./operation-record.md) — the everyday slip this is a tidy
  copy of.

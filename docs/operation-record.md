# What is an operation record?

Every single thing anyone does in Soulstream — say something, comment, attach a file,
save a version — is written down as one **operation record**. Think of it as a
**delivery slip taped to a box**.

- The **label** (the slip) holds all the details: who sent it, when, what kind of
  thing it is, which earlier slips it follows on from, and a tracking number.
- The **box** (the payload) holds only the goods — the actual words or data. No
  details are hidden *inside* the box; everything you need to sort and file it is on
  the label outside.

In Soulstream terms: the label is the NATS message **headers**, and the box is the
message **payload**. Keeping the details on the label means a computer can sort,
route, and de-duplicate slips without ever opening a box.

## What's on the label?

| On the label | What it means |
|---|---|
| **Tracking number** (`Nats-Msg-Id`) | A unique id for this exact slip. |
| **Version** (`Soulstream-Version`) | Which rulebook the slip follows (always `1` for now). |
| **From** (`Soulstream-Author`) | Which persona sent it. |
| **Follows** (`Soulstream-Parents`) | The tracking numbers of the slips this one comes right after. |
| **Kind** (`Soulstream-Type`) | What sort of action it is (a message, a comment, …). |
| **When** (`Soulstream-Ts`) | When the sender says they wrote it (a note, not the referee). |
| **Signature** (`Soulstream-Sig`) | Optional wax seal proving who really wrote it (added later). |

## The clever bit: the tracking number is also the anti-duplicate stamp

The tracking number does double duty. If a slip gets sent twice by mistake — a phone
hiccup, a retry — the post office sees the **same tracking number** and quietly throws
the second copy away. So "send it again to be safe" can never create a duplicate. The
id you make once *is* your safety net.

## Two ways to write the same slip

- **On the box** — the everyday form (headers + payload) that travels through NATS.
- **On a standard form** — a re-typed, tidied version used when a slip needs to leave
  the building and still be trustworthy. That's the [canonical
  record](./canonical-record.md).

Both say exactly the same thing; one is just written so any two people always produce
the identical page.

## Related

- [The canonical record](./canonical-record.md)
- [Personas and attribution](./persona-and-attribution.md)

# Attachments: the shared filing cabinet

Some things are too big or too binary to write on a notebook page — a spreadsheet, a
PDF, an image. So the notebook doesn't hold them. Instead there's a **shared filing
cabinet** next to it, and the notebook keeps a small **claim ticket** pointing at the
right drawer.

When you attach a file:

1. the bytes go into the filing cabinet (the realm's object store), and
2. a claim ticket is written on a notebook page — the file's **name**, where it's filed
   (the drawer), its **size**, its type, and a **fingerprint**.

Anyone reading the topic sees the claim ticket, walks to the cabinet, and pulls out the
exact file.

## The fingerprint (why it matters)

The claim ticket records a **fingerprint** of the file's contents (a digest). When you
fetch the file back, you can re-take its fingerprint and check it against the ticket. If
they match, you *know* you got the exact bytes that were filed — nothing was swapped or
corrupted. The reference is verifiable, not just a name you have to trust.

## Small, useful details

- The file can be **empty** (a zero-byte file is still a file). But a claim ticket with
  **no name** is refused — a nameless attachment helps nobody.
- A ticket can be **pinned to a page** (anchored to an operation): "this file is the
  answer to *that* question." If the page it points at can't be found, the ticket is
  flagged (loose ticket) but never thrown away.
- A ticket **pinned to another ticket** means something specific: "I am the newer
  version of that file." That's how documents grow versions — see
  [artefacts](./artefacts.md).
- Fetching a drawer that was never filled gives a clear "not found", not a crash.

## Withdrawing a file

Filed the wrong thing? Anyone can mark a ticket **withdrawn** (`soulstream
detach`): the ticket stays on its page with "withdrawn by so-and-so" on it, and —
this is the important part — **the file stays in the cabinet** so old pages never
point at nothing. If it was a version of a document, the document's "current
version" quietly falls back to the previous one ([artefacts](./artefacts.md)).

The bytes are actually thrown out exactly once: when the topic is **archived**
([lifecycle](./lifecycle.md)) — the realm's one deliberate reclamation act empties
the withdrawn drawers too. Files never marked withdrawn stay in the cabinet
forever, archived or not.

## Not yet

Locking files up (encryption) and sweeping the whole cabinet come later.

## Related

- [The topic](./topic.md) · [The realm](./realm.md) (the cabinet lives in the workshop).

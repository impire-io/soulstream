# Artefacts: the drawer of dated drawings

Imagine a drawer where you keep versions of one drawing. Each time you redraw it,
you put the **new sheet on top** — you never throw the old ones away. Whoever opens
the drawer sees the newest drawing first, and can flip down through every earlier
version, each one signed and dated by whoever drew it.

That drawer is an **artefact**: one document, kept as a stack of whole-file
versions inside a topic.

## How a version is made

There is no new machinery. A version is just an ordinary attachment (a claim
ticket in the [filing cabinet](./attachments.md)) that is **pinned to the previous
version's ticket**. That pin is the whole trick:

- an attachment pinned to another attachment says "*I am the newer version of
  that*";
- an attachment pinned to a conversation page (or not pinned at all) is just a
  file, which is simply an artefact with one version.

## Who decides which version is current?

Nobody — the notebook does. Everyone reading the topic applies the same rule:
**the version that landed last in the notebook is the current one** ("the tip").
If two people redraw the same version at the same moment, both sheets go in the
drawer, both stay forever, and everyone agrees on the same tip, because everyone
reads the same notebook in the same order. No referee needed.

## Small, useful details

- **Renaming is just a version**: the drawer's label is whatever the newest sheet
  is called.
- Every old version can still be **fetched and fingerprint-checked** — nothing was
  overwritten, only superseded.
- A **withdrawn** version ([detach](./attachments.md)) stays in the drawer with its
  mark, but is never served as current: the tip falls back to the newest version
  still standing. Withdraw every version and the drawer's label comes off the list.
- Two different drawers can end up with the same label. When you ask for a drawer
  by label and two match, you're asked to point at the drawer itself (its first
  sheet's id) instead of guessing.
- Tidying day ([rollup](./rollup.md)) changes nothing here: the drawer looks
  exactly the same before and after.

## Try it

```sh
soulstream attach my-topic ./notes.md            # version 1
soulstream revise my-topic ./notes.md --of notes.md   # version 2, pinned to 1
soulstream artefacts my-topic                    # list the drawers
soulstream artefacts my-topic notes.md           # one drawer's history
soulstream get my-topic --artefact notes.md      # fetch the current version
```

## Related

- [Attachments](./attachments.md) (the filing cabinet) ·
  [Work items](./work-items.md) (tasks that use these documents) ·
  [Rollup](./rollup.md) (why nothing is lost on tidying day).

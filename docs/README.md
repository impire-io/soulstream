# Soulstream — plain-words docs

Every concept in Soulstream, explained simply: one idea per page, an everyday analogy
before any detail. Start at the top and read down — each page builds on the ones above.

## Start here

1. [The realm](./realm.md) — the private workshop: one notebook, one supply cupboard.
2. [Provisioning](./provisioning.md) — setting the workshop up (safe to re-run).
3. [The operation record](./operation-record.md) — a delivery slip: details on the label, goods in the box.
4. [The canonical record](./canonical-record.md) — the slip re-typed on a standard form so copies always match.
5. [Personas & attribution](./persona-and-attribution.md) — everyone signs their own name; humans and agents alike.

## Working on things

6. [The topic](./topic.md) — a shared workbench / a group notebook.
7. [Materialisation](./materialisation.md) — reading the notebook front-to-back (and following live).
8. [Lifecycle](./lifecycle.md) — a topic's life: proposed → active → closed → archived.
9. [Rollup](./rollup.md) — tidying day: the pile of notes becomes one fresh first page.
10. [Sub-topics](./sub-topics.md) — sticky-note threads clipped inside a page.
11. [Discovery](./discovery.md) — the notice board, and the shout across the workshop.
12. [The curator](./curator.md) — the librarian: answers fastest, leaves polite sticky notes, never moves your books.

## Reaching people & carrying files

13. [Mentions](./mentions.md) — tapping someone on the shoulder; the ping waits in their pigeonhole.
14. [Attachments](./attachments.md) — the shared filing cabinet the notebook points to.

## Proving who wrote what

15. [Signing](./signing.md) — the wax seal: anyone can copy a letter, only you can press your seal.
16. [The persona directory](./persona-directory.md) — the phone book of seals; your pocket notebook remembers them.

## The two doors

17. [The `soulstream` CLI](./cli.md) — the remote control, for humans.
18. [The MCP adapter](./mcp.md) — the same doors, for AI agents.

## Going deeper

- The **normative design** (the "what is Soulstream, exactly" spec) lives in
  [../hq/02-DESIGN/](../hq/02-DESIGN/) — core + extensions.
- The **build history** (per-feature spec → plan → tasks → analysis) lives in
  [../specs/](../specs/), one folder per cycle (`001-foundation` … `005-mcp`).
- The **why** behind non-obvious calls: [../hq/00-GENESIS/rationale.md](../hq/00-GENESIS/rationale.md);
  the **build order**: [../hq/03-IMPLEMENTATION/ROADMAP.md](../hq/03-IMPLEMENTATION/ROADMAP.md).

## A note on audience

These pages are deliberately non-technical — a newcomer (human or AI) should be able to
follow them without reading any code. The Go packages carry their own reference
documentation (run `go doc ./...`); this folder is the "why and what", not the "how".

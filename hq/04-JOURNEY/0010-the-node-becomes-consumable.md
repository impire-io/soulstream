# Episode 0010 — The node becomes consumable: the replace drops (2026-08-02)

A small change with a consumability meaning, hours after 018 landed: the
node module's `go.mod` carried `replace github.com/impire-io/soulstream
=> ../` with a same-change-set rationale — the public mcpserver surface
and the node that first embeds it landed together, so the node had to see
the working tree. Correct at landing time; answered the moment v0.7.0 was
tagged, because **the tag is that change-set**. With the replace gone and
`soulstream v0.7.0` pinned, the module is requireable by downstream
compositions — and the first one is already waiting: soulnode's Phase 2
front door wires `node.New(Config)` + `Handler()` as its door plane (its
composition research named this surface the fourth embed ask; the
maintainer directed the incorporation today).

Measured before landing: the node module's full suite — pool, custody,
cycle-guard, refresh, OIDC flow, rig — green against the tag, no
behavioral change [measured, `go test ./...` 16 s]. Day-to-day
consequence, same as soulrealm's episode 0011: co-developing the node
against an unreleased soulstream rides an untracked `go.work`; soulstream
changes reach the node by tag bump. The node module itself remains
untagged (`node/v0.x` is a release act for another day); consumers pin a
pseudo-version until then.

Refuted/reversed: nothing — the replace was right for its landing and the
era ended the same day, which is what a tag is for.

Reversal condition: lockstep co-development where tag-bumping measurably
stalls node work (observable: soulstream tags cut solely to unblock node
builds, more than once a week) reopens the replace as a stated temporary
measure with its removal condition attached — soulrealm 0011's condition,
adopted verbatim.

Trail: `node/go.mod`; soulrealm `hq/04-JOURNEY/0011-pinned-to-the-record.md`
(the prior art); soulnode `hq/03-IMPLEMENTATION/roadmap.md` Phase 2 (the
waiting consumer). Commit: this change.

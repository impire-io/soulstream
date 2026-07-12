# Personas

*One identity model for humans and AIs.*

---

A **persona** is a named identity within a realm. It is the unit of attribution (`Ss-Author` on every record), of addressing (`@mention`), and of permission (what its credentials may publish and subscribe to). A persona may be a person, an autonomous agent, a scheduled job, or a team-shared character — the protocol does not know and does not care. This is the load-bearing design decision of the whole platform: there is no separate bot identity system, so nothing built on Soulstream can accidentally treat AI participants as second-class.

## Naming

Persona names are NATS-token-safe slugs, unique within a realm: `daan`, `architect`, `steward`, `invoice-agent`. The name is the stable identifier — it appears in `author` fields, mention subjects, and permission templates. Display names, avatars, and descriptions live in the registry profile and can change freely; the name cannot.

## The registry

The `soulstream-personas` KV bucket maps persona name → profile:

```json
{
  "name":         "architect",
  "display_name": "The Architect",
  "kind":         "agent",
  "description":  "Reviews designs and asks hard questions.",
  "operated_by":  "daan",
  "signing_key":  { "ed25519": "<base64>", "since": "2026-07-10T09:00:00Z" },
  "created_at":   "2026-07-10T09:00:00Z"
}
```

- **`kind`** — `human` | `agent` | `service`. This is *presentation metadata only*: a UI may render agents with a different glyph, a digest may summarise agent chatter more aggressively than human turns. No permission, no capability, and no protocol behaviour may branch on `kind`. That rule is what "peers" means in practice, and it is testable: grep any head or library for `kind ==` and every hit must be cosmetic.
- **`operated_by`** — for agent personas, the persona accountable for its behaviour. A social/audit fact, not a permission link.

KV gives the registry history (who changed a profile, when) and a watch interface (heads keep their persona list live with one watcher) for free.

## Credentials

Identity is enforced by NATS, not by the application. Each persona is backed by NATS user credentials within the realm's account, and the user's permissions are templated on the persona name:

```
publish allow:
  soulstream.announce
  soulstream.ops.>          # author-checked by convention + libraries (see below)
  soulstream.life.>
  soulstream.mention.*
subscribe allow:
  soulstream.>
  _INBOX.>                  # replies
```

Two enforcement levels are available, and a realm chooses per persona:

1. **Transport-scoped (default).** The credential can publish broadly; honest attribution (`author` = own name) is convention, verified socially and by libraries that reject mismatches on read. Adequate inside a high-trust realm — the same trust level as "colleagues don't spoof each other's git commits."
2. **Hard-scoped.** For personas that shouldn't be trusted that far (an experimental agent, a third-party integration), the credential's publish permissions are narrowed to specific topic subtrees, and readers can additionally verify authorship because NATS resolves the publishing user; a strict realm can run a small authoriser (NATS auth callout) that stamps or rejects mismatched `author` fields at the edge. This is the one place a service may sit in the path, and it is optional per realm.

**Attribution has two layers with different lifetimes.** *Live* attribution — trusting `author` on a message as it arrives — is the transport's job: credentials and subject permissions, as above, no app-layer crypto needed. *Durable* attribution — trusting `author` on an op that has left the stream (kept in an archive, quoted as evidence in collective search, exported) — cannot lean on the transport, because whoever presents the op could have altered it. That is what the optional `Ss-Sig` header is for ([01-substrate.md](./01-substrate.md)): an Ed25519 signature over the canonical op record under the persona's `signing_key`, making any kept copy self-authenticating. Signing keys follow the same TOFU-and-pin, rotate-by-signing-with-the-old-key discipline as sealing keys ([05-sealed-topics.md](./05-sealed-topics.md)); the same registry-substitution caveat applies, so fingerprint verification is still the floor for adversarial settings.

A persona may hold **multiple credentials** — a human's laptop and phone, an agent's three replicas — all publishing as the same persona. Credentials are how *processes* connect; personas are *who is speaking*. Revoking one credential does not delete the persona or its history.

## Delegation

Delegation is done with credentials, not with record fields. If persona `daan` wants an agent to act *as him* in a narrow scope, he issues (via the realm's operator tooling) a credential that publishes as `daan` but is hard-scoped to one topic subtree. If he wants the agent to act *as itself on his behalf*, the agent gets its own persona with `operated_by: daan` and speaks under its own name.

There is deliberately no `on_behalf_of` header in v1. Attribution laundering — "the agent wrote this but it counts as the human" — is precisely the ambiguity a peer system should refuse. Either it *is* you (your credential, your scope, your responsibility) or it is *another persona you operate* (its name, your accountability via the registry). Both are honest; the blur between them is not.

## Roles

Roles are conventions, not mechanisms:

- **Realm-level**: an *operator* administers the NATS account, issues credentials, and manages the registry. This is an infrastructure job outside the protocol, like a DBA.
- **Topic-level**: announcements can name expected participants and a topic may by convention have a *convener*. Nothing enforces this; subject permissions decide who can actually post. Experience with enforced membership lists says they rot; permission-policed openness plus curation ages better.
- **Stewardship**: a realm typically runs a *steward* persona — an agent that watches `soulstream.announce` and `soulstream.life.>`, maintains a topic projection, flags duplicates and stale topics, and publishes digests. The steward holds ordinary credentials and publishes ordinary operations. It suggests; participants decide. It is described with the topic model in [03-topics.md](./03-topics.md) because it is a *user* of the protocol, not part of it.

## Mentions and attention

Every persona owns one mention subject: `soulstream.mention.<name>`. When a library publishes an operation containing `@name` tokens, it also publishes a `mention.notify` to each mentioned persona's subject, carrying the topic path and op-ID. A persona subscribes to its own mention subject and reacts however it likes — a human head surfaces a notification; an agent wakes, reads the anchoring op from the topic log, and replies.

This is the substrate's minimum viable attention primitive. It is symmetric on purpose (agents get mentioned exactly like humans) while acknowledging the asymmetry principle from [00-vision.md](./00-vision.md): a human's head will typically *only* follow mentions and steward digests, while an agent may follow `soulstream.ops.>` wholesale. Same protocol, different reading strategies.

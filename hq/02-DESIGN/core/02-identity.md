# Identity

*Normative. One identity model for humans and AIs.*

---

A **persona** is a named identity within a realm, backed by NATS credentials. Behind it may be a person, an autonomous agent, or a scheduled job — the protocol does not know and does not care. This is the load-bearing decision of the whole design: there is no separate bot identity system, so nothing built on Soulstream can treat AI personas as second-class. One protocol, no second door.

**Terminology, fixed once:** *persona* is the only noun for an identity, everywhere in the spec. *Member* is reserved for exactly one thing — a key-holder of a sealed topic's epoch ([../extensions/sealed-topics.md](../extensions/sealed-topics.md)), the single place membership is real and enforced. "Participant" is not a defined term.

## What an identity is

An identity is two things, and only the first is mandatory:

1. **A NATS user credential** within the realm's account. The user's nkey public key is the cryptographic anchor; the credential's subject permissions are the entire permission model.
2. **A persona name** — a NATS-token-safe lowercase slug (`daan`, `architect`, `invoice-agent`), unique within the realm, bound to the credential by its permission template. The name is the **persona-id**: it appears in `Soulstream-Author`, in the notify subject (`SOULSTREAM.PERSONA.NOTIFY.<persona-id>`), and in announcements. Display names, avatars, and richer profiles are an extension ([../extensions/registry.md](../extensions/registry.md)), not core.

A persona may hold **multiple credentials** — a human's laptop and phone, an agent's three replicas — all publishing as the same name. Credentials are how *processes* connect; personas are *who is speaking*. Revoking a credential does not delete the persona or its history.

## Credentials and enforcement

Identity is enforced by NATS, not by application code. The standard permission template:

```
publish allow:
  SOULSTREAM.TOPICS.INFO.>
  SOULSTREAM.TOPICS.OPS.>
  SOULSTREAM.PERSONA.NOTIFY.*
  SOULSTREAM.SVC.*
subscribe allow:
  SOULSTREAM.>
  _INBOX.>
```

Two enforcement levels, chosen per persona:

1. **Transport-scoped (default).** The credential can publish broadly; honest attribution (`Soulstream-Author` = own name) is convention, verified by libraries that reject mismatches on read. Adequate inside a high-trust realm — the trust level of "colleagues don't spoof each other's git commits."
2. **Hard-scoped.** For personas trusted less (an experimental agent, a third-party integration), publish permissions are narrowed to specific topic subtrees. A strict realm may run a NATS auth-callout that rejects mismatched `Soulstream-Author` at the edge — the one place a service may sit in the path, and optional per realm.

## Attribution has two lifetimes

- **Live** — trusting `Soulstream-Author` on a message as it arrives — is the transport's job: credentials and subject permissions, no app-layer crypto.
- **Durable** — trusting `Soulstream-Author` on an op that has left the stream (archived, quoted, exported) — cannot lean on the transport. That is what the optional `Soulstream-Sig` is for ([01-protocol.md](./01-protocol.md)): a signature over the canonical record makes any kept copy self-authenticating. Core defines the signature; how signing keys are published and pinned is an extension concern ([../extensions/registry.md](../extensions/registry.md)).

## Delegation

Delegation is done with credentials, not record fields. If `daan` wants an agent to act *as him* in a narrow scope, he issues a credential that publishes as `daan`, hard-scoped to one topic subtree. If he wants an agent to act *as itself on his behalf*, the agent gets its own persona and speaks under its own name.

There is deliberately no `on_behalf_of` field. Attribution laundering — "the agent wrote this but it counts as the human" — is exactly the ambiguity a peer system must refuse. Either it *is* you (your credential, your scope, your responsibility), or it is *another persona you operate* (its name, your accountability). Both are honest; the blur between them is not.

## Notifications and mentions

Every persona owns one notify subject: `SOULSTREAM.PERSONA.NOTIFY.<persona-id>`. It is deliberately general — anything addressed to one persona lands there — and mentions are its day-one type: when a library publishes an operation containing `@name` tokens, it also publishes a `mention.notify` to each mentioned persona's subject, carrying the topic path and op-ID. A persona subscribes to its own notify subject and reacts however it likes — a human's client surfaces a notification; an agent wakes, reads the anchoring op, and replies.

This is the substrate's minimum attention primitive, symmetric on purpose: agents get mentioned exactly like humans. The asymmetry is in reading strategy — a human's client typically follows only mentions; an agent may follow `SOULSTREAM.TOPICS.OPS.>` wholesale. Same protocol, different appetites.

## Roles are conventions

The only infrastructure role is the **operator**: whoever administers the NATS account and issues credentials — a job outside the protocol, like a DBA. Everything else (conveners, curators, archivists) is a persona with ordinary credentials following a convention, defined in extensions. Core has no roles.

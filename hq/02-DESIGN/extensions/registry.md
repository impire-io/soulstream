# Extension: The Persona Registry

*Optional. Rich profiles, presentation metadata, and key distribution. Core identity ([../core/02-identity.md](../core/02-identity.md)) works without it.*

---

Core identity is a credential plus a name. A realm that wants more — display names, "who answers for this persona," published keys — runs the registry: a `soulstream-personas` KV bucket mapping persona name → profile:

```json
{
  "name":         "architect",
  "display_name": "The Architect",
  "description":  "Reviews designs and asks hard questions.",
  "operated_by":  "daan",
  "operator_attestation": { "operated_key": "<base64>", "sig": "<base64>" },
  "signing_key":  { "ed25519": "<base64>", "since": "2026-07-10T09:00:00Z" },
  "created_at":   "2026-07-10T09:00:00Z"
}
```

KV gives history (who changed a profile, when) and a watch interface (clients keep their persona list live with one watcher) for free. Profile documents are decoded strictly: an unknown field makes the document invalid, loudly.

## No `kind` — a persona is a voice with a key

Profiles deliberately carry **no human/agent/service classification** (an earlier `kind` field was removed in 014). The protocol cannot verify what sort of entity controls a key, so it refuses to record the claim; self-presentation is free-form `description`. The peer principle stays testable: there is no field to branch on at all.

## Operator accountability

`operated_by` names the persona accountable for this one — a social and audit fact, not a permission link. Chains of `operated_by` terminate at a **principal** (a persona with no operator); readers resolve the chain from the directory alone, reporting dangling links and cycles rather than trusting them.

The claim may carry an **operator attestation**: the operator's Ed25519 countersignature over the domain-separated statement `soulstream-operator-attestation\n<operator>\n<operated>\n<operated-key-b64>` (empty key when the operated persona has none). It travels as a portable token from operator to operated persona — profiles stay self-published. Readers report every claim as exactly one of **attested** (countersignature verifies against any key in the operator's validated chain, and the bound key is empty or in the operated persona's own chain), **unverified** (no attestation, or operator has no published key), or **failed** (present but not verifying, or operator distrusted). Failure is surfaced loudly, never hidden.

## Key distribution

The registry is where personas publish long-lived public keys:

- **`signing_key`** (Ed25519) — verifies `Soulstream-Sig` on canonical records, enabling durable attribution and exhibits ([memory.md](./memory.md)).
- **`sealing_key`** (X25519) — for sealed-topic key wrapping ([sealed-topics.md](./sealed-topics.md)).

Libraries pin the first key they see per persona (TOFU) and hard-fail on unannounced changes. Rotation is announced by publishing the new key signed by the old one; anything else is treated as a possible substitution attack and surfaced loudly. The registry is operator-controlled, so against an adversarial operator, out-of-band fingerprint verification is the floor — the substrate can carry fingerprints; only humans (or an external PKI) can verify them.

## Service announcements

Personas offering request-reply services (memory, discovery, curation) may declare them in their profile so clients can render "who offers what":

```json
{ "name": "historian",
  "services": [
    { "kind": "memory", "subject": "SOULSTREAM.SVC.MEMORY",
      "coverage_from": "2026-07-10T00:00:00Z" }
  ] }
```

Advisory, like everything in the registry — scatter/gather works without it.

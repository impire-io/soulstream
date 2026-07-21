# Data Model: Op Signing & Key Distribution

## SigningKey (identity, pure)

A persona's Ed25519 keypair.

| Field | Type | Rules |
|---|---|---|
| seed | 32 bytes | secret; stored client-side only, file mode 0600; never logged/transmitted |
| public | 32 bytes | derived from seed; published as std base64 |

Operations: `Generate()`, `FromSeed(seed)`, `Sign(canonical []byte) → base64 sig`,
plus package-level `Verify(pubB64, canonical, sigB64) → bool` (malformed key/sig ⇒ false,
never panic).

## Keyring (identity, pure)

The verifier's view of every persona's key material, built by clients from directory +
pins, consumed by `topic`'s annotate pass.

| Field | Type | Rules |
|---|---|---|
| keys | map[persona][]base64 pub | the validated chain, oldest → current |
| distrusted | map[persona]bool | true ⇒ substitution suspected ⇒ all sigs `failed` |

Nil keyring is legal everywhere and means "no key knowledge": signed ops report
`unknown-key`.

## Profile (registry)

KV value at key `<persona>` in bucket `soulstream-personas`. JSON:

| Field | Type | Rules |
|---|---|---|
| name | string | slug, must equal the KV key |
| display_name | string | optional, presentation only |
| kind | `human` \| `agent` \| `service` | presentation/audit only — **nothing may branch on it** (FR-005) |
| description | string | optional |
| operated_by | string | optional persona slug; audit fact, not a permission link |
| created_at | RFC 3339 | set on first publish |
| signing_key | `{ed25519: base64, since: RFC 3339}` | optional; the *current* key |
| rotations | `[]{from, to, proof}` (base64 × 3) | ordered; empty for never-rotated |

### Chain derivation & validation (pure)

```
chain(profile):
  if signing_key absent  → []                    (profile without key: ops = unknown-key)
  if rotations empty     → [signing_key.ed25519]
  else:
    root = rotations[0].from
    each rotations[i]: verify proof_i is Ed25519 by from_i over
        "soulstream-key-rotation\n" + name + "\n" + to_i
    each rotations[i>0]: from_i == to_{i-1}      (contiguous)
    rotations[last].to == signing_key.ed25519    (chain ends at current key)
    → [root, to_0, to_1, …]
invalid anywhere → chain invalid → persona distrusted
```

### State transitions

- **First publish** (no entry for the persona): KV `Create`.
- **Publish onto an existing entry**: metadata update via KV `Update(rev)` — stored
  `signing_key`/`rotations` are authoritative and preserved; an incoming *different*
  key is refused ("key conflict; rotate instead"), which is how a second client with a
  different key is stopped from silently replacing a published key.
- **Rotation**: append one `{from: current, to: new, proof}` entry, set `signing_key`
  to the new key, KV `Update(rev)`.

## Pins file (internal/keystore, client-side)

One JSON file per realm: `pins/<realm>.json`.

| Field | Type | Rules |
|---|---|---|
| realm | string | must match the connected realm |
| personas | map[persona][]base64 pub | the validated chain as first seen |

Reconciliation on every directory read, per persona:

| Pinned vs published chain | Outcome |
|---|---|
| no pin yet | validate profile chain internally → pin it (TOFU) |
| pinned is a prefix of published (or equal) | accept; re-pin the longer chain |
| anything else (diverged, shortened, invalid proofs) | distrust persona; sigs `failed`; loud warning |

Pins are only ever extended, never silently replaced. The file is rewritten atomically
after reconciliation.

## SigStatus (topic)

Per-op verification outcome; exactly one of:

| Value | Meaning |
|---|---|
| `unsigned` | no `Soulstream-Sig` header |
| `verified` | sig verifies against a key in the author's validated chain |
| `failed` | sig present but wrong: malformed, no chain key matches, or persona distrusted |
| `unknown-key` | sig present, author has no (known) key: nil keyring, no profile, or profile without signing_key |

Attached as `Sig SigStatus` to `Contribution`, `Attachment`, `Announcement`,
`Notification`. Never affects inclusion, ordering, lifecycle, or frontier — annotation
only (FR-010).

## Relationships

```
SigningKey (secret, client file)             Profile (KV, public)
      │ signs canonical bytes                     │ chain(profile)
      ▼                                           ▼
Record.Signature ──verify──▶ Keyring ◀──reconcile── Pins file (client)
      │                                           
      ▼                                           
SigStatus on Contribution / Attachment / Announcement / Notification
```

# Quickstart: Op Signing & Key Distribution

The dogfood flow, end to end. Assumes a provisioned realm and a NATS context (see
`specs/001-foundation/quickstart.md`).

## 1. Provision picks up the directory

```console
$ soulstream provision
stream        conformant
object_store  conformant
personas      created
```

## 2. The human sets up a signing identity

```console
$ soulstream key init
generated signing key for persona "daan"
public key: 4Zt…= (ed25519)
seed file:  ~/.config/soulstream/keys/acme-daan.ed25519

$ soulstream profile publish --display-name "Daan" --kind human
published profile for "daan" (key 4Zt…=)
```

## 3. An AI persona gets a key from its operator

```console
$ SOULSTREAM_PERSONA=architect soulstream key init
$ SOULSTREAM_PERSONA=architect soulstream profile publish \
    --kind agent --operated-by daan --description "Reviews designs."
```

The MCP session then signs automatically — no adapter flags needed when the key file
is in the default location.

## 4. Signed conversation

```console
$ soulstream post design-review-x7m2 "Ready for a look."
$ soulstream show design-review-x7m2
topic:     design-review-x7m2
name:      Design Review
lifecycle: active
contributions:
  [9f86d081] ✓ daan (turn): Ready for a look.
  [77aa01c3] ✓ architect (turn): Two questions on the rollup gate…
  [4b21aa09] ? historian (turn): (signed, but historian has not published a key)
  [08c3fe12] old-bot (turn): (unsigned — published before signing landed)
```

`✓` verified · `✗` failed · `?` unknown-key · no glyph = unsigned. Nothing is hidden,
whatever its status.

## 5. Tampering is visible

Any altered field — author, body, topic, timestamp — breaks the signature:

```console
$ soulstream show design-review-x7m2
…
  [77aa01c3] ✗ architect (turn): Two questions on the rollup gate…
```

## 6. Rotation keeps history verifiable

```console
$ soulstream key rotate
rotated signing key for "daan": 4Zt…= → 9fQ…=
previous seed kept at ~/.config/soulstream/keys/acme-daan.ed25519.prev
```

Old ops (signed with `4Zt…=`) and new ops (signed with `9fQ…=`) both show `✓` — the
directory carries the proof chain, verifiers extend their pin.

## 7. Substitution is loud

If the directory's key for a persona changes *without* a proof signed by the old key,
every reader that pinned the original refuses it:

```console
$ soulstream show design-review-x7m2
!! possible key substitution for architect — signatures from this persona are not trusted
…
  [77aa01c3] ✗ architect (turn): Two questions on the rollup gate…
```

(The same line is mirrored to stderr, so scripts and agents can catch it.)

## 8. Offline verification (what signing buys)

Export any single signed op (headers + payload). Anyone holding the persona's public
key can recompute the canonical record and verify the signature — no server, no
directory, no trust in whoever kept the copy. That is the difference between testimony
and an exhibit.

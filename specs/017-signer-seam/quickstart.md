# Quickstart: Signer Seam (017)

## Local keys (nothing changes for you)

```go
key, _ := internalOrYourKeystore.LoadKey(path) // *identity.SigningKey
cfg := realm.Config{Realm: "soulstream", Persona: "daan"}
if key != nil {
        cfg.Signer = key // assign only when non-nil — see the field's doc
}
c, err := realm.Connect(ctx, cfg)
```

Same commands, same key files, same signatures. The only visible library
difference: `key.Sign(bytes)` now returns `(string, error)` (error always
nil for a local key).

## Delegating to a custodian

Implement the two-method interface anywhere — soulstream never learns who
signs:

```go
// Your adapter, in your codebase (the consumer binary or the custodian's client):
type vaultSigner struct {
        cl      *soulidentity.Client // speaks sign.record over NATS
        persona string
        pub     string // the persona's published public key, base64
}

func (s vaultSigner) PublicKey() string { return s.pub }

func (s vaultSigner) Sign(canonical []byte) (string, error) {
        return s.cl.SignRecord(s.persona, canonical) // custodian holds the seed
}

cfg := realm.Config{Realm: "soulstream", Persona: "smith", Signer: vaultSigner{…}}
```

Every op the client publishes is now signed by the vault; readers verify it
exactly as a locally signed record — same wire bytes, same verdicts.

## The rules the seam enforces

- **A configured signer that fails, fails the publish.** No unsigned
  fallback, ever. The error names the signing failure — retry or surface it.
- **Empty is failure.** A signer returning an empty signature is treated as
  an error, because an empty signature would silently travel as "unsigned".
- **Responders go silent.** A discovery responder or memory witness whose
  signer fails answers nothing for that request (silence is the protocol's
  "no answer") and reports `-1` through its `onServed` callback so the host
  process can alert.
- **Your implementation owns its deadline.** `Sign` takes no context; bound
  your custodian round trip inside the implementation.
- **Concurrency is on you.** Clients sign from multiple goroutines
  (responders, curators, sessions); implementations must be safe for
  concurrent use.
- **No seeds.** The interface cannot express "give me the key" — surfaces
  that custody seeds (`keystore`, generation) still take the concrete
  `*identity.SigningKey` only.

## Verifying the seam locally (what the tests automate)

1. Wrap a real `SigningKey` in a delegate double; publish a turn; assert the
   signature is byte-identical to signing locally and `SigStatus` is
   verified on materialise/follow/inbox/exhibit/offline-verify.
2. Swap in a failing double; assert the publish errors, the log gained
   nothing, and a responder under the same double answers nothing while
   reporting `-1`.
3. Produce an attestation token and a rotation through delegate doubles;
   assert the existing verification paths accept both.

package identity

// Signer is the capability to sign canonical bytes as one public identity —
// the seam that lets a persona's key live somewhere other than the publishing
// process: a local seed file ([SigningKey], the default), or an external
// custodian that signs on request and never releases the key.
//
// The contract, beyond the two signatures:
//
//   - Implementations MUST be safe for concurrent use. One client signs from
//     many goroutines (responders, curators, concurrent sessions).
//   - Sign MUST return either a non-empty signature or an error — never
//     ("", nil). The publish chokepoint treats an empty signature as a
//     signing failure, because an empty signature would silently travel as
//     "unsigned" (the canonical form omits an empty sig).
//   - Implementations own their own deadlines: Sign takes no context, so a
//     delegated implementation bounds its custodian round trip itself. A
//     configured signer that fails, fails the operation — publishing never
//     falls back to unsigned.
//
// Deliberately, nothing in the contract can express access to secret key
// material: surfaces that custody seeds (key generation, the keystore) keep
// taking the concrete [SigningKey], never this interface.
//
// Go interfaces are satisfied structurally, and that is load-bearing here: a
// custodian's client library does NOT need to import this package — or this
// module — to plug in. Any type with these two methods satisfies Signer at
// the point where a consumer wires the two together. The dependency rule
// that keeps the ecosystem cycle-free: soulstream never imports a custodian,
// a custodian never imports soulstream; consumers (a node, a CLI, a daemon)
// sit above both and do the wiring.
type Signer interface {
	// PublicKey returns the public half of the identity this signer signs
	// as, encoded as standard base64 of the 32 raw Ed25519 public-key
	// bytes — the encoding profiles and pins use.
	PublicKey() string

	// Sign signs the given canonical bytes, returning the signature as
	// standard base64 of the 64 raw Ed25519 signature bytes, or an error
	// when no signature could be produced.
	Sign(canonical []byte) (string, error)
}

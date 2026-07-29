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

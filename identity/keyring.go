package identity

// Keyring is a verifier's view of every persona's published key material: for each
// persona, the validated key chain (oldest first, current key last) and whether the
// persona is distrusted because its published chain diverged from what this verifier
// had pinned (a possible substitution attack).
//
// A nil *Keyring is valid everywhere and means "no key knowledge at all": signed
// operations then verify as unknown-key, never as failures. That degradation is what
// keeps the persona directory an extension rather than a dependency.
type Keyring struct {
	// Keys maps persona name → validated chain of standard-base64 Ed25519 public
	// keys, oldest first. A signature matching any key in the chain counts as the
	// persona's.
	Keys map[string][]string
	// Distrusted marks personas whose published chain did not extend the pinned
	// chain. Their signatures must be reported failed, never silently re-trusted.
	Distrusted map[string]bool
}

// ChainFor returns the validated chain for persona and whether the persona is
// distrusted. It is nil-receiver safe: a nil keyring knows no chains.
func (kr *Keyring) ChainFor(persona string) (chain []string, distrusted bool) {
	if kr == nil {
		return nil, false
	}
	return kr.Keys[persona], kr.Distrusted[persona]
}

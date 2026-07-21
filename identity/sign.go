package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// SeedLen is the exact length, in bytes, of an Ed25519 signing-key seed.
const SeedLen = ed25519.SeedSize

// ErrBadSeed indicates seed material that is not exactly [SeedLen] bytes.
var ErrBadSeed = errors.New("identity: signing-key seed must be exactly 32 bytes")

// SigningKey is a persona's Ed25519 signing identity. The seed is the secret half:
// it stays on the persona's side and must never be published, transmitted, or logged.
// The public half is what gets published to the persona directory.
type SigningKey struct {
	priv ed25519.PrivateKey
}

// GenerateSigningKey creates a fresh signing key from the system's secure randomness.
func GenerateSigningKey() (*SigningKey, error) {
	seed := make([]byte, SeedLen)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("identity: generate signing key: %w", err)
	}
	return &SigningKey{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// SigningKeyFromSeed reconstructs the signing key from its 32-byte seed — the form
// the seed file stores. The same seed always yields the same key.
func SigningKeyFromSeed(seed []byte) (*SigningKey, error) {
	if len(seed) != SeedLen {
		return nil, fmt.Errorf("%w (got %d)", ErrBadSeed, len(seed))
	}
	return &SigningKey{priv: ed25519.NewKeyFromSeed(seed)}, nil
}

// Seed returns a copy of the secret seed. The caller owns guarding it.
func (k *SigningKey) Seed() []byte {
	seed := make([]byte, SeedLen)
	copy(seed, k.priv.Seed())
	return seed
}

// PublicKey returns the public half as standard base64 of the 32 raw key bytes —
// the encoding used in profiles and pins.
func (k *SigningKey) PublicKey() string {
	return base64.StdEncoding.EncodeToString(k.priv.Public().(ed25519.PublicKey))
}

// Sign signs the given canonical record bytes, returning the signature as standard
// base64 of the 64 raw signature bytes — the Soulstream-Sig header value.
func (k *SigningKey) Sign(canonical []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(k.priv, canonical))
}

// VerifySignature reports whether sigB64 is publicKeyB64's valid Ed25519 signature
// over the canonical bytes. Malformed key or signature material verifies as false —
// on the read side a bad encoding is just a bad signature, never a fault.
func VerifySignature(publicKeyB64 string, canonical []byte, sigB64 string) bool {
	pub, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pub), canonical, sig)
}

// RotationProofBytes is the domain-separated statement an old key signs to endorse a
// persona's new key. Binding the persona name in prevents replaying a rotation proof
// onto another persona's profile; the informational `since` timestamp is deliberately
// outside the proof.
func RotationProofBytes(persona, newPublicKeyB64 string) []byte {
	return []byte("soulstream-key-rotation\n" + persona + "\n" + newPublicKeyB64)
}

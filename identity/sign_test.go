package identity

import (
	"bytes"
	"strings"
	"testing"
)

func TestSigningKeyRoundTrip(t *testing.T) {
	k, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}

	seed := k.Seed()
	if len(seed) != SeedLen {
		t.Fatalf("seed length = %d, want %d", len(seed), SeedLen)
	}

	k2, err := SigningKeyFromSeed(seed)
	if err != nil {
		t.Fatalf("SigningKeyFromSeed: %v", err)
	}
	if k.PublicKey() != k2.PublicKey() {
		t.Errorf("same seed produced different public keys: %q vs %q", k.PublicKey(), k2.PublicKey())
	}

	msg := []byte(`{"v":1,"data":"canonical bytes"}`)
	sig := k.Sign(msg)
	if !VerifySignature(k.PublicKey(), msg, sig) {
		t.Error("signature did not verify against its own public key")
	}
}

func TestSigningKeySeedIsACopy(t *testing.T) {
	k, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	seed := k.Seed()
	pub := k.PublicKey()
	for i := range seed {
		seed[i] = 0
	}
	if k.PublicKey() != pub {
		t.Error("mutating the returned seed changed the key")
	}
}

func TestSigningKeyFromSeedRejectsBadLength(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33, 64} {
		if _, err := SigningKeyFromSeed(make([]byte, n)); err == nil {
			t.Errorf("SigningKeyFromSeed accepted a %d-byte seed", n)
		}
	}
}

func TestVerifySignatureRejects(t *testing.T) {
	k, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	other, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	msg := []byte("the canonical bytes")
	sig := k.Sign(msg)

	cases := []struct {
		name string
		pub  string
		msg  []byte
		sig  string
	}{
		{"wrong key", other.PublicKey(), msg, sig},
		{"tampered message", k.PublicKey(), []byte("other bytes"), sig},
		{"malformed sig (not base64)", k.PublicKey(), msg, "%%not-base64%%"},
		{"malformed sig (wrong length)", k.PublicKey(), msg, "c2hvcnQ="},
		{"malformed key (not base64)", "%%not-base64%%", msg, sig},
		{"malformed key (wrong length)", "c2hvcnQ=", msg, sig},
		{"empty key and sig", "", msg, ""},
	}
	for _, c := range cases {
		if VerifySignature(c.pub, c.msg, c.sig) {
			t.Errorf("%s: verified but should not", c.name)
		}
	}
}

func TestRotationProofBytes(t *testing.T) {
	got := RotationProofBytes("architect", "QUJD")
	want := []byte("soulstream-key-rotation\narchitect\nQUJD")
	if !bytes.Equal(got, want) {
		t.Errorf("RotationProofBytes = %q, want %q", got, want)
	}
	// The persona is bound in: the same key material proves nothing for another name.
	if bytes.Equal(RotationProofBytes("architect", "QUJD"), RotationProofBytes("historian", "QUJD")) {
		t.Error("proof bytes do not distinguish personas")
	}
}

func TestRotationProofSignsAndVerifies(t *testing.T) {
	oldKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	newKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}

	proof := oldKey.Sign(RotationProofBytes("daan", newKey.PublicKey()))
	if !VerifySignature(oldKey.PublicKey(), RotationProofBytes("daan", newKey.PublicKey()), proof) {
		t.Error("rotation proof did not verify against the old key")
	}
	if VerifySignature(oldKey.PublicKey(), RotationProofBytes("mallory", newKey.PublicKey()), proof) {
		t.Error("rotation proof verified for a different persona")
	}
}

func TestPublicKeyIsBase64(t *testing.T) {
	k, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	if strings.ContainsAny(k.PublicKey(), " \n\t") {
		t.Errorf("public key %q contains whitespace", k.PublicKey())
	}
}

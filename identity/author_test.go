package identity

import (
	"errors"
	"testing"
)

func TestEnforceAuthor(t *testing.T) {
	if err := EnforceAuthor("daan", "daan"); err != nil {
		t.Errorf("EnforceAuthor(self) = %v, want nil", err)
	}
	if err := EnforceAuthor("daan", "architect"); !errors.Is(err, ErrForeignAuthor) {
		t.Errorf("EnforceAuthor(foreign) = %v, want ErrForeignAuthor", err)
	}
}

type staticResolver struct {
	persona string
	ok      bool
}

func (s staticResolver) DeliveredBy(map[string][]string) (string, bool) {
	return s.persona, s.ok
}

func TestVerifyAuthor(t *testing.T) {
	// No resolver: shape-only. A valid name passes; an invalid name fails.
	if err := VerifyAuthor("daan", nil, nil); err != nil {
		t.Errorf("VerifyAuthor(valid, no resolver) = %v, want nil", err)
	}
	if err := VerifyAuthor("Bad Name", nil, nil); err == nil {
		t.Error("VerifyAuthor(invalid name) = nil, want error")
	}

	// A resolver that cannot determine the identity: still passes (shape-only).
	if err := VerifyAuthor("daan", nil, staticResolver{ok: false}); err != nil {
		t.Errorf("VerifyAuthor(unresolved) = %v, want nil", err)
	}

	// Resolver agrees with the claimed author: passes.
	if err := VerifyAuthor("daan", nil, staticResolver{persona: "daan", ok: true}); err != nil {
		t.Errorf("VerifyAuthor(match) = %v, want nil", err)
	}

	// Resolver disagrees: mismatch flagged.
	if err := VerifyAuthor("daan", nil, staticResolver{persona: "architect", ok: true}); !errors.Is(err, ErrAuthorMismatch) {
		t.Errorf("VerifyAuthor(mismatch) = %v, want ErrAuthorMismatch", err)
	}
}

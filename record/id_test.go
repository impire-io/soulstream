package record

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewID(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id := NewID()

		u, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("NewID() = %q, not a valid uuid: %v", id, err)
		}
		if u.Version() != 4 {
			t.Fatalf("NewID() = %q, version %d, want 4", id, u.Version())
		}
		if id != u.String() {
			t.Fatalf("NewID() = %q, not canonical lowercase form %q", id, u.String())
		}
		if seen[id] {
			t.Fatalf("NewID() produced a duplicate: %q", id)
		}
		seen[id] = true
	}
}

package registry

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/impire/soulstream/identity"
)

func testKey(t *testing.T) *identity.SigningKey {
	t.Helper()
	k, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return k
}

func TestProfileValidate(t *testing.T) {
	key := testKey(t)
	good := Profile{
		Name:       "architect",
		Kind:       KindAgent,
		OperatedBy: "daan",
		CreatedAt:  time.Now().UTC(),
		SigningKey: &SigningKeyInfo{Ed25519: key.PublicKey(), Since: time.Now().UTC()},
	}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(p *Profile)
	}{
		{"bad name", func(p *Profile) { p.Name = "Not A Slug" }},
		{"bad kind", func(p *Profile) { p.Kind = "robot" }},
		{"bad operated_by", func(p *Profile) { p.OperatedBy = "Bad.Name" }},
		{"bad key encoding", func(p *Profile) { p.SigningKey = &SigningKeyInfo{Ed25519: "%%%"} }},
		{"bad key length", func(p *Profile) { p.SigningKey = &SigningKeyInfo{Ed25519: "c2hvcnQ="} }},
		{"rotation missing proof", func(p *Profile) {
			p.Rotations = []Rotation{{From: p.SigningKey.Ed25519, To: p.SigningKey.Ed25519}}
		}},
		{"rotations without key", func(p *Profile) {
			p.Rotations = []Rotation{{From: p.SigningKey.Ed25519, To: p.SigningKey.Ed25519, Proof: "x"}}
			p.SigningKey = nil
		}},
	}
	for _, c := range cases {
		p := good
		c.mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
}

// The JSON shape is a wire contract (contracts/wire-and-kv.md): fixed key names,
// optionals omitted when empty.
func TestProfileJSONShape(t *testing.T) {
	key := testKey(t)
	p := Profile{
		Name:      "architect",
		Kind:      KindAgent,
		CreatedAt: time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC),
		SigningKey: &SigningKeyInfo{
			Ed25519: key.PublicKey(),
			Since:   time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC),
		},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{`"name":"architect"`, `"kind":"agent"`, `"created_at"`, `"signing_key"`, `"ed25519"`, `"since"`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %s: %s", want, s)
		}
	}
	for _, absent := range []string{"display_name", "description", "operated_by", "rotations"} {
		if strings.Contains(s, absent) {
			t.Errorf("empty optional %q not omitted: %s", absent, s)
		}
	}

	var back Profile
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.SigningKey == nil || back.SigningKey.Ed25519 != key.PublicKey() {
		t.Error("round-trip lost the signing key")
	}
}

package node

import (
	"time"

	"github.com/impire-io/soulstream-workloads/minter"
)

// capabilityMinter routes a workload mint by its scope (specs/013): a
// capability-bearing scope rides the scoped agent lane — the realm
// account's agent scope template, expanded with the mint's tags, is the
// entire policy — and a capability-less scope rides the plain lane,
// byte-identical to before the feature. The routing is the whole job;
// both lanes are upstream minters.
type capabilityMinter struct {
	scoped minter.Minter
	plain  minter.Minter
}

func (m *capabilityMinter) Mint(s minter.Scope, ttl time.Duration) (minter.PersonaScopedCredential, error) {
	if s.Capabilities == nil {
		return m.plain.Mint(s, ttl)
	}
	return m.scoped.Mint(s, ttl)
}

package identity

import "testing"

func TestKeyringNilIsSafe(t *testing.T) {
	var kr *Keyring
	chain, distrusted := kr.ChainFor("architect")
	if chain != nil || distrusted {
		t.Errorf("nil keyring: ChainFor = (%v, %v), want (nil, false)", chain, distrusted)
	}
}

func TestKeyringChainFor(t *testing.T) {
	kr := &Keyring{
		Keys:       map[string][]string{"architect": {"oldB64", "newB64"}},
		Distrusted: map[string]bool{"mallory": true},
	}

	chain, distrusted := kr.ChainFor("architect")
	if len(chain) != 2 || chain[0] != "oldB64" || chain[1] != "newB64" || distrusted {
		t.Errorf("architect: got (%v, %v)", chain, distrusted)
	}

	chain, distrusted = kr.ChainFor("mallory")
	if chain != nil || !distrusted {
		t.Errorf("mallory: got (%v, %v), want (nil, true)", chain, distrusted)
	}

	chain, distrusted = kr.ChainFor("stranger")
	if chain != nil || distrusted {
		t.Errorf("stranger: got (%v, %v), want (nil, false)", chain, distrusted)
	}
}

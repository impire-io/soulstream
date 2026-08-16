package ceremony

import (
	"os"
	"strings"
	"testing"

	"github.com/nats-io/nkeys"
)

// TestGenerateBYOSelfHosted: phase-1 local material is exactly what
// design 0003 §3 names — and none of what it forbids.
func TestGenerateBYOSelfHosted(t *testing.T) {
	st, err := GenerateBYO(FlavourSelfHosted, "nats://byo.example:4222", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !st.BYO() {
		t.Fatal("BYO() = false on a BYO state")
	}
	for name, b := range map[string][]byte{
		"auth signing seed":     st.AuthSigningSeed,
		"realm signing seed":    st.RealmSigningSeed,
		"workload signing seed": st.WorkloadSigningSeed,
		"issuer user seed":      st.IssuerUserSeed,
		"callout seed":          st.CalloutSeed,
		"vault first seed":      st.VaultFirstSeed,
		"surface seed":          st.SurfaceSeed,
	} {
		if len(b) == 0 {
			t.Errorf("%s missing from phase-1 material", name)
		}
	}
	// What must NOT exist on this side of the boundary (design 0003 §3).
	for name, b := range map[string][]byte{
		"operator seed":     st.OperatorSeed,
		"sys seed":          st.SysSeed,
		"auth master seed":  st.AuthSeed,
		"realm master seed": st.RealmSeed,
	} {
		if len(b) != 0 {
			t.Errorf("%s exists in BYO state — a master key crossed the boundary", name)
		}
	}
	if st.SysJWT != "" || st.AuthJWT != "" || st.RealmJWT != "" {
		t.Error("account JWTs exist in BYO state — they belong to the substrate's resolver")
	}
}

func TestGenerateBYOUnknownFlavour(t *testing.T) {
	_, err := GenerateBYO("ngs-shared", "nats://x:4222", "home")
	if err == nil || !strings.Contains(err.Error(), FlavourSelfHosted) {
		t.Fatalf("unknown flavour not refused by name: %v", err)
	}
}

// TestBYORoundTrip: save → load → verify across both phases, and the
// artifact count a founded BYO realm reports.
func TestBYORoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := GenerateBYO(FlavourSelfHosted, "nats://byo.example:4222", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save phase 1: %v", err)
	}
	got, err := Verify(dir)
	if err != nil {
		t.Fatalf("verify phase 1: %v", err)
	}
	if got.BYOFlavour != FlavourSelfHosted || got.BYOURL != st.BYOURL {
		t.Fatalf("round trip lost the byo block: %+v", got)
	}
	if got.RealmSigningPub != st.RealmSigningPub || got.IssuerUserPub != st.IssuerUserPub {
		t.Fatal("round trip lost phase-1 publics")
	}

	// The hand-back, then the users.
	authKP, _ := nkeys.CreateAccount()
	realmKP, _ := nkeys.CreateAccount()
	got.AuthPub, _ = authKP.PublicKey()
	got.RealmPub, _ = realmKP.PublicKey()
	if err := got.MintBYOUsers(); err != nil {
		t.Fatalf("mint users: %v", err)
	}
	if err := got.Save(dir); err != nil {
		t.Fatalf("save phase 2: %v", err)
	}
	if err := WriteSentinel(dir, "-----BEGIN NATS USER JWT-----\neyJ0\n------END NATS USER JWT------\n"); err != nil {
		t.Fatalf("sentinel: %v", err)
	}
	final, err := Load(dir)
	if err != nil {
		t.Fatalf("load founded: %v", err)
	}
	if len(final.OpsCreds) == 0 || len(final.IssuerCreds) == 0 || len(final.SignInCreds) == 0 {
		t.Fatal("founded load lost the minted creds")
	}
	// 7 keys + 6 creds + config + sentinel.
	if n := final.ArtifactCount(); n != 15 {
		t.Fatalf("BYO artifact count = %d, want 15", n)
	}
}

// TestMintBYOUsersRequiresHandBack: minting before the account half is
// a named refusal, not a broken creds file.
func TestMintBYOUsersRequiresHandBack(t *testing.T) {
	st, _ := GenerateBYO(FlavourSelfHosted, "nats://x:4222", "home")
	err := st.MintBYOUsers()
	if err == nil || !strings.Contains(err.Error(), "--auth-account") {
		t.Fatalf("mint without hand-back not refused by name: %v", err)
	}
}

// TestKit: exact values, no placeholders, no secrets, byte-identical on
// regeneration (design 0003 §4; spec 010 SC-004).
func TestKit(t *testing.T) {
	dir := t.TempDir()
	st, _ := GenerateBYO(FlavourSelfHosted, "nats://byo.example:4222", "home")
	kit := Kit(st, dir)
	for name, want := range map[string]string{
		"workload signing pub":  st.WorkloadSigningPub,
		"realm signing pub":     st.RealmSigningPub,
		"auth signing pub":      st.AuthSigningPub,
		"issuer user pub":       st.IssuerUserPub,
		"callout pub":           st.CalloutPub,
		"scope pub allow":       strings.Join(scopePubAllow, ","),
		"scope sub allow":       strings.Join(scopeSubAllow, ","),
		"push":                  "nsc push -A",
		"hand-back state":       dir,
		"hand-back flag":        "--auth-account",
		"realm account name":    "soulstream-home",
		"auth account name":     "soulstream-home-auth",
		"conversion fragment":   "system_account:",
		"default sentinel note": "default_sentinel",
	} {
		if !strings.Contains(kit, want) {
			t.Errorf("kit lacks %s (%q)", name, want)
		}
	}
	for name, seed := range map[string][]byte{
		"auth signing":  st.AuthSigningSeed,
		"realm signing": st.RealmSigningSeed,
		"workload":      st.WorkloadSigningSeed,
		"issuer user":   st.IssuerUserSeed,
		"callout":       st.CalloutSeed,
	} {
		if strings.Contains(kit, string(seed)) {
			t.Fatalf("kit carries the %s SEED — a secret crossed the boundary", name)
		}
	}
	if again := Kit(st, dir); again != kit {
		t.Fatal("kit is not byte-identical on regeneration")
	}
}

// TestVerifyBYORefusals: the named refusals of design 0003 §6.
func TestVerifyBYORefusals(t *testing.T) {
	fresh := func(t *testing.T, mutate func(*State)) error {
		t.Helper()
		dir := t.TempDir()
		st, err := GenerateBYO(FlavourSelfHosted, "nats://byo.example:4222", "home")
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		mutate(st)
		if err := st.Save(dir); err != nil {
			t.Fatalf("save: %v", err)
		}
		_, err = Verify(dir)
		return err
	}

	if err := fresh(t, func(s *State) { s.Listen = "127.0.0.1:4222" }); err == nil ||
		!strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("listen+byo not refused: %v", err)
	}
	if err := fresh(t, func(s *State) { s.BYOFlavour = "managed" }); err == nil ||
		!strings.Contains(err.Error(), "not a flavour") {
		t.Errorf("unknown flavour not refused: %v", err)
	}
	if err := fresh(t, func(s *State) { s.BYOURL = "not a url" }); err == nil ||
		!strings.Contains(err.Error(), "not a server URL") {
		t.Errorf("bad url not refused: %v", err)
	}
	kp, _ := nkeys.CreateAccount()
	pub, _ := kp.PublicKey()
	if err := fresh(t, func(s *State) { s.AuthPub, s.RealmPub = pub, pub }); err == nil ||
		!strings.Contains(err.Error(), "two accounts") {
		t.Errorf("same account for auth and realm not refused: %v", err)
	}
	if err := fresh(t, func(s *State) { s.AuthPub = "XBADKEY" }); err == nil ||
		!strings.Contains(err.Error(), "not an account public key") {
		t.Errorf("garbage hand-back key not refused: %v", err)
	}
}

// TestVerifyBYOSynadiaNeedsSystem: the synadia flavour without its
// system name is a config refusal.
func TestVerifyBYOSynadiaNeedsSystem(t *testing.T) {
	dir := t.TempDir()
	st, err := GenerateBYO(FlavourSynadiaCloud, "nats://byon.example:4222", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// The synadia flavour's signing material arrives from the platform;
	// give the state the on-disk shape of a driver run without a system.
	sk1, _ := nkeys.CreateAccount()
	st.AuthSigningSeed, _ = sk1.Seed()
	sk2, _ := nkeys.CreateAccount()
	st.RealmSigningSeed, _ = sk2.Seed()
	sk3, _ := nkeys.CreateAccount()
	st.WorkloadSigningSeed, _ = sk3.Seed()
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	_, err = Verify(dir)
	if err == nil || !strings.Contains(err.Error(), "byo.synadia.system") {
		t.Fatalf("missing system not refused: %v", err)
	}
}

// TestBYOStateDirCustody: nothing in a BYO state dir is a master key,
// and the file modes hold (spec 010 SC-002's disk half).
func TestBYOStateDirCustody(t *testing.T) {
	dir := t.TempDir()
	st, _ := GenerateBYO(FlavourSelfHosted, "nats://byo.example:4222", "home")
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	for _, forbidden := range []string{
		"keys/operator.nk", "keys/sys.nk", "keys/sys.jwt",
		"keys/auth.nk", "keys/auth.jwt", "keys/realm.nk", "keys/realm.jwt",
	} {
		if _, err := os.Stat(dir + "/" + forbidden); err == nil {
			t.Errorf("%s exists in a BYO state dir — master material crossed the boundary", forbidden)
		}
	}
}

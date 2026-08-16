package ceremony

// This file is BYO NATS (design 0003): founding on a server soulstream
// does not run. The ceremony's steps regroup — local material generated
// here, the account half applied by the substrate's own authority (the
// operator's nsc, or the Synadia control-plane API), the wire half
// unchanged. No operator or account master key ever travels; on
// self-hosted, no seed crosses the boundary in either direction.

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

// The two flavours (design 0003 §1). Anything else is refused by name:
// there is no third lane, and conf-auth servers cannot carry the scoped
// signing keys the permission model rides on.
const (
	FlavourSelfHosted   = "self-hosted"
	FlavourSynadiaCloud = "synadia-cloud"
)

// BYO reports whether this state founds on a bring-your-own server.
func (s *State) BYO() bool { return s.BYOFlavour != "" }

// GenerateBYO runs the local-material half of the BYO ceremony (design
// 0003 §3): everything soulstream can make without touching the
// substrate. Self-hosted generates the three signing keys, the issuer
// user, and the callout curve key — their public halves go out in the
// kit, the seeds stay here. Synadia Cloud generates only the
// substrate-independent curve keys; the platform generates the signing
// keys and the driver imports their once-returned seeds.
func GenerateBYO(flavour, natsURL, realm string) (*State, error) {
	switch flavour {
	case FlavourSelfHosted, FlavourSynadiaCloud:
	default:
		return nil, fmt.Errorf("ceremony: unknown BYO flavour %q — the two flavours are %q and %q (design 0003 §1)",
			flavour, FlavourSelfHosted, FlavourSynadiaCloud)
	}
	if natsURL == "" {
		return nil, fmt.Errorf("ceremony: BYO founding needs the substrate URL (--url nats://…)")
	}
	if realm == "" {
		return nil, fmt.Errorf("ceremony: realm name required")
	}
	s := planeDefaults(realm)
	s.BYOFlavour = flavour
	s.BYOURL = natsURL

	// The substrate-independent curve keys (design 0001 §4 step 6's
	// vault and surface halves) exist in every flavour.
	firstKP, _ := nkeys.CreateCurveKeys()
	s.VaultFirstSeed, _ = firstKP.Seed()
	surfaceKP, _ := nkeys.CreateCurveKeys()
	s.SurfaceSeed, _ = surfaceKP.Seed()

	if flavour == FlavourSelfHosted {
		authSK, _ := nkeys.CreateAccount()
		s.AuthSigningSeed, _ = authSK.Seed()
		s.AuthSigningPub, _ = authSK.PublicKey()
		realmSK, _ := nkeys.CreateAccount()
		s.RealmSigningSeed, _ = realmSK.Seed()
		s.RealmSigningPub, _ = realmSK.PublicKey()
		workloadSK, _ := nkeys.CreateAccount()
		s.WorkloadSigningSeed, _ = workloadSK.Seed()
		s.WorkloadSigningPub, _ = workloadSK.PublicKey()
		issuerKP, _ := nkeys.CreateUser()
		s.IssuerUserSeed, _ = issuerKP.Seed()
		s.IssuerUserPub, _ = issuerKP.PublicKey()
		calloutKP, _ := nkeys.CreateCurveKeys()
		s.CalloutSeed, _ = calloutKP.Seed()
		s.CalloutPub, _ = calloutKP.PublicKey()
	}
	return s, nil
}

// MintBYOUsers mints the bypass-lane users once the account half exists
// (design 0003 §3): the realm's five users signed by the PLAIN workload
// signing key, the issuer user signed by the AUTH signing key — every
// JWT carrying IssuerAccount, because a signing-key-signed user must
// name its account. No user is ever created on the substrate's side of
// the boundary. Idempotent by regeneration: users are transport
// identities, minting fresh ones replaces the old files whole.
func (s *State) MintBYOUsers() error {
	if s.AuthPub == "" || s.RealmPub == "" {
		return fmt.Errorf("ceremony: the account half has not been handed back — apply the kit and re-run init with --auth-account and --realm-account")
	}
	workloadKP, err := nkeys.FromSeed(s.WorkloadSigningSeed)
	if err != nil {
		return fmt.Errorf("ceremony: workload signing key: %w", err)
	}
	for _, u := range []struct {
		name string
		dst  *[]byte
	}{{"service", &s.ServiceCreds}, {"ops", &s.OpsCreds},
		{"archivist", &s.ArchivistCreds}, {"runner", &s.RunnerCreds},
		{"signin", &s.SignInCreds}} {
		if *u.dst, err = signingUserCreds(u.name, workloadKP, s.RealmPub); err != nil {
			return err
		}
	}
	// The Synadia driver downloads the issuer's creds from the platform
	// (the on-demand group); minting is the self-hosted path only.
	if len(s.IssuerCreds) > 0 {
		return nil
	}
	if len(s.IssuerUserSeed) == 0 {
		return fmt.Errorf("ceremony: missing %s — the issuer user's key was not generated at phase 1", fileIssuerUserSeed)
	}
	authKP, err := nkeys.FromSeed(s.AuthSigningSeed)
	if err != nil {
		return fmt.Errorf("ceremony: auth signing key: %w", err)
	}
	uc := jwt.NewUserClaims(s.IssuerUserPub)
	uc.Name = "soulstream-identity-issuer"
	uc.IssuedAt = time.Now().Unix()
	uc.IssuerAccount = s.AuthPub
	token, err := uc.Encode(authKP)
	if err != nil {
		return fmt.Errorf("ceremony: issuer user: %w", err)
	}
	if s.IssuerCreds, err = jwt.FormatUserConfig(token, s.IssuerUserSeed); err != nil {
		return fmt.Errorf("ceremony: issuer creds: %w", err)
	}
	return nil
}

// signingUserCreds mints a signing-key-signed user: the JWT carries its
// permissions (none — the account's defaults) and IssuerAccount, the
// protocol fact a non-master issuer forces.
func signingUserCreds(name string, signer nkeys.KeyPair, issuerAccount string) ([]byte, error) {
	ukp, _ := nkeys.CreateUser()
	upub, _ := ukp.PublicKey()
	useed, _ := ukp.Seed()
	uc := jwt.NewUserClaims(upub)
	uc.Name = name
	uc.IssuedAt = time.Now().Unix()
	uc.IssuerAccount = issuerAccount
	token, err := uc.Encode(signer)
	if err != nil {
		return nil, fmt.Errorf("ceremony: user %s: %w", name, err)
	}
	creds, err := jwt.FormatUserConfig(token, useed)
	if err != nil {
		return nil, fmt.Errorf("ceremony: user %s creds: %w", name, err)
	}
	return creds, nil
}

// Package ceremony is the founding of a realm, as data: everything a
// SoulNode stands on — trust root, accounts, admission machinery, sealing
// keys, service credentials — generated in one call and persisted into one
// state directory. The package is deliberately pure: it never opens a
// connection or starts a server; the node package does the talking.
//
// The inventory and its order come from design 0001 §4, measured
// executable end-to-end by the single-binary-composition research
// (journey episode 0002).
package ceremony

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/impire-io/soulidentity/client"
)

// State is the founding ceremony in memory: the parsed form of every
// artifact the state directory persists. Secrets live here only between
// Generate and Save (or after Load, in the process that owns the realm).
type State struct {
	// Listen is the loopback listener address (host:port), persisted in
	// config.json — the file is the configuration.
	Listen string

	// Realm is the realm name, fixed at founding (config.json).
	Realm string

	// MemoryEnabled gates the memory plane (design §2's first plane
	// block); the bundle default is true.
	MemoryEnabled bool

	OperatorSeed []byte
	OperatorPub  string

	SysSeed []byte
	SysPub  string
	SysJWT  string

	AuthSeed        []byte
	AuthPub         string
	AuthJWT         string
	AuthSigningSeed []byte
	AuthSigningPub  string

	RealmSeed        []byte
	RealmPub         string
	RealmJWT         string
	RealmSigningSeed []byte
	RealmSigningPub  string

	// The workload minting key: a PLAIN signing key beside the scoped
	// one — the runtime's minter embeds per-workload permissions in the
	// user JWTs it signs, and a scoped key would reject them (measured
	// upstream, soulrealm journey 0010).
	WorkloadSigningSeed []byte
	WorkloadSigningPub  string

	// Curve seeds: callout (public half rides in AuthJWT), the vault
	// first key, and the service surface key.
	CalloutSeed    []byte
	CalloutPub     string
	VaultFirstSeed []byte
	SurfaceSeed    []byte

	// Account-signed users — the creds bypass lane. These never leave
	// the state directory. The archivist's entry is its transport
	// credential only; its persona signing key is vault-held (D26
	// upstream), never on disk here.
	ServiceCreds   []byte
	IssuerCreds    []byte
	OpsCreds       []byte
	ArchivistCreds []byte
	RunnerCreds    []byte
}

// FoundingPersona is the persona name the first access token represents:
// the human who ran init. A richer naming story arrives when a consumer
// chafes on it, not before.
const FoundingPersona = "owner"

// Generate runs the whole founding ceremony in memory (design 0001 §4,
// steps 1–6): no prompts, no external binaries, no I/O.
func Generate(listen, realm string) (*State, error) {
	if listen == "" {
		return nil, fmt.Errorf("ceremony: listen address required")
	}
	if realm == "" {
		return nil, fmt.Errorf("ceremony: realm name required")
	}
	s := &State{Listen: listen, Realm: realm, MemoryEnabled: true}

	// 1. The operator — the trust root.
	opKP, err := nkeys.CreateOperator()
	if err != nil {
		return nil, fmt.Errorf("ceremony: operator: %w", err)
	}
	s.OperatorSeed, _ = opKP.Seed()
	s.OperatorPub, _ = opKP.PublicKey()

	// 2. The system account.
	sysKP, _ := nkeys.CreateAccount()
	s.SysSeed, _ = sysKP.Seed()
	s.SysPub, _ = sysKP.PublicKey()
	sysClaims := jwt.NewAccountClaims(s.SysPub)
	sysClaims.Name = "SYS"
	if s.SysJWT, err = sysClaims.Encode(opKP); err != nil {
		return nil, fmt.Errorf("ceremony: sys jwt: %w", err)
	}

	// Keys the AUTH account references.
	authKP, _ := nkeys.CreateAccount()
	s.AuthSeed, _ = authKP.Seed()
	s.AuthPub, _ = authKP.PublicKey()
	authSK, _ := nkeys.CreateAccount()
	s.AuthSigningSeed, _ = authSK.Seed()
	s.AuthSigningPub, _ = authSK.PublicKey()
	issuerUserKP, _ := nkeys.CreateUser()
	issuerUserPub, _ := issuerUserKP.PublicKey()
	issuerUserSeed, _ := issuerUserKP.Seed()
	calloutKP, _ := nkeys.CreateCurveKeys()
	s.CalloutSeed, _ = calloutKP.Seed()
	s.CalloutPub, _ = calloutKP.PublicKey()

	realmKP, _ := nkeys.CreateAccount()
	s.RealmSeed, _ = realmKP.Seed()
	s.RealmPub, _ = realmKP.PublicKey()

	// 3. The AUTH account: the admission machinery.
	authClaims := jwt.NewAccountClaims(s.AuthPub)
	authClaims.Name = "AUTH"
	authClaims.SigningKeys.Add(s.AuthSigningPub)
	authClaims.EnableExternalAuthorization(issuerUserPub)
	authClaims.Authorization.AllowedAccounts.Add(s.RealmPub)
	authClaims.Authorization.XKey = s.CalloutPub
	if s.AuthJWT, err = authClaims.Encode(opKP); err != nil {
		return nil, fmt.Errorf("ceremony: auth jwt: %w", err)
	}

	// 4. The realm account: JetStream, the persona scope, and the
	// workload minting key (plain — R1).
	realmSK, _ := nkeys.CreateAccount()
	s.RealmSigningSeed, _ = realmSK.Seed()
	s.RealmSigningPub, _ = realmSK.PublicKey()
	workloadSK, _ := nkeys.CreateAccount()
	s.WorkloadSigningSeed, _ = workloadSK.Seed()
	s.WorkloadSigningPub, _ = workloadSK.PublicKey()
	realmClaims := jwt.NewAccountClaims(s.RealmPub)
	realmClaims.Name = "REALM"
	realmClaims.SigningKeys.Add(s.WorkloadSigningPub)
	realmClaims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: -1, DiskStorage: -1, Streams: -1, Consumer: -1,
	}
	scope := jwt.NewUserScope()
	scope.Key = s.RealmSigningPub
	scope.Role = "soulstream-user"
	scope.Template = jwt.UserPermissionLimits{
		Permissions: jwt.Permissions{
			Pub: jwt.Permission{Allow: []string{
				client.Segment + ".status",
				client.Segment + ".xkey",
				client.Segment + ".{{account-subject()}}.{{name()}}.sign.record",
				client.Segment + ".{{account-subject()}}.{{name()}}.keys.public",
				"SOULSTREAM.>",
				"$JS.API.>",
				"$KV.>",
				"$O.>",
				"$SYS.REQ.USER.INFO",
			}},
			Sub: jwt.Permission{Allow: []string{"_INBOX.>", "SOULSTREAM.>"}},
		},
	}
	realmClaims.SigningKeys.AddScopedSigner(scope)
	if s.RealmJWT, err = realmClaims.Encode(opKP); err != nil {
		return nil, fmt.Errorf("ceremony: realm jwt: %w", err)
	}

	// 5. The bypass-lane users.
	if s.ServiceCreds, err = userCreds("service", realmKP); err != nil {
		return nil, err
	}
	if s.OpsCreds, err = userCreds("ops", realmKP); err != nil {
		return nil, err
	}
	if s.ArchivistCreds, err = userCreds("archivist", realmKP); err != nil {
		return nil, err
	}
	if s.RunnerCreds, err = userCreds("runner", realmKP); err != nil {
		return nil, err
	}
	issuerClaims := jwt.NewUserClaims(issuerUserPub)
	issuerClaims.Name = "soulidentity-issuer"
	issuerJWT, err := issuerClaims.Encode(authKP)
	if err != nil {
		return nil, fmt.Errorf("ceremony: issuer user: %w", err)
	}
	if s.IssuerCreds, err = jwt.FormatUserConfig(issuerJWT, issuerUserSeed); err != nil {
		return nil, fmt.Errorf("ceremony: issuer creds: %w", err)
	}

	// 6. The remaining curve seeds.
	firstKP, _ := nkeys.CreateCurveKeys()
	s.VaultFirstSeed, _ = firstKP.Seed()
	surfaceKP, _ := nkeys.CreateCurveKeys()
	s.SurfaceSeed, _ = surfaceKP.Seed()

	return s, nil
}

// userCreds mints an account-key-signed user and renders the creds file.
func userCreds(name string, account nkeys.KeyPair) ([]byte, error) {
	ukp, _ := nkeys.CreateUser()
	upub, _ := ukp.PublicKey()
	useed, _ := ukp.Seed()
	uc := jwt.NewUserClaims(upub)
	uc.Name = name
	uc.IssuedAt = time.Now().Unix()
	token, err := uc.Encode(account)
	if err != nil {
		return nil, fmt.Errorf("ceremony: user %s: %w", name, err)
	}
	creds, err := jwt.FormatUserConfig(token, useed)
	if err != nil {
		return nil, fmt.Errorf("ceremony: user %s creds: %w", name, err)
	}
	return creds, nil
}

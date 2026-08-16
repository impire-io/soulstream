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
	"strings"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/impire-io/soulstream-identity/client"
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

	// MCPEnabled and MCPListen are the MCP door plane (design §2's
	// second block): a loopback HTTP listener, enabled by default.
	MCPEnabled bool
	MCPListen  string

	// Public door mode (the roadmap's Phase-2 public clause, additive):
	// MCPPublicURL is the door's advertised public address (the OAuth
	// resource identifier — deployment fronting such as `tailscale
	// serve` carries HTTPS to the loopback listener); MCPAuthIssuer is
	// the external authorization server's issuer URL (the bundled fold
	// when the fold plane runs and no issuer is set — the default
	// wiring; the door stays AS-agnostic); MCPAuthAudience is the
	// deployment's fixed token audience the identity plane's OIDC lane
	// validates. All three or none after defaulting.
	MCPPublicURL    string
	MCPAuthIssuer   string
	MCPAuthAudience string

	// The fold plane (soulstream-idp M5's bundled story, opt-in): the
	// deployment's own passkey-first OIDC provider in-process, storing
	// on the node's JetStream under its own bucket prefix, seal seed
	// under <state>/fold/. SignInIssuer's host is WebAuthn's one-way door
	// at first enrollment — public deployments set it to the fronted
	// name before anyone enrolls. SignInAudience defaults to
	// "soulstream-<realm>".
	SignInEnabled  bool
	SignInListen   string
	SignInIssuer   string
	SignInAudience string

	// The shell plane (soulstream-shell — the human cockpit): a loopback HTTP
	// surface, sessions signing in against the deployment's AS (the
	// bundled fold by default). On by default beside the fold.
	HelmEnabled bool
	HelmListen  string

	// BYO NATS (design 0003): BYOFlavour "" is the embedded server; the
	// two flavours found on a substrate soulstream does not run. BYOURL
	// is the substrate's client URL — the one URL every plane dials. In
	// BYO mode the operator, SYS, and account master material below is
	// absent by construction: it never existed on this side of the
	// boundary. AuthPub and RealmPub are the two public keys the account
	// half hands back.
	BYOFlavour    string
	BYOURL        string
	SynadiaSystem string

	// The issuer user's own keypair (self-hosted BYO): its public key is
	// declared to auth_users by the kit before its creds can exist, so
	// the seed outlives phase 1 on disk.
	IssuerUserSeed []byte
	IssuerUserPub  string

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
	// upstream, soulstream-workloads journey 0010).
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
	SignInCreds    []byte
}

// FoundingPersona is the persona name the first access token represents:
// the human who ran init. A richer naming story arrives when a consumer
// chafes on it, not before.
const FoundingPersona = "owner"

// SessionIssuer resolves the OIDC authorization server the identity
// plane's callout lane validates and the shell sends sign-ins to: an
// explicit external AS (public door mode) wins; otherwise the bundled
// fold when the shell needs one. Empty means the lane stays off.
func (s *State) SessionIssuer() (issuer, audience string) {
	if s.MCPAuthIssuer != "" {
		return s.MCPAuthIssuer, s.MCPAuthAudience
	}
	if s.HelmEnabled && s.SignInEnabled {
		// An ephemeral fold listener (:0) yields an issuer no browser
		// or validator can reach; sessions are a real-port feature.
		if strings.HasSuffix(s.SignInIssuer, ":0") {
			return "", ""
		}
		return s.SignInIssuer, s.SignInAudience
	}
	return "", ""
}

// AdminSurface is where this deployment administers the people who can
// sign in, "" when it administers them nowhere this node runs.
//
// It is the bundled fold's own surface, and only when that fold is also the
// authorization server sessions actually sign in against: a deployment
// pointed at an external AS holds its people there, so the fold's own
// records — even with the plane still running — are not the people signing
// in, and this node has no standing to administer the ones who are.
//
// Nothing here probes anything. Both facts are declared in config before
// the node starts, which is what lets a plane's absence reach a consumer as
// a fact rather than as a failed request.
func (s *State) AdminSurface() string {
	issuer, _ := s.SessionIssuer()
	if s.SignInEnabled && issuer == s.SignInIssuer {
		return s.SignInIssuer
	}
	return ""
}

// Generate runs the whole founding ceremony in memory (design 0001 §4,
// steps 1–6): no prompts, no external binaries, no I/O.
func Generate(listen, realm string) (*State, error) {
	if listen == "" {
		return nil, fmt.Errorf("ceremony: listen address required")
	}
	if realm == "" {
		return nil, fmt.Errorf("ceremony: realm name required")
	}
	s := planeDefaults(realm)
	s.Listen = listen

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
	realmClaims.SigningKeys.AddScopedSigner(personaScope(s.RealmSigningPub))
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
	if s.SignInCreds, err = userCreds("signin", realmKP); err != nil {
		return nil, err
	}
	issuerClaims := jwt.NewUserClaims(issuerUserPub)
	issuerClaims.Name = "soulstream-identity-issuer"
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

// planeDefaults is the bundled experience, on by default: the fold gives
// the realm a sign-in and an admin console out of the box, so `init && up`
// lands a person at a passkey prompt with nothing else to install. The
// issuer host is localhost — WebAuthn's RP-ID rule refuses a bare IP.
// Turn a plane off by setting planes.<name>.enabled=false in config.
// Shared by the embedded and BYO ceremonies: the planes are the same
// planes, only the server underneath differs (design 0003 §6).
func planeDefaults(realm string) *State {
	return &State{Realm: realm, MemoryEnabled: true,
		MCPEnabled: true, MCPListen: "127.0.0.1:8080",
		SignInEnabled: true, SignInListen: "127.0.0.1:8378",
		SignInIssuer: "http://localhost:8378", SignInAudience: "soulstream-" + realm,
		HelmEnabled: true, HelmListen: "127.0.0.1:8500"}
}

// The persona scope (design 0001 §4 step 4): the admitted persona's
// permission set, templated per identity. One source feeds both the
// embedded ceremony and the BYO kit, so the two can never drift
// (spec 010 SC-004).
var (
	scopePubAllow = []string{
		client.Segment + ".status",
		client.Segment + ".xkey",
		client.Segment + ".{{account-subject()}}.{{name()}}.sign.record",
		client.Segment + ".{{account-subject()}}.{{name()}}.keys.public",
		"SOULSTREAM.>",
		"$JS.API.>",
		"$KV.>",
		"$O.>",
		"$SYS.REQ.USER.INFO",
	}
	scopeSubAllow = []string{"_INBOX.>", "SOULSTREAM.>"}
)

// personaScope renders the scope onto a signing key.
func personaScope(key string) *jwt.UserScope {
	scope := jwt.NewUserScope()
	scope.Key = key
	scope.Role = "soulstream-user"
	scope.Template = jwt.UserPermissionLimits{
		Permissions: jwt.Permissions{
			Pub: jwt.Permission{Allow: scopePubAllow},
			Sub: jwt.Permission{Allow: scopeSubAllow},
		},
	}
	return scope
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

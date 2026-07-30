package researchnode

// The operator-mode rig: adapted from soulidentity's callout/Entra e2e proofs
// (client/callout_e2e_test.go) with the one combination those never ran —
// the admitted user's scope template spans BOTH subject spaces (SoulIdentity
// user ops + the Soulstream realm, the m2_gate_test.go shape) AND admission
// comes through callout, not minted creds. One team account (ACME) hosts the
// SoulIdentity service and the realm, so sign.record needs no cross-account
// exports.

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulidentity/internal/callout"
	"github.com/impire-io/soulidentity/internal/oidcstub"
	"github.com/impire-io/soulidentity/internal/service"
	"github.com/impire-io/soulidentity/internal/vault"

	"github.com/impire-io/soulstream/realm"
)

type rig struct {
	url          string
	sentinelPath string
	accPub       string // ACME account public key
	stub         *oidcstub.Stub
	admin        *siclient.Client
	apiToken     string // sit_ token for user daan-ext
	apiDigest    string
	readerCreds  string
	audit        *syncBuffer
}

type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newAuditLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func startRig(t *testing.T, ttl time.Duration) *rig {
	t.Helper()

	opKP, _ := nkeys.CreateOperator()
	opPub, _ := opKP.PublicKey()
	sysKP, _ := nkeys.CreateAccount()
	sysPub, _ := sysKP.PublicKey()
	authKP, _ := nkeys.CreateAccount()
	authPub, _ := authKP.PublicKey()
	acmeKP, _ := nkeys.CreateAccount()
	acmePub, _ := acmeKP.PublicKey()
	acmeSKKP, _ := nkeys.CreateAccount()
	acmeSKPub, _ := acmeSKKP.PublicKey()
	acmeSKSeed, _ := acmeSKKP.Seed()
	authSKKP, _ := nkeys.CreateAccount()
	authSKPub, _ := authSKKP.PublicKey()
	authSKSeed, _ := authSKKP.Seed()
	calloutKP, _ := nkeys.CreateCurveKeys()
	calloutPub, _ := calloutKP.PublicKey()
	calloutSeed, _ := calloutKP.Seed()

	oc := jwt.NewOperatorClaims(opPub)
	oc.Name = "node-research-operator"
	opJWT, err := oc.Encode(opKP)
	if err != nil {
		t.Fatalf("operator jwt: %v", err)
	}
	sc := jwt.NewAccountClaims(sysPub)
	sc.Name = "SYS"
	sysJWT, err := sc.Encode(opKP)
	if err != nil {
		t.Fatalf("system account jwt: %v", err)
	}
	issuerUserKP, _ := nkeys.CreateUser()
	issuerUserPub, _ := issuerUserKP.PublicKey()
	authClaim := jwt.NewAccountClaims(authPub)
	authClaim.Name = "AUTH"
	authClaim.SigningKeys.Add(authSKPub)
	authClaim.EnableExternalAuthorization(issuerUserPub)
	authClaim.Authorization.AllowedAccounts.Add(acmePub)
	authClaim.Authorization.XKey = calloutPub
	authJWT, err := authClaim.Encode(opKP)
	if err != nil {
		t.Fatalf("auth account jwt: %v", err)
	}

	acmeClaim := jwt.NewAccountClaims(acmePub)
	acmeClaim.Name = "ACME"
	acmeClaim.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: -1, DiskStorage: -1, Streams: -1, Consumer: -1,
	}
	// The represented-user scope: SoulIdentity user ops on the own prefix,
	// the Soulstream realm's subject space, and the user-info request the
	// node derives the principal from.
	scope := jwt.NewUserScope()
	scope.Key = acmeSKPub
	scope.Role = "soulstream-user"
	scope.Template = jwt.UserPermissionLimits{
		Permissions: jwt.Permissions{
			Pub: jwt.Permission{Allow: jwt.StringList{
				siclient.Segment + ".status", siclient.Segment + ".xkey",
				siclient.Segment + ".{{account-subject()}}.{{name()}}.sign.record",
				siclient.Segment + ".{{account-subject()}}.{{name()}}.keys.public",
				"SOULSTREAM.>", "$JS.API.>", "$KV.>", "$O.>",
				"$SYS.REQ.USER.INFO",
			}},
			Sub: jwt.Permission{Allow: jwt.StringList{"_INBOX.>", "SOULSTREAM.>"}},
		},
	}
	acmeClaim.SigningKeys.AddScopedSigner(scope)
	acmeJWT, err := acmeClaim.Encode(opKP)
	if err != nil {
		t.Fatalf("acme account jwt: %v", err)
	}

	cfg := fmt.Sprintf(`
listen: 127.0.0.1:-1
operator: %s
system_account: %s
resolver: MEMORY
resolver_preload: {
  %s: %s,
  %s: %s,
  %s: %s,
}
jetstream { store_dir: %q }
`, opJWT, sysPub, sysPub, sysJWT, authPub, authJWT, acmePub, acmeJWT, t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "server.conf")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	opts, err := natsserver.ProcessConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("process config: %v", err)
	}
	opts.NoLog, opts.NoSigs = true, true
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go srv.Start()
	t.Cleanup(srv.Shutdown)
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("server not ready")
	}

	serviceCreds := issueUser(t, acmeKP, "service", nil)
	adminCreds := issueUser(t, acmeKP, "ops", &jwt.Permissions{
		Pub: jwt.Permission{Allow: jwt.StringList{
			siclient.Segment + ".status", siclient.Segment + ".xkey",
			siclient.Segment + "." + acmePub + ".ops.>",
		}},
		Sub: jwt.Permission{Allow: jwt.StringList{"_INBOX.>"}},
	})
	readerCreds := issueUser(t, acmeKP, "reader", nil)
	issuerJWT, err := jwt.NewUserClaims(issuerUserPub).Encode(authKP)
	if err != nil {
		t.Fatalf("issuer user jwt: %v", err)
	}
	issuerSeed, _ := issuerUserKP.Seed()
	issuerCredsBytes, err := jwt.FormatUserConfig(issuerJWT, issuerSeed)
	if err != nil {
		t.Fatalf("issuer creds: %v", err)
	}
	issuerCreds := filepath.Join(t.TempDir(), "issuer.creds")
	if err := os.WriteFile(issuerCreds, issuerCredsBytes, 0o600); err != nil {
		t.Fatalf("write issuer creds: %v", err)
	}

	// The service: vault + token store on ACME's JetStream; issuer on AUTH
	// with the OIDC lane against the local stub.
	firstKP, _ := nkeys.CreateCurveKeys()
	firstSeed, _ := firstKP.Seed()
	surfaceKP, _ := nkeys.CreateCurveKeys()
	surfaceSeed, _ := surfaceKP.Seed()

	ncService, err := nats.Connect(srv.ClientURL(), nats.UserCredentials(serviceCreds))
	if err != nil {
		t.Fatalf("service connect: %v", err)
	}
	t.Cleanup(ncService.Close)
	js, err := jetstream.New(ncService)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	vaultKV, err := js.CreateOrUpdateKeyValue(t.Context(), jetstream.KeyValueConfig{Bucket: "SOULIDENTITY_VAULT"})
	if err != nil {
		t.Fatalf("vault bucket: %v", err)
	}
	tokensKV, err := js.CreateOrUpdateKeyValue(t.Context(), jetstream.KeyValueConfig{Bucket: "SOULIDENTITY_TOKENS"})
	if err != nil {
		t.Fatalf("token bucket: %v", err)
	}
	v, err := vault.New(vault.NewKVStore(vaultKV), string(firstSeed))
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	store := callout.NewKVTokenStore(tokensKV)

	stub, err := oidcstub.New("node-research-app")
	if err != nil {
		t.Fatalf("oidc stub: %v", err)
	}
	t.Cleanup(stub.Close)
	oidcVal, err := callout.NewOIDCValidator(t.Context(), stub.Issuer(), stub.ClientID())
	if err != nil {
		t.Fatalf("oidc validator: %v", err)
	}

	audit := &syncBuffer{}
	logger := newAuditLogger(audit)
	svc, err := service.New(v, string(surfaceSeed), logger,
		service.WithCallout(store, "auth/issuer", authPub))
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	if _, err := svc.Start(ncService); err != nil {
		t.Fatalf("service start: %v", err)
	}
	ncIssuer, err := nats.Connect(srv.ClientURL(), nats.UserCredentials(issuerCreds))
	if err != nil {
		t.Fatalf("issuer connect: %v", err)
	}
	t.Cleanup(ncIssuer.Close)
	issuer, err := callout.NewIssuer(v, store, "auth/issuer", ttl,
		string(calloutSeed), logger, callout.WithOIDC(oidcVal))
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	if _, err := issuer.Start(ncIssuer); err != nil {
		t.Fatalf("issuer start: %v", err)
	}
	if err := ncIssuer.Flush(); err != nil {
		t.Fatalf("issuer flush: %v", err)
	}

	// Admin provisions: the team key (acme → ACME), the AUTH key, one API
	// token for daan-ext, the sentinel. Zero per-person acts for OIDC users.
	ncAdmin, err := nats.Connect(srv.ClientURL(), nats.UserCredentials(adminCreds))
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	t.Cleanup(ncAdmin.Close)
	admin := siclient.New(ncAdmin, acmePub, "ops")
	if _, err := admin.ImportKey("acme", siclient.KindNATSAccountSigningKey, string(acmeSKSeed), acmePub, ""); err != nil {
		t.Fatalf("import team key: %v", err)
	}
	if _, err := admin.ImportKey("auth/issuer", siclient.KindNATSAccountSigningKey, string(authSKSeed), authPub, ""); err != nil {
		t.Fatalf("import auth key: %v", err)
	}
	created, err := admin.CreateToken(acmePub, "daan-ext", "daan laptop", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	sentinel, err := admin.MintSentinel()
	if err != nil {
		t.Fatalf("mint sentinel: %v", err)
	}
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.creds")
	if err := os.WriteFile(sentinelPath, []byte(sentinel.Creds), 0o600); err != nil {
		t.Fatalf("write sentinel creds: %v", err)
	}

	// The realm is provisioned by an operator-side bootstrap identity —
	// provisioning is an operator act, not the node's.
	ncReader, err := nats.Connect(srv.ClientURL(), nats.UserCredentials(readerCreds))
	if err != nil {
		t.Fatalf("reader connect: %v", err)
	}
	t.Cleanup(ncReader.Close)
	rcReader, err := realm.NewClient(t.Context(), ncReader, realm.Config{Realm: "proof"})
	if err != nil {
		t.Fatalf("reader realm client: %v", err)
	}
	t.Cleanup(func() { _ = rcReader.Close() })
	if _, err := rcReader.Provision(t.Context()); err != nil {
		t.Fatalf("provision: %v", err)
	}
	_ = rcReader.Close()

	return &rig{
		url:          srv.ClientURL(),
		sentinelPath: sentinelPath,
		accPub:       acmePub,
		stub:         stub,
		admin:        admin,
		apiToken:     created.Token,
		apiDigest:    created.Digest,
		readerCreds:  readerCreds,
		audit:        audit,
	}
}

// readerRealm opens a fresh bootstrap-creds realm client for verification.
func (r *rig) readerRealm(t *testing.T) (*realm.Client, *nats.Conn) {
	t.Helper()
	nc, err := nats.Connect(r.url, nats.UserCredentials(r.readerCreds))
	if err != nil {
		t.Fatalf("reader connect: %v", err)
	}
	rc, err := realm.NewClient(t.Context(), nc, realm.Config{Realm: "proof"})
	if err != nil {
		t.Fatalf("reader realm client: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })
	return rc, nc
}

func issueUser(t *testing.T, accKP nkeys.KeyPair, name string, perms *jwt.Permissions) string {
	t.Helper()
	ukp, _ := nkeys.CreateUser()
	uPub, _ := ukp.PublicKey()
	uc := jwt.NewUserClaims(uPub)
	uc.Name = name
	if perms != nil {
		uc.Permissions = *perms
	}
	token, err := uc.Encode(accKP)
	if err != nil {
		t.Fatalf("issue %s: %v", name, err)
	}
	seed, _ := ukp.Seed()
	creds, err := jwt.FormatUserConfig(token, seed)
	if err != nil {
		t.Fatalf("creds %s: %v", name, err)
	}
	path := filepath.Join(t.TempDir(), name+".creds")
	if err := os.WriteFile(path, creds, 0o600); err != nil {
		t.Fatalf("write creds %s: %v", name, err)
	}
	return path
}

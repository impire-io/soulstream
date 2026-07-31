// Package soulnoderig is the throwaway rig for the single-binary-composition
// research topic (hq/01-RESEARCH/single-binary-composition/README.md).
//
// Provision is the Bar 3 artifact: the entire first-boot ceremony as one
// function — every key, JWT, bucket, and creds artifact generated in code
// from nothing but an empty directory, no config file, no nsc, no manual
// step. The embedded server it boots is the Bar 1 subject: operator mode
// with auth-callout admission, provisioned purely through server.Options
// (MemAccResolver — the config-file MEMORY resolver the sibling wire rig
// used is deliberately avoided), optionally with no TCP listener at all
// (DontListen + in-process connections — the SoulNode shape).
package soulnoderig

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/jwt/v2"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulidentity/internal/callout"
	"github.com/impire-io/soulidentity/internal/service"
	"github.com/impire-io/soulidentity/internal/vault"
)

// Inventory names every artifact the ceremony generates, in order. The Bar 3
// check is that this list and the code below agree 1:1 — each entry cites the
// provision step that creates it.
var Inventory = []string{
	"operator nkey (trust root; TrustedKeys on the server)",
	"system account nkey + JWT (SYS)",
	"AUTH account nkey + JWT (external authorization: issuer user, allowed accounts, callout xkey)",
	"AUTH account signing key (signs admitted-user JWTs; imported into the vault as auth/issuer)",
	"AUTH issuer user nkey + creds (the callout issuer's own connection)",
	"callout curve keypair (public in the AUTH JWT, seed to the issuer)",
	"realm account nkey + JWT (JetStream unlimited; hosts service and realm)",
	"realm scoped signing key + scope template (the admitted persona's permission set)",
	"realm service user creds + realm ops user creds (account-key signed, bypass lane)",
	"vault first key (curve seed; seals SOULIDENTITY_VAULT)",
	"service surface xkey (curve seed; seals the request/reply surface)",
	"JetStream KV buckets SOULIDENTITY_VAULT + SOULIDENTITY_TOKENS",
	"sentinel creds (deny-all bearer artifact, minted by the service, written to disk)",
	"API token sit_… + revocation digest (minted by the service)",
}

// SyncBuffer collects the service/issuer audit log for assertions.
type SyncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *SyncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *SyncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func timeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// Rig is a provisioned SoulNode-shaped deployment: embedded operator-mode
// server, SoulIdentity service + callout issuer in-process, admission
// artifacts minted and ready to be exercised.
type Rig struct {
	Srv          *natsserver.Server
	InProcess    bool
	RealmPub     string
	AuthPub      string
	Persona      string
	SentinelPath string
	Token        string
	Digest       string
	Admin        *siclient.Client
	Audit        *SyncBuffer

	conns []*nats.Conn
}

// Connect opens a client connection to the rig's server the way the rig was
// provisioned to be reached: in-process when DontListen, TCP otherwise.
func (r *Rig) Connect(opts ...nats.Option) (*nats.Conn, error) {
	url := nats.DefaultURL
	if r.InProcess {
		opts = append(opts, nats.InProcessServer(r.Srv))
	} else {
		url = r.Srv.ClientURL()
	}
	// Fail fast: admission refusals must surface as connect errors.
	opts = append(opts, nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	return nats.Connect(url, opts...)
}

func (r *Rig) track(nc *nats.Conn, err error) (*nats.Conn, error) {
	if err == nil {
		r.conns = append(r.conns, nc)
	}
	return nc, err
}

// Shutdown closes every rig-owned connection and stops the server.
func (r *Rig) Shutdown() {
	for _, nc := range r.conns {
		nc.Close()
	}
	r.Srv.Shutdown()
}

// Provision runs the whole first-boot ceremony into dir and returns a running
// rig. inProcess=true boots the server with no TCP listener at all
// (DontListen); every connection then rides nats.InProcessServer — the
// SoulNode shape. inProcess=false is the TCP control arm.
func Provision(dir string, inProcess bool) (*Rig, error) {
	// --- keys (Inventory 1–8) ---
	opKP, _ := nkeys.CreateOperator()
	opPub, _ := opKP.PublicKey()

	sysKP, _ := nkeys.CreateAccount()
	sysPub, _ := sysKP.PublicKey()

	authKP, _ := nkeys.CreateAccount()
	authPub, _ := authKP.PublicKey()
	authSK, _ := nkeys.CreateAccount()
	authSKPub, _ := authSK.PublicKey()
	authSKSeed, _ := authSK.Seed()

	issuerUserKP, _ := nkeys.CreateUser()
	issuerUserPub, _ := issuerUserKP.PublicKey()
	issuerUserSeed, _ := issuerUserKP.Seed()

	calloutKP, _ := nkeys.CreateCurveKeys()
	calloutPub, _ := calloutKP.PublicKey()
	calloutSeed, _ := calloutKP.Seed()

	realmKP, _ := nkeys.CreateAccount()
	realmPub, _ := realmKP.PublicKey()
	realmSK, _ := nkeys.CreateAccount()
	realmSKPub, _ := realmSK.PublicKey()
	realmSKSeed, _ := realmSK.Seed()

	firstKP, _ := nkeys.CreateCurveKeys()
	firstSeed, _ := firstKP.Seed()
	surfaceKP, _ := nkeys.CreateCurveKeys()
	surfaceSeed, _ := surfaceKP.Seed()

	// --- account JWTs (Inventory 2, 3, 7, 8) ---
	sysClaims := jwt.NewAccountClaims(sysPub)
	sysClaims.Name = "SYS"
	sysJWT, err := sysClaims.Encode(opKP)
	if err != nil {
		return nil, fmt.Errorf("sys jwt: %w", err)
	}

	authClaims := jwt.NewAccountClaims(authPub)
	authClaims.Name = "AUTH"
	authClaims.SigningKeys.Add(authSKPub)
	authClaims.EnableExternalAuthorization(issuerUserPub)
	authClaims.Authorization.AllowedAccounts.Add(realmPub)
	authClaims.Authorization.XKey = calloutPub
	authJWT, err := authClaims.Encode(opKP)
	if err != nil {
		return nil, fmt.Errorf("auth jwt: %w", err)
	}

	realmClaims := jwt.NewAccountClaims(realmPub)
	realmClaims.Name = "REALM"
	realmClaims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: -1, DiskStorage: -1, Streams: -1, Consumer: -1,
	}
	scope := jwt.NewUserScope()
	scope.Key = realmSKPub
	scope.Role = "soulstream-user"
	scope.Template = jwt.UserPermissionLimits{
		Permissions: jwt.Permissions{
			Pub: jwt.Permission{Allow: []string{
				siclient.Segment + ".status",
				siclient.Segment + ".xkey",
				siclient.Segment + ".{{account-subject()}}.{{name()}}.sign.record",
				siclient.Segment + ".{{account-subject()}}.{{name()}}.keys.public",
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
	realmJWT, err := realmClaims.Encode(opKP)
	if err != nil {
		return nil, fmt.Errorf("realm jwt: %w", err)
	}

	// --- the server, pure server.Options: no config file (Bar 1 subject) ---
	res := &natsserver.MemAccResolver{}
	for pub, token := range map[string]string{sysPub: sysJWT, authPub: authJWT, realmPub: realmJWT} {
		if err := res.Store(pub, token); err != nil {
			return nil, fmt.Errorf("resolver store: %w", err)
		}
	}
	opts := &natsserver.Options{
		Host:            "127.0.0.1",
		Port:            -1,
		DontListen:      inProcess,
		JetStream:       true,
		StoreDir:        filepath.Join(dir, "jetstream"),
		TrustedKeys:     []string{opPub},
		SystemAccount:   sysPub,
		AccountResolver: res,
		NoLog:           true,
		NoSigs:          true,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("new server: %w", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		srv.Shutdown()
		return nil, fmt.Errorf("server not ready")
	}

	r := &Rig{Srv: srv, InProcess: inProcess, RealmPub: realmPub, AuthPub: authPub,
		Persona: "daan-ext", Audit: &SyncBuffer{}}
	fail := func(err error) (*Rig, error) { r.Shutdown(); return nil, err }
	logger := slog.New(slog.NewTextHandler(r.Audit, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// --- realm users, account-key signed: the creds bypass lane (Inventory 9) ---
	userJWTSeed := func(name string, signer nkeys.KeyPair) (string, string, error) {
		ukp, _ := nkeys.CreateUser()
		upub, _ := ukp.PublicKey()
		useed, _ := ukp.Seed()
		uc := jwt.NewUserClaims(upub)
		uc.Name = name
		token, err := uc.Encode(signer)
		return token, string(useed), err
	}
	serviceJWT, serviceSeed, err := userJWTSeed("service", realmKP)
	if err != nil {
		return fail(fmt.Errorf("service user: %w", err))
	}
	opsJWT, opsSeed, err := userJWTSeed("ops", realmKP)
	if err != nil {
		return fail(fmt.Errorf("ops user: %w", err))
	}
	issuerClaims := jwt.NewUserClaims(issuerUserPub)
	issuerClaims.Name = "soulidentity-issuer"
	issuerJWT, err := issuerClaims.Encode(authKP)
	if err != nil {
		return fail(fmt.Errorf("issuer user: %w", err))
	}

	// --- vault, buckets, service, issuer: all in-process (Inventory 10–12) ---
	ncService, err := r.track(r.Connect(nats.UserJWTAndSeed(serviceJWT, serviceSeed), nats.Name("soulidentity")))
	if err != nil {
		return fail(fmt.Errorf("service connect: %w", err))
	}
	js, err := jetstream.New(ncService)
	if err != nil {
		return fail(fmt.Errorf("jetstream: %w", err))
	}
	ctx, cancel := timeoutCtx()
	defer cancel()
	vaultKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "SOULIDENTITY_VAULT"})
	if err != nil {
		return fail(fmt.Errorf("vault bucket: %w", err))
	}
	tokensKV, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "SOULIDENTITY_TOKENS"})
	if err != nil {
		return fail(fmt.Errorf("tokens bucket: %w", err))
	}
	v, err := vault.New(vault.NewKVStore(vaultKV), string(firstSeed))
	if err != nil {
		return fail(fmt.Errorf("vault: %w", err))
	}
	store := callout.NewKVTokenStore(tokensKV)
	svc, err := service.New(v, string(surfaceSeed), logger,
		service.WithCallout(store, "auth/issuer", authPub))
	if err != nil {
		return fail(fmt.Errorf("service: %w", err))
	}
	if _, err := svc.Start(ncService); err != nil {
		return fail(fmt.Errorf("service start: %w", err))
	}

	ncIssuer, err := r.track(r.Connect(nats.UserJWTAndSeed(issuerJWT, string(issuerUserSeed)), nats.Name("soulidentity-issuer")))
	if err != nil {
		return fail(fmt.Errorf("issuer connect: %w", err))
	}
	issuer, err := callout.NewIssuer(v, store, "auth/issuer", 15*time.Minute, string(calloutSeed), logger)
	if err != nil {
		return fail(fmt.Errorf("issuer: %w", err))
	}
	if _, err := issuer.Start(ncIssuer); err != nil {
		return fail(fmt.Errorf("issuer start: %w", err))
	}
	if err := ncIssuer.Flush(); err != nil {
		return fail(fmt.Errorf("issuer flush: %w", err))
	}

	// --- admin acts through the public client (Inventory 13–14) ---
	ncAdmin, err := r.track(r.Connect(nats.UserJWTAndSeed(opsJWT, opsSeed), nats.Name("ops")))
	if err != nil {
		return fail(fmt.Errorf("admin connect: %w", err))
	}
	admin := siclient.New(ncAdmin, realmPub, "ops")
	if _, err := admin.ImportKey("realm", siclient.KindNATSAccountSigningKey, string(realmSKSeed), realmPub, ""); err != nil {
		return fail(fmt.Errorf("import realm signing key: %w", err))
	}
	if _, err := admin.ImportKey("auth/issuer", siclient.KindNATSAccountSigningKey, string(authSKSeed), authPub, ""); err != nil {
		return fail(fmt.Errorf("import auth signing key: %w", err))
	}
	created, err := admin.CreateToken(realmPub, r.Persona, "rig", 0)
	if err != nil {
		return fail(fmt.Errorf("create token: %w", err))
	}
	sentinel, err := admin.MintSentinel()
	if err != nil {
		return fail(fmt.Errorf("mint sentinel: %w", err))
	}
	sentinelPath := filepath.Join(dir, "sentinel.creds")
	if err := os.WriteFile(sentinelPath, []byte(sentinel.Creds), 0o600); err != nil {
		return fail(fmt.Errorf("write sentinel: %w", err))
	}

	r.Admin = admin
	r.Token = created.Token
	r.Digest = created.Digest
	r.SentinelPath = sentinelPath
	return r, nil
}

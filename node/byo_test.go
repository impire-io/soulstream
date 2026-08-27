package node

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"

	"github.com/impire-io/soulstream/ceremony"
)

// byoRig is the substrate's side of the boundary: an operator-mode
// nats-server stood up from a CONFIG FILE (the shape a real operator
// runs), whose operator "applies the kit faithfully" — authoring the
// two account JWTs from nothing but the ceremony's public keys, exactly
// what the kit document carries. Its master keys never reach soulstream;
// the test asserts that at the end (spec 010 SC-002).
type byoRig struct {
	srv      *natsserver.Server
	url      string
	authPub  string
	realmPub string

	// The operator's own secrets, kept to prove they never land in the
	// state dir.
	operatorSeed []byte
	realmSeed    []byte
}

// startByoRig applies the kit: everything it reads from st is a public
// key the kit renders. omitWorkloadSK models a partially applied kit
// (spec 010 SC-003).
func startByoRig(t *testing.T, st *ceremony.State, omitWorkloadSK bool) *byoRig {
	t.Helper()
	opKP, _ := nkeys.CreateOperator()
	opPub, _ := opKP.PublicKey()
	opSeed, _ := opKP.Seed()
	opClaims := jwt.NewOperatorClaims(opPub)
	opClaims.Name = "byo-rig-operator"
	opJWT, err := opClaims.Encode(opKP)
	if err != nil {
		t.Fatalf("rig operator jwt: %v", err)
	}
	sysKP, _ := nkeys.CreateAccount()
	sysPub, _ := sysKP.PublicKey()
	sysClaims := jwt.NewAccountClaims(sysPub)
	sysClaims.Name = "SYS"
	sysJWT, err := sysClaims.Encode(opKP)
	if err != nil {
		t.Fatalf("rig sys jwt: %v", err)
	}

	authKP, _ := nkeys.CreateAccount()
	authPub, _ := authKP.PublicKey()
	realmKP, _ := nkeys.CreateAccount()
	realmPub, _ := realmKP.PublicKey()
	realmSeed, _ := realmKP.Seed()

	// Kit §1, the AUTH account: the declared issuer user, the realm as
	// the one allowed account, the declared callout curve key, the
	// declared signing key.
	authClaims := jwt.NewAccountClaims(authPub)
	authClaims.Name = "soulstream-" + st.Realm + "-auth"
	authClaims.SigningKeys.Add(st.AuthSigningPub)
	authClaims.EnableExternalAuthorization(st.IssuerUserPub)
	authClaims.Authorization.AllowedAccounts.Add(realmPub)
	authClaims.Authorization.XKey = st.CalloutPub
	authJWT, err := authClaims.Encode(opKP)
	if err != nil {
		t.Fatalf("rig auth jwt: %v", err)
	}

	// Kit §1, the realm account: JetStream, the plain workload signing
	// key, the scoped signing key carrying the persona scope — rendered
	// from the same source the kit prints.
	realmClaims := jwt.NewAccountClaims(realmPub)
	realmClaims.Name = "soulstream-" + st.Realm
	realmClaims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: -1, DiskStorage: -1, Streams: -1, Consumer: -1,
	}
	if !omitWorkloadSK {
		realmClaims.SigningKeys.Add(st.WorkloadSigningPub)
	}
	scopePub, scopeSub := ceremony.PersonaScopeAllows()
	scope := jwt.NewUserScope()
	scope.Key = st.RealmSigningPub
	scope.Role = "soulstream-user"
	scope.Template = jwt.UserPermissionLimits{Permissions: jwt.Permissions{
		Pub: jwt.Permission{Allow: scopePub},
		Sub: jwt.Permission{Allow: scopeSub},
	}}
	realmClaims.SigningKeys.AddScopedSigner(scope)
	realmJWT, err := realmClaims.Encode(opKP)
	if err != nil {
		t.Fatalf("rig realm jwt: %v", err)
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
`, opJWT, sysPub, sysPub, sysJWT, authPub, authJWT, realmPub, realmJWT, t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "server.conf")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("rig config: %v", err)
	}
	opts, err := natsserver.ProcessConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("rig options: %v", err)
	}
	opts.NoLog, opts.NoSigs = true, true
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("rig server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("rig server not ready")
	}
	t.Cleanup(srv.Shutdown)
	return &byoRig{srv: srv, url: srv.ClientURL(), authPub: authPub, realmPub: realmPub,
		operatorSeed: opSeed, realmSeed: realmSeed}
}

// TestBYOFounding is spec 010 SC-001 + SC-002: the full self-hosted
// founding against a server soulstream did not configure, M1.1
// semantics through the callout, restart on the same state, and the
// custody audit.
func TestBYOFounding(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.GenerateBYO(ceremony.FlavourSelfHosted, "nats://placeholder:4222", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MCPListen = "127.0.0.1:0"
	st.SignInEnabled = false // no browser in this gate; port hygiene as in shellplane_test
	st.HelmEnabled = false

	// The operator applies the kit (reading only publics), then the
	// substrate URL and the hand-back arrive.
	rig := startByoRig(t, st, false)
	st.BYOURL = rig.url
	if err := st.Save(dir); err != nil {
		t.Fatalf("save phase 1: %v", err)
	}
	st.AuthPub = rig.authPub
	st.RealmPub = rig.realmPub
	if err := st.MintBYOUsers(); err != nil {
		t.Fatalf("mint users: %v", err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save phase 2: %v", err)
	}

	// The probes pass on a faithfully applied kit.
	if err := ProbeSubstrate(st, dir); err != nil {
		t.Fatalf("probe: %v", err)
	}

	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if n.URL() != rig.url {
		t.Fatalf("node url = %s, want the substrate %s", n.URL(), rig.url)
	}
	token, err := Found(n, st, dir)
	if err != nil {
		t.Fatalf("found: %v (audit: %s)", err, audit.String())
	}
	if err := SmokeAdmission(st.BYOURL, ceremony.SentinelPath(dir), token); err != nil {
		t.Fatalf("smoke: %v (audit: %s)", err, audit.String())
	}

	// M1.1 observation (a): the persona is server-asserted and scoped to
	// its own prefix — on a server soulstream does not run.
	nc, err := nats.Connect(rig.url,
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("token lane refused: %v (audit: %s)", err, audit.String())
	}
	persona, account, allow := principal(t, nc)
	nc.Close()
	if persona != ceremony.FoundingPersona {
		t.Fatalf("persona = %q, want %q", persona, ceremony.FoundingPersona)
	}
	if account != rig.realmPub {
		t.Fatalf("account = %q, want the rig's realm account", account)
	}
	for _, subj := range allow {
		if strings.Contains(subj, "sign.record") && !strings.Contains(subj, persona) {
			t.Fatalf("sign.record grant %q escapes the persona prefix", subj)
		}
	}
	// Garbage refusal is audited on our side (D20: reasons live here).
	if !strings.Contains(audit.String(), "REFUSED") {
		t.Fatalf("garbage refusal missing from the audit log: %s", audit.String())
	}
	n.Stop()

	// The state dir is a founded BYO realm: verify, count, restart.
	got, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify founded: %v", err)
	}
	if !ceremony.Founded(dir) {
		t.Fatal("sentinel missing — founding did not complete")
	}
	if n := got.ArtifactCount(); n != 16 {
		t.Fatalf("artifact count = %d, want 16", n)
	}
	n2, err := Start(Config{StateDir: dir, State: got, AuditWriter: audit})
	if err != nil {
		t.Fatalf("restart on founded state: %v", err)
	}
	if _, err := n2.Ops().Status(); err != nil {
		t.Fatalf("status after restart: %v", err)
	}
	n2.Stop()

	// SC-002, the custody audit: no master material in the state dir —
	// by name and by content.
	for _, forbidden := range []string{
		"keys/operator.nk", "keys/sys.nk", "keys/sys.jwt",
		"keys/auth.nk", "keys/auth.jwt", "keys/realm.nk", "keys/realm.jwt",
	} {
		if _, err := os.Stat(filepath.Join(dir, forbidden)); err == nil {
			t.Errorf("%s exists — master material crossed the boundary", forbidden)
		}
	}
	for _, rel := range stateDirFiles(t, dir) {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if strings.Contains(string(content), string(rig.operatorSeed)) {
			t.Errorf("%s carries the rig's OPERATOR seed", rel)
		}
		if strings.Contains(string(content), string(rig.realmSeed)) {
			t.Errorf("%s carries the rig's realm MASTER seed", rel)
		}
	}
}

// TestBYOConfAuthRefused is spec 010 SC-003's first half: a server not
// in operator mode draws the named refusal, before anything else runs.
func TestBYOConfAuthRefused(t *testing.T) {
	srv, err := natsserver.NewServer(&natsserver.Options{
		Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true,
	})
	if err != nil {
		t.Fatalf("plain server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("plain server not ready")
	}
	defer srv.Shutdown()

	dir := t.TempDir()
	st, _ := ceremony.GenerateBYO(ceremony.FlavourSelfHosted, srv.ClientURL(), "home")
	authKP, _ := nkeys.CreateAccount()
	realmKP, _ := nkeys.CreateAccount()
	st.AuthPub, _ = authKP.PublicKey()
	st.RealmPub, _ = realmKP.PublicKey()
	if err := st.MintBYOUsers(); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	err = ProbeSubstrate(st, dir)
	if err == nil || !strings.Contains(err.Error(), "not running operator mode") {
		t.Fatalf("conf-auth server not refused by name: %v", err)
	}
}

// TestBYOPartialKit is SC-003's second half: a kit applied without the
// workload signing key draws a refusal naming that kit item.
func TestBYOPartialKit(t *testing.T) {
	dir := t.TempDir()
	st, _ := ceremony.GenerateBYO(ceremony.FlavourSelfHosted, "nats://placeholder:4222", "home")
	rig := startByoRig(t, st, true) // the operator skipped the workload --sk
	st.BYOURL = rig.url
	st.AuthPub = rig.authPub
	st.RealmPub = rig.realmPub
	if err := st.MintBYOUsers(); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	err := ProbeSubstrate(st, dir)
	if err == nil || !strings.Contains(err.Error(), "workload signing key") {
		t.Fatalf("partial kit not refused by its item's name: %v", err)
	}
}

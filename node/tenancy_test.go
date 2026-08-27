package node

import (
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream/ceremony"
)

// TestTenantsInTheHouse is spec 012's gate: the tenancy op family
// reachable through the running house (SystemConn wired, D35/D47), a
// tenant created over the sealed surface admitting a token-lane person
// as a USABLE user — and, the persistence clause, the tenant still
// admitting after the node stops and starts again on the same state
// directory (the dir resolver keeping what the runtime taught it,
// including AUTH's amended allowed_accounts).
func TestTenantsInTheHouse(t *testing.T) {
	dir := t.TempDir()

	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Only the planes this gate exercises: server + identity.
	st.MemoryEnabled, st.MCPEnabled, st.SignInEnabled, st.HelmEnabled = false, false, false, false
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := Found(n, st, dir); err != nil {
		t.Fatalf("found: %v", err)
	}

	// A tenant is born through the ops surface the CLI drives.
	born := time.Now()
	rec, err := n.Ops().AccountCreate("acme")
	if err != nil {
		t.Fatalf("account create through the house: %v (audit: %s)", err, audit.String())
	}

	// A person in the tenant: token issued over the same surface, then
	// sentinel + token through the callout — usable, not merely admitted.
	created, err := n.Ops().CreateToken(rec.Account, "alice", "tenancy-gate", 0)
	if err != nil {
		t.Fatalf("token for the tenant: %v", err)
	}
	dialTenant := func(url string) (*nats.Conn, error) {
		return nats.Connect(url,
			nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(created.Token),
			nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	}
	prove := func(url, phase string) {
		nc, err := dialTenant(url)
		if err != nil {
			t.Fatalf("%s: admission into the tenant: %v (audit: %s)", phase, err, audit.String())
		}
		defer nc.Close()
		sub, err := nc.SubscribeSync("SOULSTREAM.tenancy.gate")
		if err != nil {
			t.Fatalf("%s: tenant user cannot subscribe (inert): %v", phase, err)
		}
		if err := nc.Publish("SOULSTREAM.tenancy.gate", []byte("alive")); err != nil {
			t.Fatal(err)
		}
		if _, err := sub.NextMsg(2 * time.Second); err != nil {
			t.Fatalf("%s: round trip in the tenant: %v", phase, err)
		}
	}
	prove(n.URL(), "first run")
	t.Logf("account create -> usable admission in %v", time.Since(born))

	// The persistence clause: stop the house, start it again on the same
	// state, and the tenant is still there — resolver dir intact, AUTH
	// still listing it, the vault binding still minting.
	n.Stop()
	st2, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify after restart: %v", err)
	}
	n2, err := Start(Config{StateDir: dir, State: st2, AuditWriter: audit})
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	defer n2.Stop()
	prove(n2.URL(), "after restart")

	// The record survived too: the op surface still resolves the tenant.
	got, err := n2.Ops().AccountResolve("acme")
	if err != nil || got.Account != rec.Account {
		t.Fatalf("resolve after restart: %v %+v", err, got)
	}

	// And suspension bites through the same surface.
	if _, err := n2.Ops().AccountSuspend("acme"); err != nil {
		t.Fatal(err)
	}
	if ncS, err := dialTenant(n2.URL()); err == nil {
		ncS.Close()
		t.Fatal("suspended tenant admitted a connection")
	}
	if _, err := n2.Ops().AccountResume("acme"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit.String(), "accounts.create") {
		t.Fatal("accounts.create left no audit line")
	}
}

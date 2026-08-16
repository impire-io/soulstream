package synadia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nkeys"
)

// cloudStub is a stateful stand-in for the control-plane API: enough of
// its surface for the driver's sequence, remembering what was created
// so idempotence is observable. The live Cloud run stays the manual
// runbook (quickstart.md — the Entra-lane precedent).
type cloudStub struct {
	mux      *http.ServeMux
	accounts map[string]*stubAccount // id → account
	order    []string                // account ids in creation order
	callouts map[string]string       // callout id → control account id
	creates  int                     // POSTs that made something new

	// The lossy channel (measured live 2026-08-16: the private-link
	// tunnel cycling mid-request): flakes[method+" "+lastSegment]
	// counts 500s to serve before letting that endpoint through;
	// failGroups[name] does the same per signing-key group create.
	flakes     map[string]int
	failGroups map[string]int
}

// handler wraps the mux with the flake middleware.
func (s *cloudStub) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		key := r.Method + " " + parts[len(parts)-1]
		if s.flakes[key] > 0 {
			s.flakes[key]--
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"nats: timeout"}`)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

type stubAccount struct {
	id, name, pub string
	groups        []stubGroup
	users         []stubUser
}

type stubGroup struct {
	id, name     string
	programmatic bool
}

type stubUser struct{ id, name string }

func newCloudStub(t *testing.T) *cloudStub {
	t.Helper()
	s := &cloudStub{accounts: map[string]*stubAccount{}, callouts: map[string]string{},
		flakes: map[string]int{}, failGroups: map[string]int{}}
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	item := func(a *stubAccount) map[string]any {
		return map[string]any{"id": a.id, "name": a.name,
			"account_public_key": a.pub, "is_system_account": false}
	}

	mux.HandleFunc("GET /api/core/beta/teams", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"items": []any{map[string]any{"id": "t1", "name": "impire"}}})
	})
	mux.HandleFunc("GET /api/core/beta/teams/t1/systems", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"items": []any{map[string]any{"id": "s1", "name": "dev-sys"}}})
	})
	mux.HandleFunc("GET /api/core/beta/systems/s1/accounts", func(w http.ResponseWriter, _ *http.Request) {
		items := []any{}
		for _, id := range s.order {
			items = append(items, item(s.accounts[id]))
		}
		writeJSON(w, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/core/beta/systems/s1/accounts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		kp, _ := nkeys.CreateAccount()
		pub, _ := kp.PublicKey()
		a := &stubAccount{id: fmt.Sprintf("a%d", len(s.accounts)+1), name: req.Name, pub: pub}
		s.accounts[a.id] = a
		s.order = append(s.order, a.id)
		s.creates++
		writeJSON(w, item(a))
	})
	mux.HandleFunc("GET /api/core/beta/accounts/{id}/account-sk-groups", func(w http.ResponseWriter, r *http.Request) {
		a := s.accounts[r.PathValue("id")]
		items := []any{}
		for _, g := range a.groups {
			items = append(items, map[string]any{"id": g.id, "name": g.name, "programmatic": g.programmatic})
		}
		writeJSON(w, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/core/beta/accounts/{id}/account-sk-groups", func(w http.ResponseWriter, r *http.Request) {
		a := s.accounts[r.PathValue("id")]
		var req struct {
			Name         string `json:"name"`
			Programmatic bool   `json:"programmatic"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if s.failGroups[req.Name] > 0 {
			s.failGroups[req.Name]--
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"nats: timeout"}`)
			return
		}
		g := stubGroup{id: fmt.Sprintf("%s-g%d", a.id, len(a.groups)+1),
			name: req.Name, programmatic: req.Programmatic}
		a.groups = append(a.groups, g)
		s.creates++
		resp := map[string]any{"id": g.id, "name": g.name, "programmatic": g.programmatic}
		if req.Programmatic {
			kp, _ := nkeys.CreateAccount()
			seed, _ := kp.Seed()
			resp["seed"] = string(seed)
		}
		writeJSON(w, resp)
	})
	mux.HandleFunc("POST /api/core/beta/systems/s1/auth-callout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ControlAccount string `json:"control_account"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		// Re-enabling draws a persistent 500 on the real platform
		// (measured 2026-08-16) — the driver must be list-first, never
		// enable-first, or resumes die here.
		if len(s.callouts) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"an unexpected error occurred"}`)
			return
		}
		s.callouts["c1"] = req.ControlAccount
		s.creates++
		writeJSON(w, map[string]any{"id": "c1", "control_account_id": req.ControlAccount, "system_id": "s1"})
	})
	mux.HandleFunc("GET /api/core/beta/systems/s1/auth-callout", func(w http.ResponseWriter, _ *http.Request) {
		items := []any{}
		for id, ctrl := range s.callouts {
			items = append(items, map[string]any{"id": id, "control_account_id": ctrl, "system_id": "s1"})
		}
		writeJSON(w, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/core/beta/auth-callout/c1/target-accounts", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{})
	})
	mux.HandleFunc("POST /api/core/beta/auth-callout/c1/users", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{})
	})
	mux.HandleFunc("GET /api/core/beta/accounts/{id}/nats-users", func(w http.ResponseWriter, r *http.Request) {
		a := s.accounts[r.PathValue("id")]
		items := []any{}
		for _, u := range a.users {
			items = append(items, map[string]any{"id": u.id, "name": u.name})
		}
		writeJSON(w, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/core/beta/accounts/{id}/nats-users", func(w http.ResponseWriter, r *http.Request) {
		a := s.accounts[r.PathValue("id")]
		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		u := stubUser{id: fmt.Sprintf("%s-u%d", a.id, len(a.users)+1), name: req.Name}
		a.users = append(a.users, u)
		s.creates++
		writeJSON(w, map[string]any{"id": u.id, "name": u.name})
	})
	mux.HandleFunc("POST /api/core/beta/nats-users/{id}/creds", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "-----BEGIN NATS USER JWT-----\nstub\n------END NATS USER JWT------\n")
	})
	s.mux = mux
	return s
}

func (s *cloudStub) byName(name string) *stubAccount {
	for _, a := range s.accounts {
		if a.name == name {
			return a
		}
	}
	return nil
}

// TestSetupSequence: one run creates the whole account half — the two
// accounts, three programmatic groups (the realm's scoped + plain, the
// AUTH signing group), the on-demand group, the issuer user — and hands
// back the pubs, the once-returned seeds, and the downloaded creds.
func TestSetupSequence(t *testing.T) {
	stub := newCloudStub(t)
	ts := httptest.NewServer(stub.handler())
	defer ts.Close()

	res, err := Setup(context.Background(), Config{
		BaseURL: ts.URL, Token: "uat_test", System: "dev-sys", Realm: "home",
	}, []string{"identity.status"}, []string{"_INBOX.>"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	realmAcct := stub.byName("soulstream-home")
	authAcct := stub.byName("soulstream-home-auth")
	if realmAcct == nil || authAcct == nil {
		t.Fatal("the two accounts were not created")
	}
	if res.RealmAccountPub != realmAcct.pub || res.AuthAccountPub != authAcct.pub {
		t.Fatal("handed-back pubs do not match the platform's accounts")
	}
	if len(res.RealmScopedSeed) == 0 || len(res.WorkloadSeed) == 0 || len(res.AuthSigningSeed) == 0 {
		t.Fatal("a once-returned seed is missing from the result")
	}
	if !strings.Contains(string(res.IssuerCreds), "NATS USER JWT") {
		t.Fatalf("issuer creds not downloaded: %q", res.IssuerCreds)
	}
	var names []string
	for _, g := range realmAcct.groups {
		names = append(names, fmt.Sprintf("%s(prog=%v)", g.name, g.programmatic))
	}
	got := strings.Join(names, " ")
	if !strings.Contains(got, "soulstream-user(prog=true)") || !strings.Contains(got, "soulstream-workload(prog=true)") {
		t.Fatalf("realm account groups wrong: %s", got)
	}
	if len(authAcct.users) != 1 || authAcct.users[0].name != "soulstream-identity-issuer" {
		t.Fatalf("issuer user wrong: %+v", authAcct.users)
	}
}

// TestSetupIdempotence: a resumed run holding the seeds creates nothing
// new; a resumed run that LOST a seed is refused by name — the platform
// returns a programmatic seed exactly once (journey 0038, measured).
func TestSetupIdempotence(t *testing.T) {
	stub := newCloudStub(t)
	ts := httptest.NewServer(stub.handler())
	defer ts.Close()
	cfg := Config{BaseURL: ts.URL, Token: "uat_test", System: "dev-sys", Realm: "home"}

	// The system resolves by id as well as by name — an operator pasting
	// the id from the console must not be refused.
	cfg.System = "s1"
	first, err := Setup(context.Background(), cfg, []string{"p"}, []string{"s"})
	if err != nil {
		t.Fatalf("first run (system by id): %v", err)
	}
	creates := stub.creates

	cfg.Existing = Existing{
		RealmScopedSeed: first.RealmScopedSeed,
		WorkloadSeed:    first.WorkloadSeed,
		AuthSigningSeed: first.AuthSigningSeed,
	}
	second, err := Setup(context.Background(), cfg, []string{"p"}, []string{"s"})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if stub.creates != creates {
		t.Fatalf("resumed run created %d new resources", stub.creates-creates)
	}
	if string(second.RealmScopedSeed) != string(first.RealmScopedSeed) {
		t.Fatal("resumed run lost the custodied seed")
	}

	cfg.Existing = Existing{}
	_, err = Setup(context.Background(), cfg, []string{"p"}, []string{"s"})
	if err == nil || !strings.Contains(err.Error(), "exactly once") {
		t.Fatalf("lost seed not refused by name: %v", err)
	}
}

// TestSetupRefusals: no token and no system are named refusals before
// any request is made.
func TestSetupRefusals(t *testing.T) {
	if _, err := Setup(context.Background(), Config{System: "x"}, nil, nil); err == nil ||
		!strings.Contains(err.Error(), "SOULSTREAM_SYNADIA_TOKEN") {
		t.Fatalf("missing token: %v", err)
	}
	if _, err := Setup(context.Background(), Config{Token: "uat_x"}, nil, nil); err == nil ||
		!strings.Contains(err.Error(), "--synadia-system") {
		t.Fatalf("missing system: %v", err)
	}
}

// TestSetupSurvivesFlakyChannel: the measured BYON failure mode — the
// private-link tunnel cycling mid-request, ~50% of mutations drawing
// 500 "nats: timeout" — must not need a babysitter: one Setup call
// pushes through with bounded retries, and every once-returned seed
// reaches OnSeed.
func TestSetupSurvivesFlakyChannel(t *testing.T) {
	old := backoffBase
	backoffBase = time.Millisecond
	defer func() { backoffBase = old }()

	stub := newCloudStub(t)
	stub.flakes["POST accounts"] = 1
	stub.flakes["GET account-sk-groups"] = 1
	stub.flakes["POST auth-callout"] = 1
	stub.flakes["POST target-accounts"] = 1
	stub.flakes["POST nats-users"] = 1
	stub.flakes["POST creds"] = 2
	stub.failGroups[GroupWorkload] = 1
	ts := httptest.NewServer(stub.handler())
	defer ts.Close()

	seeds := map[string][]byte{}
	res, err := Setup(context.Background(), Config{
		BaseURL: ts.URL, Token: "uat_test", System: "dev-sys", Realm: "home",
		OnSeed: func(group string, seed []byte) error {
			seeds[group] = seed
			return nil
		},
	}, []string{"p"}, []string{"s"})
	if err != nil {
		t.Fatalf("setup through the flaky channel: %v", err)
	}
	if len(seeds) != 3 {
		t.Fatalf("OnSeed delivered %d seeds, want 3", len(seeds))
	}
	if string(seeds[GroupScoped]) != string(res.RealmScopedSeed) {
		t.Fatal("OnSeed's scoped seed differs from the result's")
	}
}

// TestSetupMidFailureKeepsSeed: the live 2026-08-16 incident, replayed —
// the scoped group's seed arrives, then the channel dies for good on
// the workload group. The seed must already be with OnSeed, and the
// resumed run must reuse the group instead of refusing or doubling it.
func TestSetupMidFailureKeepsSeed(t *testing.T) {
	old := backoffBase
	backoffBase = time.Millisecond
	defer func() { backoffBase = old }()

	stub := newCloudStub(t)
	stub.failGroups[GroupWorkload] = 999 // permanently dead this run
	ts := httptest.NewServer(stub.handler())
	defer ts.Close()

	seeds := map[string][]byte{}
	cfg := Config{BaseURL: ts.URL, Token: "uat_test", System: "dev-sys", Realm: "home",
		OnSeed: func(group string, seed []byte) error {
			seeds[group] = seed
			return nil
		}}
	_, err := Setup(context.Background(), cfg, []string{"p"}, []string{"s"})
	if err == nil {
		t.Fatal("setup succeeded through a dead channel")
	}
	if len(seeds[GroupScoped]) == 0 {
		t.Fatal("the scoped seed did not reach OnSeed before the failure — a mid-run death loses it")
	}

	// The channel heals; the resumed run holds the persisted seed.
	stub.failGroups[GroupWorkload] = 0
	cfg.Existing = Existing{RealmScopedSeed: seeds[GroupScoped]}
	res, err := Setup(context.Background(), cfg, []string{"p"}, []string{"s"})
	if err != nil {
		t.Fatalf("resumed run: %v", err)
	}
	if string(res.RealmScopedSeed) != string(seeds[GroupScoped]) {
		t.Fatal("resumed run lost the custodied scoped seed")
	}
	scoped := 0
	for _, g := range stub.byName("soulstream-home").groups {
		if g.name == GroupScoped {
			scoped++
		}
	}
	if scoped != 1 {
		t.Fatalf("scoped group created %d times, want exactly 1", scoped)
	}
}

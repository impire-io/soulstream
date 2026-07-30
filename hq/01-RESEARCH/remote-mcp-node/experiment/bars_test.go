package researchnode

// The pre-registered bars of hq/01-RESEARCH/remote-mcp-node, run against the
// prototype node. Bar 4 (a hosted no-install client) cannot run in-process
// and is out of this file's scope.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulstream/identity"
	"github.com/impire-io/soulstream/topic"
)

// bearerRT injects a freshly minted bearer on every request — the shape of a
// client that refreshes its token; a fixed function models a static token.
type bearerRT struct {
	mint func() string
}

func (rt bearerRT) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+rt.mint())
	return http.DefaultTransport.RoundTrip(r)
}

func mcpSession(t *testing.T, url string, mint func() string) *mcp.ClientSession {
	t.Helper()
	sess, err := dialMCP(t, url, mint)
	if err != nil {
		t.Fatalf("mcp connect: %v", err)
	}
	return sess
}

func dialMCP(t *testing.T, url string, mint func() string) (*mcp.ClientSession, error) {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "bar-test", Version: "0.0.1"}, nil)
	sess, err := c.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: bearerRT{mint: mint}},
	}, nil)
	if err == nil {
		t.Cleanup(func() { _ = sess.Close() })
	}
	return sess, err
}

func callText(t *testing.T, sess *mcp.ClientSession, tool string, args map[string]any) string {
	t.Helper()
	res, err := sess.CallTool(t.Context(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", tool, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %+v", tool, res.Content)
	}
	txt, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("call %s: not text content: %T", tool, res.Content[0])
	}
	return txt.Text
}

// TestBar1AdmissionByBadge: both lanes admit through the node with
// attribution == the token's principal; revoked and garbage tokens refuse at
// the NATS edge with no realm write; the node's config is a URL, a realm
// name, and (finding) the operator-mode sentinel.
func TestBar1AdmissionByBadge(t *testing.T) {
	r := startRig(t, 2*time.Minute)

	// Probe [recorded]: does the BARE token option work in operator mode,
	// or is the sentinel required? This decides the honest reading of the
	// bar's "nothing credential-shaped".
	if nc, err := nats.Connect(r.url, nats.Token(r.apiToken)); err == nil {
		nc.Close()
		t.Log("bare-token probe: ADMITTED without sentinel (operator mode)")
	} else {
		t.Logf("bare-token probe: refused without sentinel: %v", err)
	}

	node := New(Config{NATSURL: r.url, Realm: "proof", SentinelPath: r.sentinelPath})
	web := httptest.NewServer(node)
	t.Cleanup(web.Close)

	// API-token lane.
	apiSess := mcpSession(t, web.URL, func() string { return r.apiToken })
	who := callText(t, apiSess, "whoami", nil)
	if who != "daan-ext@"+r.accPub {
		t.Fatalf("api-lane principal: got %q", who)
	}
	path := callText(t, apiSess, "start_topic", map[string]any{
		"name": "bar-one", "subject_matter": "admission by badge",
	})
	if _, err := apiSess.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "post_turn", Arguments: map[string]any{"path": path, "body": "posted by the api-token lane"},
	}); err != nil {
		t.Fatalf("api-lane post: %v", err)
	}

	// OIDC lane: an Entra-shaped token whose role names the declared team.
	const oid = "aaaaaaaa-1111-2222-3333-bbbbbbbbcccc"
	claims := r.stub.Claims(oid, "acme")
	claims["preferred_username"] = "daan@example.com"
	entraTok, err := r.stub.Token(claims)
	if err != nil {
		t.Fatalf("stub token: %v", err)
	}
	oidcSess := mcpSession(t, web.URL, func() string { return entraTok })
	who = callText(t, oidcSess, "whoami", nil)
	if who != oid+"@"+r.accPub {
		t.Fatalf("oidc-lane principal: got %q", who)
	}
	if _, err := oidcSess.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "post_turn", Arguments: map[string]any{"path": path, "body": "posted by the oidc lane"},
	}); err != nil {
		t.Fatalf("oidc-lane post: %v", err)
	}

	// Attribution is read from the REALM, not from the node's word.
	rc, _ := r.readerRealm(t)
	mt, err := topic.Open(rc, path).Materialise(t.Context())
	if err != nil {
		t.Fatalf("materialise: %v", err)
	}
	if len(mt.Contributions) != 2 {
		t.Fatalf("want 2 contributions, got %d", len(mt.Contributions))
	}
	gotAuthors := map[string]bool{}
	for _, c := range mt.Contributions {
		gotAuthors[c.Author] = true
	}
	if !gotAuthors["daan-ext"] || !gotAuthors[oid] {
		t.Fatalf("attribution != principals: %v", gotAuthors)
	}

	// Refusals: garbage, then revoked — no server for the session, and the
	// realm sees no third write.
	if _, err := dialMCP(t, web.URL, func() string { return "sit_" + strings.Repeat("00", 32) }); err == nil {
		t.Fatal("garbage token yielded an MCP session")
	} else {
		t.Logf("garbage token refused: %v", err)
	}
	if err := r.admin.RevokeToken(r.apiDigest); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	// A fresh CONNECT with the revoked token must refuse. The original
	// node's pool entry is still live — revocation reaches an existing
	// connection only at the TTL disconnect (the known bound) — so the
	// fresh-connect proof runs on a second node.
	node2 := New(Config{NATSURL: r.url, Realm: "proof", SentinelPath: r.sentinelPath})
	web2 := httptest.NewServer(node2)
	t.Cleanup(web2.Close)
	if _, err := dialMCP(t, web2.URL, func() string { return r.apiToken }); err == nil {
		t.Fatal("revoked token yielded an MCP session on a fresh node")
	} else {
		t.Logf("revoked token refused: %v", err)
	}

	mt, err = topic.Open(rc, path).Materialise(t.Context())
	if err != nil {
		t.Fatalf("re-materialise: %v", err)
	}
	if len(mt.Contributions) != 2 {
		t.Fatalf("a refused session wrote to the realm: %d contributions", len(mt.Contributions))
	}
	if !strings.Contains(r.audit.String(), "callout REFUSED") {
		t.Fatal("refusals not in the audit log")
	}
	t.Logf("bar 1 PASS: api-lane persona=daan-ext, oidc-lane persona=%s (oid), refusals draw no realm write", oid)
}

// TestBar2CustodyEndToEnd: a turn posted through the node verifies from a
// keyring built from ONE keys.public answer; without the keyring it reads
// unknown-key (the verdict is earned); the node held no key material.
func TestBar2CustodyEndToEnd(t *testing.T) {
	r := startRig(t, 2*time.Minute)
	node := New(Config{NATSURL: r.url, Realm: "proof", SentinelPath: r.sentinelPath})
	web := httptest.NewServer(node)
	t.Cleanup(web.Close)

	sess := mcpSession(t, web.URL, func() string { return r.apiToken })
	path := callText(t, sess, "start_topic", map[string]any{
		"name": "bar-two", "subject_matter": "custody through the node",
	})
	callText(t, sess, "post_turn", map[string]any{"path": path, "body": "sealed by the vault, not the node"})

	rc, ncReader := r.readerRealm(t)
	h := topic.Open(rc, path)

	// Negative control: no keyring → unknown-key.
	bare, err := h.Materialise(t.Context())
	if err != nil {
		t.Fatalf("materialise (no keyring): %v", err)
	}
	if bare.Announcement == nil || bare.Announcement.Sig != topic.SigUnknownKey {
		t.Fatalf("without a keyring the announcement must read unknown-key, got %+v", bare.Announcement)
	}

	// The directory read: ONE keys.public answer from the identity plane.
	pub, err := siclient.New(ncReader, r.accPub, "reader").PersonaPublicKey("daan-ext")
	if err != nil {
		t.Fatalf("directory read: %v", err)
	}
	h.UseKeyring(&identity.Keyring{Keys: map[string][]string{"daan-ext": {pub}}})
	mt, err := h.Materialise(t.Context())
	if err != nil {
		t.Fatalf("materialise: %v", err)
	}
	if mt.Announcement == nil || mt.Announcement.Sig != topic.SigVerified {
		t.Fatalf("announcement not verified: %+v", mt.Announcement)
	}
	for i, c := range mt.Contributions {
		if c.Sig != topic.SigVerified {
			t.Fatalf("contribution %d not verified: sig=%s", i, c.Sig)
		}
	}
	t.Logf("bar 2 PASS: unknown-key without keyring, SigVerified from one keys.public answer; node config held URL+realm+sentinel only")
}

// TestBar3ExpiryAndRevocation (5s TTL): a session outlives 3× the TTL on
// refreshed badges with at most in-flight calls failing per re-proof; a
// role-stripped fresh badge (revocation at the IdP) stops writes within 2×
// TTL of the strip.
func TestBar3ExpiryAndRevocation(t *testing.T) {
	const ttl = 5 * time.Second
	r := startRig(t, ttl)
	node := New(Config{NATSURL: r.url, Realm: "proof", SentinelPath: r.sentinelPath})
	web := httptest.NewServer(node)
	t.Cleanup(web.Close)

	const oid = "dddddddd-4444-5555-6666-eeeeeeeeffff"
	var stripped atomic.Bool
	var minted, mintedStripped atomic.Int64
	mint := func() string {
		var tok string
		var err error
		if stripped.Load() {
			tok, err = r.stub.Token(r.stub.Claims(oid)) // role removed at the IdP
			mintedStripped.Add(1)
		} else {
			tok, err = r.stub.Token(r.stub.Claims(oid, "acme"))
		}
		minted.Add(1)
		if err != nil {
			t.Errorf("stub token: %v", err)
		}
		return tok
	}

	sess := mcpSession(t, web.URL, mint)
	path := callText(t, sess, "start_topic", map[string]any{
		"name": "bar-three", "subject_matter": "expiry and revocation",
	})

	// Phase 1: keep writing across 3× TTL with refreshed badges.
	start := time.Now()
	var ok, failed int
	var failTimes []time.Duration
	for time.Since(start) < 3*ttl+2*time.Second {
		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		res, err := sess.CallTool(ctx, &mcp.CallToolParams{
			Name: "post_turn", Arguments: map[string]any{"path": path, "body": "tick"},
		})
		cancel()
		if err != nil || res.IsError {
			failed++
			failTimes = append(failTimes, time.Since(start).Round(100*time.Millisecond))
		} else {
			ok++
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("phase 1: %d ok, %d failed over %s (ttl %s); failures at %v", ok, failed, time.Since(start).Round(time.Second), ttl, failTimes)
	if ok < 10 {
		t.Fatalf("session did not keep working across 3×TTL: only %d successful writes", ok)
	}
	// "At most the single in-flight call failing per re-proof": ~3 TTL
	// disconnects expected; allow one in-flight failure each.
	if failed > 4 {
		t.Fatalf("too many failed calls across re-proofs: %d", failed)
	}

	// The session must end on a working note before the strip measurement.
	callText(t, sess, "post_turn", map[string]any{"path": path, "body": "still here"})

	// Phase 2: the IdP strips the role — the freshest badge is now bad.
	stripAt := time.Now()
	stripped.Store(true)
	deadline := time.Now().Add(6 * ttl)
	var lastOK, firstFail time.Duration
	bitten := false
	consecFails := 0
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		res, err := sess.CallTool(ctx, &mcp.CallToolParams{
			Name: "post_turn", Arguments: map[string]any{"path": path, "body": "post-strip tick"},
		})
		cancel()
		if err == nil && res.IsError {
			err = fmt.Errorf("tool error: %v", res.Content)
		}
		if err != nil {
			t.Logf("  +%s post_turn FAIL: %v", time.Since(stripAt).Round(100*time.Millisecond), err)
			consecFails++
			if consecFails == 1 {
				firstFail = time.Since(stripAt)
			}
			if consecFails >= 3 {
				bitten = true
				break
			}
		} else {
			t.Logf("  +%s post_turn ok", time.Since(stripAt).Round(100*time.Millisecond))
			consecFails = 0
			lastOK = time.Since(stripAt)
		}
		time.Sleep(500 * time.Millisecond)
	}
	auditPath := "bar3-audit.log"
	_ = os.WriteFile(auditPath, []byte(r.audit.String()), 0o600)
	t.Logf("full audit at %s; admissions=%d refusals=%d", auditPath,
		strings.Count(r.audit.String(), "callout ADMITTED"), strings.Count(r.audit.String(), "callout REFUSED"))
	if !bitten {
		a := r.audit.String()
		if len(a) > 3000 {
			a = a[len(a)-3000:]
		}
		t.Logf("minted=%d stripped=%d; audit tail:\n%s", minted.Load(), mintedStripped.Load(), a)
		t.Fatalf("revocation never bit: writes still succeeding %s after the strip (last ok %s)", time.Since(stripAt), lastOK)
	}
	t.Logf("phase 2: last successful write %s after strip, persistent failure from %s (bound 2×ttl = %s)", lastOK.Round(100*time.Millisecond), firstFail.Round(100*time.Millisecond), 2*ttl)
	if lastOK > 2*ttl {
		t.Fatalf("writes survived past 2×TTL after the strip: last ok at %s", lastOK)
	}
	if !strings.Contains(r.audit.String(), "no roles claim") {
		t.Fatal("role-stripped refusal missing from the audit log")
	}
}

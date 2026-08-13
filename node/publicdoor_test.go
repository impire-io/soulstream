package node

// The public-door gate (roadmap Phase 2, public mode; spec 005): with
// planes.door carrying public_url/auth_issuer/auth_audience, a hosted
// client that knows only the door's URL walks the full OAuth story —
// 401 challenge → resource metadata → AS discovery → DCR → code+PKCE →
// token — and the bearer admits an MCP session through the realm's own
// callout, persona named by the token's subject. The founding-token
// lane coexists untouched; the AS is the upstream contract's stand-in
// (rigtest), because the door is AS-agnostic by design — soulstream-idp is
// the intended default, not a dependency.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream-mcp/rigtest"

	"github.com/impire-io/soulstream/ceremony"
)

func TestPublicDoor(t *testing.T) {
	as, err := rigtest.NewASStub("soulstream-door")
	if err != nil {
		t.Fatalf("as stub: %v", err)
	}
	defer as.Close()

	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.DoorListen = "127.0.0.1:0"
	st.DoorPublicURL = "https://door.example.test" // the advertised name; fronting carries it
	st.DoorAuthIssuer = as.Issuer()
	st.DoorAuthAudience = as.Audience()
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	// The round-trip keeps the public fields (contracts/config.md).
	loaded, err := ceremony.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.DoorPublicURL != st.DoorPublicURL || loaded.DoorAuthIssuer != st.DoorAuthIssuer ||
		loaded.DoorAuthAudience != st.DoorAuthAudience {
		t.Fatalf("public door config did not survive the round-trip: %+v", loaded)
	}

	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer n.Stop()
	foundingToken, err := Found(n, st, dir)
	if err != nil {
		t.Fatalf("found: %v", err)
	}

	// --- The discovery walk, knowing only the door's URL.
	httpc := http.DefaultClient
	cold, err := httpc.Get(n.DoorURL())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, cold.Body)
	_ = cold.Body.Close()
	wwwAuth := cold.Header.Get("WWW-Authenticate")
	if cold.StatusCode != http.StatusUnauthorized || !strings.Contains(wwwAuth, "resource_metadata=") {
		t.Fatalf("cold request: want 401 naming resource_metadata, got %d %q", cold.StatusCode, wwwAuth)
	}
	mdResp, err := httpc.Get(n.DoorURL() + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatal(err)
	}
	var md struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(mdResp.Body).Decode(&md); err != nil {
		t.Fatalf("resource metadata: %v", err)
	}
	_ = mdResp.Body.Close()
	if md.Resource != st.DoorPublicURL {
		t.Fatalf("advertised resource %q, want %q", md.Resource, st.DoorPublicURL)
	}
	if len(md.AuthorizationServers) != 1 || md.AuthorizationServers[0] != as.Issuer() {
		t.Fatalf("advertised AS %v, want [%s]", md.AuthorizationServers, as.Issuer())
	}

	// --- The OAuth story against the advertised AS: DCR → code+PKCE →
	// token, exactly what a hosted MCP client performs.
	accessToken := oauthSignIn(t, md.AuthorizationServers[0], "visitor-7", "realm")

	// --- The bearer opens the door; whoami names the token's subject.
	dial := func(bearer string) (*mcp.ClientSession, error) {
		client := mcp.NewClient(&mcp.Implementation{Name: "soulstream-public-test", Version: "0.0.1"}, nil)
		transport := &mcp.StreamableClientTransport{
			Endpoint:   n.DoorURL(),
			HTTPClient: &http.Client{Transport: bearerRT{bearer: bearer}},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return client.Connect(ctx, transport, nil)
	}
	sess, err := dial(accessToken)
	if err != nil {
		t.Fatalf("mcp connect with OIDC bearer: %v (audit: %s)", err, audit.String())
	}
	defer func() { _ = sess.Close() }()
	ctx := context.Background()
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "soulstream_whoami"})
	if err != nil || res.IsError {
		t.Fatalf("whoami: %v (isError %v)", err, res != nil && res.IsError)
	}
	var whoami strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			whoami.WriteString(tc.Text)
		}
	}
	if !strings.Contains(whoami.String(), "visitor-7") {
		t.Fatalf("whoami does not name the OIDC subject: %s", whoami.String())
	}

	// The admission is attributed to the OIDC lane in the audit.
	if s := audit.String(); !strings.Contains(s, "lane=oidc") || !strings.Contains(s, "role=realm") {
		t.Fatalf("OIDC admission not attributed:\n%s", s)
	}

	// --- Coexistence and refusals.
	if sessOwner, err := dial(foundingToken); err != nil {
		t.Fatalf("founding token stopped working in public mode: %v", err)
	} else {
		_ = sessOwner.Close()
	}
	undeclared := oauthSignIn(t, as.Issuer(), "visitor-8", "marketing")
	if badSess, err := dial(undeclared); err == nil {
		_ = badSess.Close()
		t.Fatal("a token naming an undeclared role formed a session")
	}
	if badSess, err := dial("sit_" + strings.Repeat("00", 32)); err == nil {
		_ = badSess.Close()
		t.Fatal("a garbage bearer formed a session")
	}
}

// oauthSignIn walks DCR → authorize (PKCE S256) → token against the AS
// and returns the access token — the scripted hosted client.
func oauthSignIn(t *testing.T, issuer, oid, role string) string {
	t.Helper()
	httpc := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	var disc struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		RegistrationEndpoint  string `json:"registration_endpoint"`
	}
	dResp, err := httpc.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(dResp.Body).Decode(&disc); err != nil {
		t.Fatal(err)
	}
	_ = dResp.Body.Close()

	const redirect = "http://127.0.0.1:1/cb"
	regBody, _ := json.Marshal(map[string]any{"redirect_uris": []string{redirect}})
	rResp, err := httpc.Post(disc.RegistrationEndpoint, "application/json", strings.NewReader(string(regBody)))
	if err != nil {
		t.Fatal(err)
	}
	var reg struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(rResp.Body).Decode(&reg); err != nil {
		t.Fatal(err)
	}
	_ = rResp.Body.Close()

	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	q := url.Values{
		"response_type": {"code"}, "client_id": {reg.ClientID},
		"redirect_uri": {redirect}, "code_challenge": {challenge},
		"code_challenge_method": {"S256"}, "state": {"s"},
		"login_hint": {oid}, "roles_hint": {role},
	}
	aResp, err := httpc.Get(disc.AuthorizationEndpoint + "?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, aResp.Body)
	_ = aResp.Body.Close()
	loc, err := url.Parse(aResp.Header.Get("Location"))
	if err != nil || loc.Query().Get("code") == "" {
		t.Fatalf("authorize: no code in %q (%v)", aResp.Header.Get("Location"), err)
	}

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {loc.Query().Get("code")},
		"code_verifier": {verifier}, "redirect_uri": {redirect},
	}
	tResp, err := httpc.Post(disc.TokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tResp.Body).Decode(&tok); err != nil {
		t.Fatal(err)
	}
	_ = tResp.Body.Close()
	if tok.AccessToken == "" {
		t.Fatal("no access token")
	}
	return tok.AccessToken
}

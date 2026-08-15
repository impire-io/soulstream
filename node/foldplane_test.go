package node

// The folded-realm gate (spec 006): the single-binary story complete —
// `planes.fold` runs the bundled OIDC provider through soulstream-idp's
// public embed seam, public door mode defaults its AS at the bundled
// fold, and a browser user's passkey sign-in becomes an MCP session at
// the door with zero external services. The virtual authenticator is
// soulstream-idp's public authtest; everything else is the walk any hosted
// client performs.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire-io/soulstream-idp/authtest"

	"github.com/impire-io/soulstream/ceremony"
)

// sha256Sum is the PKCE S256 challenge of a verifier.
func sha256Sum(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// extractAttr pulls an id="..."-anchored value out of the login page.
func extractAttr(t *testing.T, page, id string) string {
	t.Helper()
	marker := `id="` + id + `" value="`
	i := strings.Index(page, marker)
	if i < 0 {
		t.Fatalf("no %s in page:\n%s", id, page)
	}
	rest := page[i+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}

func TestFoldedRealm(t *testing.T) {
	// The fold's issuer must be named at config time: reserve its port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	foldAddr := ln.Addr().String()
	_ = ln.Close()
	_, foldPort, _ := net.SplitHostPort(foldAddr)

	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MCPListen = "127.0.0.1:0"
	st.MCPPublicURL = "https://door.example.test"
	st.SignInEnabled = true
	st.SignInListen = foldAddr
	// Clear the issuer/audience so the derived defaults (localhost host,
	// the fold's own port) drive them — that is the wiring under test.
	st.SignInIssuer = ""
	st.SignInAudience = ""
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := ceremony.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SignInIssuer != "http://localhost:"+foldPort {
		t.Fatalf("fold issuer defaulted to %q, want http://localhost:%s", loaded.SignInIssuer, foldPort)
	}
	if loaded.MCPAuthIssuer != loaded.SignInIssuer || loaded.MCPAuthAudience != "soulstream-home" {
		t.Fatalf("the default wiring did not point the door at the bundled fold: issuer=%q audience=%q",
			loaded.MCPAuthIssuer, loaded.MCPAuthAudience)
	}

	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: loaded, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer n.Stop()
	foundingToken, err := Found(n, loaded, dir)
	if err != nil {
		t.Fatalf("found: %v", err)
	}
	if n.SignInURL() == "" {
		t.Fatal("fold enabled but no URL")
	}

	// --- The hosted client's discovery walk from the door alone.
	mdResp, err := http.Get(n.MCPURL() + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatal(err)
	}
	var md struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(mdResp.Body).Decode(&md); err != nil {
		t.Fatal(err)
	}
	_ = mdResp.Body.Close()
	if len(md.AuthorizationServers) != 1 || md.AuthorizationServers[0] != n.SignInURL() {
		t.Fatalf("the door advertises %v, want the bundled fold %s", md.AuthorizationServers, n.SignInURL())
	}

	// --- The founding invite (soulstream-idp M3: enrollment requires one;
	// the seam delivered it, the founding output prints it once).
	if n.FoldInvite() == "" {
		t.Fatal("the fold plane delivered no founding invite")
	}

	// --- The browser: DCR, then a passkey ENROLLMENT at the bundled
	// fold against the founding invite.
	token := foldSignIn(t, n.SignInURL(), ceremony.FoundingPersona, n.FoldInvite())

	// --- The bearer opens the door; whoami names the fold user.
	dial := func(bearer string) (*mcp.ClientSession, error) {
		client := mcp.NewClient(&mcp.Implementation{Name: "folded-realm-test", Version: "0.0.1"}, nil)
		transport := &mcp.StreamableClientTransport{
			Endpoint:   n.MCPURL(),
			HTTPClient: &http.Client{Transport: bearerRT{bearer: bearer}},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return client.Connect(ctx, transport, nil)
	}
	sess, err := dial(token)
	if err != nil {
		t.Fatalf("mcp connect with fold bearer: %v (audit: %s)", err, audit.String())
	}
	defer func() { _ = sess.Close() }()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: "soulstream_whoami"})
	if err != nil || res.IsError {
		t.Fatalf("whoami: %v (isError %v)", err, res != nil && res.IsError)
	}
	var whoami strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			whoami.WriteString(tc.Text)
		}
	}
	// The OIDC lane names the persona by the token's oid — the fold's
	// stable user id.
	if !strings.Contains(whoami.String(), "u-") {
		t.Fatalf("whoami does not name the fold user's oid: %s", whoami.String())
	}
	if s := audit.String(); !strings.Contains(s, "lane=oidc") || !strings.Contains(s, "role=realm") {
		t.Fatalf("fold admission not attributed:\n%s", s)
	}

	// --- Coexistence and refusal.
	if ownerSess, err := dial(foundingToken); err != nil {
		t.Fatalf("founding token stopped working beside the fold: %v", err)
	} else {
		_ = ownerSess.Close()
	}
	if badSess, err := dial("sit_" + strings.Repeat("00", 32)); err == nil {
		_ = badSess.Close()
		t.Fatal("garbage bearer formed a session")
	}
}

// foldSignIn is the scripted browser against the bundled fold: DCR →
// authorize → passkey ceremony (a non-empty invite enrolls) →
// callback → code → token.
func foldSignIn(t *testing.T, issuer, username, invite string) string {
	t.Helper()
	// The RP id is the issuer host — localhost since the fold default
	// (WebAuthn refuses a bare IP).
	auth, err := authtest.New("localhost", issuer)
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	httpc := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, _ []*http.Request) error {
		if req.URL.Host == "127.0.0.1:1" {
			return http.ErrUseLastResponse
		}
		return nil
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
	if disc.RegistrationEndpoint == "" {
		t.Fatal("the bundled fold advertises no DCR")
	}

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

	verifier := strings.Repeat("f", 43)
	sum := sha256Sum(verifier)
	q := url.Values{
		"response_type": {"code"}, "client_id": {reg.ClientID},
		"redirect_uri": {redirect}, "scope": {"openid"},
		"code_challenge": {sum}, "code_challenge_method": {"S256"}, "state": {"s"},
	}
	aResp, err := httpc.Get(disc.AuthorizationEndpoint + "?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(aResp.Body)
	_ = aResp.Body.Close()
	authReqID := aResp.Request.URL.Query().Get("authRequestID")
	csrf := extractAttr(t, string(page), "csrf")

	cq := url.Values{"authRequestID": {authReqID}, "csrf": {csrf}, "username": {username}, "invite": {invite}}
	// The console is the standalone deployment's surface (idp design
	// D31): the bundled sign-in plane serves the admin API, never the
	// HTML console — /admin answers like any path nobody claimed.
	if adminResp, err := httpc.Get(issuer + "/admin/"); err != nil {
		t.Fatal(err)
	} else {
		_ = adminResp.Body.Close()
		if adminResp.StatusCode != http.StatusNotFound {
			t.Fatalf("/admin/ on the bundled sign-in plane answered %d, want 404", adminResp.StatusCode)
		}
	}

	beginResp, err := httpc.Post(issuer+"/login/begin?"+cq.Encode(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var begin struct {
		CeremonyID string          `json:"ceremonyID"`
		Kind       string          `json:"kind"`
		Options    json.RawMessage `json:"options"`
	}
	if err := json.NewDecoder(beginResp.Body).Decode(&begin); err != nil {
		t.Fatal(err)
	}
	_ = beginResp.Body.Close()
	var waBody []byte
	if begin.Kind == "register" {
		waBody, err = auth.CreateResponse(begin.Options)
	} else {
		waBody, err = auth.GetResponse(begin.Options)
	}
	if err != nil {
		t.Fatal(err)
	}
	cq.Set("ceremonyID", begin.CeremonyID)
	finResp, err := httpc.Post(issuer+"/login/finish?"+cq.Encode(), "application/json", strings.NewReader(string(waBody)))
	if err != nil {
		t.Fatal(err)
	}
	var fin struct {
		Redirect string `json:"redirect"`
	}
	if err := json.NewDecoder(finResp.Body).Decode(&fin); err != nil {
		t.Fatal(err)
	}
	_ = finResp.Body.Close()
	target := fin.Redirect
	if !strings.HasPrefix(target, "http") {
		target = issuer + target
	}
	cb, err := httpc.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, cb.Body)
	_ = cb.Body.Close()
	loc, err := url.Parse(cb.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {loc.Query().Get("code")},
		"code_verifier": {verifier}, "redirect_uri": {redirect}, "client_id": {reg.ClientID},
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
		t.Fatal("no access token from the bundled fold")
	}
	return tok.AccessToken
}

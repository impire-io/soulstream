package node

// The folded-realm gate (spec 006): the single-binary story complete —
// `planes.fold` runs the bundled OIDC provider through soulfold's
// public embed seam, public door mode defaults its AS at the bundled
// fold, and a browser user's passkey sign-in becomes an MCP session at
// the door with zero external services. The virtual authenticator is
// soulfold's public authtest; everything else is the walk any hosted
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

	"github.com/impire-io/soulfold/authtest"

	"github.com/impire-io/soulnode/ceremony"
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

	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.DoorListen = "127.0.0.1:0"
	st.DoorPublicURL = "https://door.example.test"
	st.FoldEnabled = true
	st.FoldListen = foldAddr
	// DoorAuthIssuer, DoorAuthAudience, FoldIssuer, FoldAudience stay
	// empty: the default wiring is under test.
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := ceremony.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.FoldIssuer != "http://"+foldAddr {
		t.Fatalf("fold issuer defaulted to %q", loaded.FoldIssuer)
	}
	if loaded.DoorAuthIssuer != loaded.FoldIssuer || loaded.DoorAuthAudience != "soulnode-home" {
		t.Fatalf("the default wiring did not point the door at the bundled fold: issuer=%q audience=%q",
			loaded.DoorAuthIssuer, loaded.DoorAuthAudience)
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
	if n.FoldURL() == "" {
		t.Fatal("fold enabled but no URL")
	}

	// --- The hosted client's discovery walk from the door alone.
	mdResp, err := http.Get(n.DoorURL() + "/.well-known/oauth-protected-resource")
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
	if len(md.AuthorizationServers) != 1 || md.AuthorizationServers[0] != n.FoldURL() {
		t.Fatalf("the door advertises %v, want the bundled fold %s", md.AuthorizationServers, n.FoldURL())
	}

	// --- The browser: DCR, then a passkey sign-in at the bundled fold.
	token := foldSignIn(t, n.FoldURL(), ceremony.FoundingPersona)

	// --- The bearer opens the door; whoami names the fold user.
	dial := func(bearer string) (*mcp.ClientSession, error) {
		client := mcp.NewClient(&mcp.Implementation{Name: "folded-realm-test", Version: "0.0.1"}, nil)
		transport := &mcp.StreamableClientTransport{
			Endpoint:   n.DoorURL(),
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
// authorize → passkey ceremony (first touch enrolls) → callback → code
// → token.
func foldSignIn(t *testing.T, issuer, username string) string {
	t.Helper()
	auth, err := authtest.New("127.0.0.1", issuer)
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

	cq := url.Values{"authRequestID": {authReqID}, "csrf": {csrf}, "username": {username}}
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

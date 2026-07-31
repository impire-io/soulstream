package researchnode

// The OAuth edge (Bar 4's discovery half), testable without a rig: resource
// metadata, the 401 challenge without a badge, and 401 invalid_token when
// admission fails — plus eviction: a refused hint must retry, not poison.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOAuthEdge(t *testing.T) {
	n := New(Config{
		NATSURL:      "nats://127.0.0.1:1", // nothing listens: every admission fails fast
		Realm:        "proof",
		PublicURL:    "https://node.example.ts.net",
		AuthIssuer:   "https://tenant.auth0.example/",
		SentinelPath: "",
	})
	web := httptest.NewServer(n)
	t.Cleanup(web.Close)

	// RFC 9728 resource metadata.
	res, err := http.Get(web.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatalf("metadata get: %v", err)
	}
	defer res.Body.Close()
	var meta struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&meta); err != nil {
		t.Fatalf("metadata decode: %v", err)
	}
	if meta.Resource != "https://node.example.ts.net" ||
		len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "https://tenant.auth0.example/" {
		t.Fatalf("metadata wrong: %+v", meta)
	}

	// No badge → 401 with the discovery challenge.
	res, err = http.Post(web.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no-badge status: %d", res.StatusCode)
	}
	if h := res.Header.Get("WWW-Authenticate"); !strings.Contains(h, "resource_metadata=") {
		t.Fatalf("challenge missing resource_metadata: %q", h)
	}

	// A badge that fails admission → 401 invalid_token (not the bare 400).
	req, _ := http.NewRequest(http.MethodPost, web.URL, nil)
	req.Header.Set("Authorization", "Bearer sit_deadbeef")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("bad-badge post: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad-badge status: %d", res.StatusCode)
	}
	if h := res.Header.Get("WWW-Authenticate"); !strings.Contains(h, `error="invalid_token"`) {
		t.Fatalf("challenge missing invalid_token: %q", h)
	}

	// Eviction: the failed hint retries admission on the next request rather
	// than staying a corpse (build attempts observable via fresh errors).
	if n.Err("sit_deadbeef") == nil {
		t.Fatal("expected a recorded admission error")
	}
	req2, _ := http.NewRequest(http.MethodPost, web.URL, nil)
	req2.Header.Set("Authorization", "Bearer sit_deadbeef")
	res, err = http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("retry post: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("retry status: %d", res.StatusCode)
	}
}

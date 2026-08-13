package node

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/impire-io/soulstream/ceremony"
)

// TestHelmPlane proves the composition: `planes.shell` runs soulstream-shell
// through its public embed seam on the node's own lanes, the surface is
// closed until sign-in, and the disabled arm stays byte-identical to a
// shell-less node. The full human ceremony (passkey, session, act,
// custody scan) is soulstream-shell's own consumer-position gate — this test
// holds the node's side of the contract.
func TestHelmPlane(t *testing.T) {
	dir := t.TempDir()
	foldPort := reserveHelmTestPort(t)
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	st.DoorListen = "127.0.0.1:0"
	st.FoldListen = "127.0.0.1:" + foldPort
	st.FoldIssuer = "http://localhost:" + foldPort
	st.HelmListen = "127.0.0.1:0"
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Stop()
	if _, err := Found(n, st, dir); err != nil {
		t.Fatal(err)
	}

	if n.HelmURL() == "" {
		t.Fatal("shell plane enabled but HelmURL is empty")
	}
	resp, err := http.Get(n.HelmURL() + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(page), "Sign in") {
		t.Fatalf("shell front page: %d %s", resp.StatusCode, page)
	}
	if strings.Contains(string(page), "Sign out") {
		t.Fatal("unauthenticated visitor holds a session")
	}
	live, err := http.Get(n.HelmURL() + "/live")
	if err != nil {
		t.Fatal(err)
	}
	_ = live.Body.Close()
	if live.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /live = %d, want 401", live.StatusCode)
	}
}

// TestHelmDisabled: the disabled arm runs no shell and answers no URL.
func TestHelmDisabled(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatal(err)
	}
	st.DoorListen = "127.0.0.1:0"
	st.FoldEnabled = false
	st.HelmEnabled = false
	if err := st.Save(dir); err != nil {
		t.Fatal(err)
	}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Stop()
	if n.HelmURL() != "" {
		t.Fatalf("disabled shell answers %q", n.HelmURL())
	}
}

func reserveHelmTestPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	_ = ln.Close()
	return port
}

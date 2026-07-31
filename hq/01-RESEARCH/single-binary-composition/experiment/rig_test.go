package soulnoderig

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulidentity/client"
)

// userInfo mirrors the $SYS.REQ.USER.INFO response shape the remote-mcp-node
// prototype reads (node.go principalOf): the server-asserted principal is the
// expanded soulidentity.<account>.<name>.sign.record grant.
type userInfo struct {
	Data struct {
		User        string `json:"user"`
		Account     string `json:"account"`
		Permissions struct {
			Publish struct {
				Allow []string `json:"allow"`
			} `json:"publish"`
		} `json:"permissions"`
	} `json:"data"`
}

// principalOf asks the server who this connection is: no client claim anywhere.
func principalOf(t *testing.T, nc *nats.Conn) (persona, account string, allow []string) {
	t.Helper()
	msg, err := nc.Request("$SYS.REQ.USER.INFO", nil, 5*time.Second)
	if err != nil {
		t.Fatalf("USER.INFO: %v", err)
	}
	var info userInfo
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		t.Fatalf("USER.INFO decode: %v", err)
	}
	allow = info.Data.Permissions.Publish.Allow
	for _, subj := range allow {
		parts := strings.Split(subj, ".")
		if len(parts) == 5 && parts[0] == siclient.Segment && parts[3] == "sign" && parts[4] == "record" {
			return parts[2], parts[1], allow
		}
	}
	t.Fatalf("no expanded sign.record grant in resolved permissions: %v", allow)
	return "", "", nil
}

// observe runs the three pre-registered Bar 1 observations against a
// provisioned rig: (a) token-lane admission with server-resolved scoping,
// (b) garbage-token refusal, (c) revoked-token refusal.
func observe(t *testing.T, r *Rig) {
	t.Helper()
	lap := stopwatch(t)

	// Informational only (the sibling rig's amended Bar 1 finding): a bare
	// token without the sentinel must not admit in operator mode.
	if nc, err := r.Connect(nats.Token(r.Token)); err == nil {
		nc.Close()
		t.Fatalf("bare token admitted without sentinel — contradicts the wire rig's finding")
	}
	lap("bare-token refusal")

	// (a) sentinel + token admits; the server names the principal and scope.
	nc, err := r.Connect(nats.UserCredentials(r.SentinelPath), nats.Token(r.Token))
	if err != nil {
		t.Fatalf("token lane refused: %v", err)
	}
	defer nc.Close()
	lap("token-lane admission")
	persona, account, allow := principalOf(t, nc)
	lap("USER.INFO principal")
	if persona != r.Persona || account != r.RealmPub {
		t.Fatalf("principal = %s@%s, want %s@%s", persona, account, r.Persona, r.RealmPub)
	}
	ownPrefix := fmt.Sprintf("%s.%s.%s.", siclient.Segment, r.RealmPub, r.Persona)
	for _, subj := range allow {
		if strings.HasPrefix(subj, siclient.Segment+".") &&
			subj != siclient.Segment+".status" && subj != siclient.Segment+".xkey" &&
			!strings.HasPrefix(subj, ownPrefix) {
			t.Fatalf("resolved grant escapes the persona's own prefix: %s", subj)
		}
	}

	// (b) garbage token: no connection forms, the audit records the refusal.
	if ncBad, err := r.Connect(nats.UserCredentials(r.SentinelPath),
		nats.Token("sit_"+strings.Repeat("00", 32))); err == nil {
		ncBad.Close()
		t.Fatal("garbage token admitted")
	}
	if !strings.Contains(r.Audit.String(), "callout REFUSED") {
		t.Fatal("no 'callout REFUSED' in the audit after the garbage token")
	}
	lap("garbage-token refusal")

	// (c) revoked token: refusal after the digest is gone from the store.
	if err := r.Admin.RevokeToken(r.Digest); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if ncRevoked, err := r.Connect(nats.UserCredentials(r.SentinelPath), nats.Token(r.Token)); err == nil {
		ncRevoked.Close()
		t.Fatal("revoked token admitted")
	}
	lap("revoked-token refusal")
}

// stopwatch logs the duration of each phase so mode-divergent latencies are
// visible in -v output.
func stopwatch(t *testing.T) func(string) {
	last := time.Now()
	return func(phase string) {
		t.Logf("%-24s %v", phase, time.Since(last).Round(time.Millisecond))
		last = time.Now()
	}
}

// TestBar1InProcess is the pre-registered Bar 1 protocol in the SoulNode
// shape: DontListen (no TCP listener exists at all), every connection
// in-process, three consecutive fresh-rig runs, 3/3 observations each.
func TestBar1InProcess(t *testing.T) {
	for run := 1; run <= 3; run++ {
		t.Run(fmt.Sprintf("run-%d", run), func(t *testing.T) {
			r, err := Provision(t.TempDir(), true)
			if err != nil {
				t.Fatalf("provision: %v", err)
			}
			defer r.Shutdown()
			observe(t, r)
		})
	}
}

// TestBar1TCPControl is the control arm: the identical ceremony reached over
// a real TCP listener, so any in-process divergence would show against it.
func TestBar1TCPControl(t *testing.T) {
	r, err := Provision(t.TempDir(), false)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer r.Shutdown()
	observe(t, r)
}

// TestBar3CeremonyFromEmptyDir asserts the Bar 3 threshold mechanically
// where it can: the ceremony starts from an empty directory, uses no
// external binary (by construction — everything above is library code), and
// ends with the admission artifacts on disk and a working admission. The
// 1:1 inventory↔code agreement is reviewed, not asserted: Inventory entries
// cite their provision steps.
func TestBar3CeremonyFromEmptyDir(t *testing.T) {
	dir := t.TempDir()
	r, err := Provision(dir, true)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer r.Shutdown()

	if len(Inventory) == 0 {
		t.Fatal("empty ceremony inventory")
	}
	if r.SentinelPath == "" || r.Token == "" || r.Digest == "" {
		t.Fatal("ceremony ended without its admission artifacts")
	}
	if !strings.HasPrefix(r.Token, "sit_") {
		t.Fatalf("API token has unexpected shape: %q", r.Token[:8])
	}
	nc, err := r.Connect(nats.UserCredentials(r.SentinelPath), nats.Token(r.Token))
	if err != nil {
		t.Fatalf("fresh-dir ceremony did not reach admission: %v", err)
	}
	nc.Close()
}

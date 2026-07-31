package soulnoderig

import (
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

// TestRefusalLatencyProbe pins down whose timeout the 10s in-process refusal
// block belongs to: if it scales with the client's nats.Timeout the client is
// waiting out its own connect deadline (the server never surfaces the -ERR
// over the in-process pipe); if it stays fixed it is a server-side wait.
func TestRefusalLatencyProbe(t *testing.T) {
	r, err := Provision(t.TempDir(), true)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer r.Shutdown()

	garbage := "sit_" + strings.Repeat("00", 32)

	for _, tc := range []struct {
		name    string
		timeout time.Duration
	}{
		{"default-timeout", 0},
		{"one-second-timeout", time.Second},
	} {
		start := time.Now()
		opts := []nats.Option{nats.UserCredentials(r.SentinelPath), nats.Token(garbage)}
		if tc.timeout != 0 {
			opts = append(opts, nats.Timeout(tc.timeout))
		}
		nc, err := r.Connect(opts...)
		took := time.Since(start).Round(time.Millisecond)
		if err == nil {
			nc.Close()
			t.Fatalf("%s: garbage token admitted", tc.name)
		}
		t.Logf("%-20s refused in %v (err: %v)", tc.name, took, err)
	}
}

// TestRefusalTransportIsolation separates the two candidate variables: a
// TCP-listening server refusing (i) a TCP client and (ii) an in-process-pipe
// client. If (ii) still blocks 10s, the divergence is the pipe transport, not
// DontListen mode.
func TestRefusalTransportIsolation(t *testing.T) {
	r, err := Provision(t.TempDir(), false) // listener ON
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer r.Shutdown()

	garbage := "sit_" + strings.Repeat("00", 32)

	start := time.Now()
	_, errTCP := nats.Connect(r.Srv.ClientURL(),
		nats.UserCredentials(r.SentinelPath), nats.Token(garbage),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	t.Logf("tcp client:        refused in %v (err: %v)",
		time.Since(start).Round(time.Millisecond), errTCP)

	start = time.Now()
	_, errPipe := nats.Connect(nats.DefaultURL,
		nats.InProcessServer(r.Srv),
		nats.UserCredentials(r.SentinelPath), nats.Token(garbage),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	t.Logf("in-process client: refused in %v (err: %v)",
		time.Since(start).Round(time.Millisecond), errPipe)

	if errTCP == nil || errPipe == nil {
		t.Fatal("garbage token admitted on one of the transports")
	}
}

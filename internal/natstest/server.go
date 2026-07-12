package natstest

import (
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// StartJetStream starts an in-process NATS server with JetStream enabled, backed by
// a per-test temporary store directory, and returns its client URL together with a
// cleanup function. The store directory is removed automatically when the test ends.
//
// Typical use:
//
//	url, cleanup := natstest.StartJetStream(t)
//	defer cleanup()
//	nc, _ := nats.Connect(url)
func StartJetStream(t *testing.T) (url string, cleanup func()) {
	t.Helper()

	opts := &server.Options{
		JetStream: true,
		StoreDir:  t.TempDir(),
		Host:      "127.0.0.1",
		Port:      -1, // pick a random free port
		NoLog:     true,
		NoSigs:    true, // do not install OS signal handlers in tests
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("natstest: new server: %v", err)
	}

	go ns.Start()

	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		t.Fatal("natstest: server not ready for connections")
	}

	return ns.ClientURL(), ns.Shutdown
}

package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestCurateCommand(t *testing.T) {
	connect := testConnector(t)
	base := []string{"--realm", "acme", "--persona", "daan"}
	run(connect, append(base, "provision")...)
	_, out, _ := run(connect, append(base, "start", "Design Review")...)
	path := strings.TrimSpace(out)
	run(connect, append(base, "post", path, "the xylophone gate question")...)

	// The curator, long-running under its own persona, cancellable.
	curCtx, stopCurator := context.WithCancel(context.Background())
	defer stopCurator()
	var curOut, curErr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- Run(curCtx, []string{"--realm", "acme", "--persona", "curator", "curate", "--scan-every", "100ms"}, &curOut, &curErr, connect)
	}()

	// Wait for the projection, then ask for body-only content.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(curOut.String(), "projection ready") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(curOut.String(), `curating as "curator"`) {
		t.Fatalf("curate banner missing: %q", curOut.String())
	}
	if !strings.Contains(curOut.String(), "projection ready: 1 topics") {
		t.Fatalf("projection-ready line missing: %q", curOut.String())
	}

	code, out, errs := run(connect, append(base, "discover", "xylophone", "--timeout", "700ms")...)
	if code != 0 {
		t.Fatalf("discover exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "Design Review") || !strings.Contains(out, "answered by: curator") {
		t.Errorf("content ask not answered by the curator:\n%s", out)
	}

	stopCurator()
	select {
	case codeCur := <-done:
		if codeCur != 0 {
			t.Errorf("curate exit %d: %s", codeCur, curErr.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("curate did not stop on cancel")
	}

	// A persona is required.
	if code, _, _ := run(connect, "--realm", "acme", "curate"); code == 0 {
		t.Error("curate without persona should fail")
	}
}

package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestDiscoverAndRespond(t *testing.T) {
	connect, url := testConnectorWithURL(t)
	_ = url
	base := []string{"--realm", "acme", "--persona", "daan"}
	run(connect, append(base, "provision")...)
	run(connect, append(base, "start", "Q2 VAT filing", "--subject", "filing", "--tag", "finance")...)

	// Silence first: no responder running.
	code, out, errs := run(connect, append(base, "discover", "vat", "--timeout", "300ms")...)
	if code != 0 {
		t.Fatalf("silent discover exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "no answers before the deadline") {
		t.Errorf("silent discover output: %q", out)
	}

	// Start a responder under another persona, in-process with a cancellable ctx.
	respCtx, stopResponder := context.WithCancel(context.Background())
	defer stopResponder()
	var respOut, respErr bytes.Buffer
	respDone := make(chan int, 1)
	go func() {
		respDone <- Run(respCtx, []string{"--realm", "acme", "--persona", "architect", "respond"}, &respOut, &respErr, connect)
	}()
	time.Sleep(150 * time.Millisecond) // let the subscription establish

	code, out, errs = run(connect, append(base, "discover", "vat", "--timeout", "700ms")...)
	if code != 0 {
		t.Fatalf("discover exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "Q2 VAT filing") || !strings.Contains(out, "answered by: architect") {
		t.Errorf("discover output:\n%s", out)
	}

	stopResponder()
	select {
	case codeResp := <-respDone:
		if codeResp != 0 {
			t.Errorf("respond exit %d: %s", codeResp, respErr.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("respond did not stop on context cancel")
	}
	if !strings.Contains(respOut.String(), "responding to discovery as \"architect\"") {
		t.Errorf("respond banner missing: %q", respOut.String())
	}
	if !strings.Contains(respOut.String(), `served "vat": 1 matches`) {
		t.Errorf("respond served line missing: %q", respOut.String())
	}

	// discover requires a persona (it publishes a request as someone).
	if code, _, _ := run(connect, "--realm", "acme", "discover", "vat"); code == 0 {
		t.Error("discover without persona should fail")
	}
}

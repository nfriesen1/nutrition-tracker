package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// waitForListener polls until the address accepts a connection, or the deadline passes.
func waitForListener(t *testing.T, addr string, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not start listening on %s within %s", addr, within)
}

// The graceful shutdown contract: cancelling the context stops the listener and returns
// cleanly, rather than dying mid-request and giving the client a connection reset.
//
// This drives run() through a context instead of a real signal because Windows only
// delivers os.Interrupt via console events, and a detached test process has no console.
// Production is Linux, where SIGTERM reaches the same code path — see docs/06-deployment.md.
func TestRunShutsDownOnContextCancel(t *testing.T) {
	const addr = "127.0.0.1:18080"
	t.Setenv("PORT", "18080")
	t.Setenv("ENV", "development")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	waitForListener(t, addr, 5*time.Second)

	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() returned %v, want nil on graceful shutdown", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("run() did not return after the context was cancelled")
	}

	// The listener must actually be released, or the next deploy cannot bind the port.
	if conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		_ = conn.Close()
		t.Error("server is still accepting connections after shutdown")
	}
}

// A port already in use must surface as a startup error rather than a process that is
// alive but serving nothing.
//
// The occupying listener binds 0.0.0.0, matching what run() binds. Binding 127.0.0.1
// instead does not conflict on Windows — the two coexist — so the test would pass on
// Linux and hang here for no reason related to the code.
func TestRunReportsListenFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	defer func() { _ = listener.Close() }()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("splitting address: %v", err)
	}

	t.Setenv("PORT", port)
	t.Setenv("ENV", "development")

	// Generous enough to be reliable, short enough that a regression fails fast rather
	// than stalling the suite.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err = run(ctx)

	if err == nil {
		t.Fatal("run() error = nil, want a bind failure")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Errorf("run() returned ErrServerClosed, want the underlying bind error")
	}
	// It must fail immediately, not sit until the context expires — a deploy that cannot
	// bind should exit at once so the orchestrator holds the previous version.
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("took %s to report a bind failure; should be immediate", elapsed)
	}
}

func TestRunRejectsBadConfig(t *testing.T) {
	t.Setenv("ENV", "prod")

	if err := run(context.Background()); err == nil {
		t.Fatal("run() error = nil; a typo'd ENV must not boot")
	}
}

package papermcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestRealStdioTimeoutAndRestart(t *testing.T) {
	adapter := os.Getenv("PAPER_ADAPTER_PATH")
	if adapter == "" {
		t.Skip("set PAPER_ADAPTER_PATH to the project's Python adapter")
	}
	if !filepath.IsAbs(adapter) {
		t.Fatal("PAPER_ADAPTER_PATH must be absolute")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child-pid")
	wrapper := filepath.Join(dir, "start")
	t.Setenv("PAPER_TEST_PID", pidFile)
	t.Setenv("PAPER_TEST_ADAPTER", adapter)
	script := "#!/bin/sh\nprintf '%s\\n' \"$$\" > \"$PAPER_TEST_PID\"\nexec \"$PAPER_TEST_ADAPTER\" \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	p := New(wrapper, "--fixture", "--miss-b", "--a", "a", "--b", "b", "--outcome-a", "No", "--outcome-b", "No")
	done := make(chan struct{})
	go func() { defer close(done); p.Run(ctx) }()
	defer func() { cancel(); <-done }()

	first := awaitSnapshot(t, ctx, p, 0)
	request := FillRequest{
		RequestID: "b", Session: first.Session, Generation: first.Generation, SnapshotID: first.ID,
		Token: first.Legs[1].Token, Side: "buy", Shares: decimal.NewFromInt(5), Limit: decimal.New(48, -2), Intent: "taker",
	}
	timed, stop := context.WithTimeout(ctx, 80*time.Millisecond)
	defer stop()
	failed := make(chan error, 1)
	go func() { _, err := p.Fill(timed, request); failed <- err }()
	if _, err := p.Snapshot(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-failed; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("slow fill: %v", err)
	}
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil || pid <= 1 {
		t.Fatalf("invalid child PID: %v", err)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}

	second := awaitSnapshot(t, ctx, p, first.Session)
	if first.Generation == second.Generation {
		t.Fatal("restarted adapter reused a generation")
	}
	if _, err := p.Fill(ctx, request); !errors.Is(err, ErrOldSession) {
		t.Fatalf("old fill: %v", err)
	}
}

func awaitSnapshot(t *testing.T, ctx context.Context, p *Provider, oldSession uint64) Snapshot {
	t.Helper()
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("snapshot unavailable: %+v", p.Status())
		case <-ticker.C:
			call, stop := context.WithTimeout(ctx, 200*time.Millisecond)
			s, err := p.Snapshot(call)
			stop()
			if err == nil && s.Session > oldSession && s.WS && len(s.Legs) == 2 &&
				s.Legs[0].ExchangeMS != nil && s.Legs[1].ExchangeMS != nil {
				return s
			}
		}
	}
}

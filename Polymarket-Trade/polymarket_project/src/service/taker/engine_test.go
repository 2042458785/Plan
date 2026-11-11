package taker

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"Polymarket/src/pkg/papermcp"

	"github.com/shopspring/decimal"
)

func d(value string) decimal.Decimal { return decimal.RequireFromString(value) }

func level(price, size string) papermcp.Level { return papermcp.Level{Price: d(price), Size: d(size)} }

func testSnapshot(now time.Time) papermcp.Snapshot {
	stamp, age, rate := now.UnixMilli(), int64(0), d("0.05")
	makeLeg := func(token string) papermcp.Leg {
		return papermcp.Leg{Token: token, ExchangeMS: &stamp, ReceiveAgeMS: &age, Tick: d("0.01"),
			Minimum: d("5"), Rate: &rate, Accepting: true,
			Asks: []papermcp.Level{level("0.48", "100")}, Bids: []papermcp.Level{level("0.45", "100")}}
	}
	return papermcp.Snapshot{Session: 1, Generation: "generation", ID: "book", WS: true,
		RequestedAt: now, Legs: []papermcp.Leg{makeLeg("a"), makeLeg("b")}}
}

func requireAmount(t *testing.T, got decimal.Decimal, expected string) {
	t.Helper()
	if !got.Equal(d(expected)) {
		t.Fatalf("got %s want %s", got, expected)
	}
}

func TestSyncBoundary(t *testing.T) {
	now := time.Now()
	snapshot := testSnapshot(now)
	for _, diff := range []int64{199, 200} {
		stamp := now.UnixMilli() - diff
		snapshot.Legs[1].ExchangeMS = &stamp
		skew, err := Sync(snapshot)
		if skew.Milliseconds() != diff || (err == nil) != (diff < 200) {
			t.Fatalf("skew %d: got %v %v", diff, skew, err)
		}
	}
	snapshot.Legs[1].ExchangeMS = nil
	if _, err := Sync(snapshot); !errors.Is(err, ErrLocalTS) {
		t.Fatal(err)
	}
	snapshot = testSnapshot(now.Add(-3 * time.Second))
	if _, err := CalculateVWAP(snapshot, d("5"), one, d("5"), now, time.Second); !errors.Is(err, ErrStale) {
		t.Fatalf("same-age stale books accepted: %v", err)
	}
}

func TestVWAPDepthAndDynamicFees(t *testing.T) {
	now := time.Now()
	snapshot := testSnapshot(now)
	snapshot.Legs[0].Asks = []papermcp.Level{level("0.49", "10"), level("0.46", "3")}
	rate := d("0.04")
	snapshot.Legs[1].Rate = &rate
	quote, err := CalculateVWAP(snapshot, d("5"), one, d("5"), now, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	requireAmount(t, quote.Legs[0].VWAP(), "0.472")
	requireAmount(t, quote.Legs[0].Fees, "0.06225")
	requireAmount(t, quote.Legs[1].Fees, "0.04992")
	requireAmount(t, quote.NetFloor, "0.12783")
}

func TestQuoteRejectsInvalidInputs(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		change  func(*papermcp.Snapshot)
		balance decimal.Decimal
	}{
		{"off tick", func(s *papermcp.Snapshot) { s.Legs[0].Tick = d("0.05") }, d("5")},
		{"minimum", func(s *papermcp.Snapshot) { s.Legs[0].Minimum = d("6") }, d("5")},
		{"unknown rate", func(s *papermcp.Snapshot) { s.Legs[1].Rate = nil }, d("5")},
		{"depth", func(s *papermcp.Snapshot) { s.Legs[1].Asks[0].Size = d("4") }, d("5")},
		{"balance", func(_ *papermcp.Snapshot) {}, d("4.92")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testSnapshot(now)
			tt.change(&s)
			if _, err := CalculateVWAP(s, d("5"), one, tt.balance, now, time.Second); err == nil {
				t.Fatal("invalid opportunity accepted")
			}
		})
	}
	if _, err := bookSweep(testSnapshot(now).Legs[0], "buy", d("5"), d("0.48"), "maker"); err == nil || err.Error() != MakerBlocked {
		t.Fatalf("maker intent result: %v", err)
	}
}

func TestPartialAndLateFillAccounting(t *testing.T) {
	now := time.Now()
	snapshot := testSnapshot(now)
	a, err := bookSweep(snapshot.Legs[0], "buy", d("10"), d("0.48"), "taker")
	if err != nil {
		t.Fatal(err)
	}
	b, err := bookSweep(snapshot.Legs[1], "buy", d("5"), d("0.48"), "taker")
	if err != nil {
		t.Fatal(err)
	}
	s := runState{View: View{Phase: pendingB, Ledger: Ledger{Cash: d("50")}}, deadline: now}
	s.Ledger.confirmA([2]string{"a", "b"}, a)
	s.onExecution(context.Background(), nil, Config{}, event{kind: "b", finished: now.Add(-time.Millisecond), sweep: b}, nil)
	s.onExecution(context.Background(), nil, Config{}, event{kind: "b", finished: now.Add(-time.Millisecond), sweep: b}, nil)
	requireAmount(t, s.Ledger.Exposure.Shares, "5")
	requireAmount(t, s.Ledger.Paired.Shares, "5")
	s.onExecution(context.Background(), nil, Config{}, event{kind: "b", finished: now, sweep: b}, nil)
	requireAmount(t, s.Ledger.Exposure.Shares, "5")
	s.Phase = hedging
	hedge, err := bookSweep(snapshot.Legs[0], "sell", d("5"), d("0.45"), "taker")
	if err != nil {
		t.Fatal(err)
	}
	receipt := event{kind: "hedge", sweep: hedge}
	s.onExecution(context.Background(), nil, Config{}, receipt, nil)
	s.onExecution(context.Background(), nil, Config{}, receipt, nil)
	requireAmount(t, s.Ledger.Realized, "-0.27428")
	requireAmount(t, s.Ledger.Exposure.Shares, "0")
	requireAmount(t, s.Ledger.Paired.Cost, "4.92480")
	snapshot.Legs[0].Payout = &one
	zero := decimal.Zero
	snapshot.Legs[1].Payout = &zero
	pnl, ok := s.Ledger.settle(snapshot)
	if !ok {
		t.Fatal("valid result was not settled")
	}
	requireAmount(t, pnl, "0.07520")
	requireAmount(t, s.Ledger.Cash, "49.80092")
}

func TestFailedHedgePausesAndPreservesExposure(t *testing.T) {
	s := runState{View: View{Phase: hedging, Ledger: Ledger{Exposure: Position{Shares: d("5"), Cost: d("2.46240")}}}}
	s.onExecution(context.Background(), nil, Config{}, event{kind: "hedge", err: context.DeadlineExceeded}, nil)
	if !s.Paused || s.Phase != failed || !strings.Contains(s.Logs[0], "[HEDGE_FAILED]") {
		t.Fatalf("unexpected state: %+v", s.View)
	}
	requireAmount(t, s.Ledger.Exposure.Shares, "5")
}

func TestRejectMismatchedFill(t *testing.T) {
	s := testSnapshot(time.Now())
	request := fillRequest(1, "a", s, s.Legs[0], "buy", d("5"), d("0.48"))
	result := papermcp.FillResult{RequestID: request.RequestID, Generation: request.Generation,
		SnapshotID: request.SnapshotID, Token: request.Token, Side: request.Side, Status: "filled",
		Fills: []papermcp.Fill{{Price: d("0.48"), Shares: d("5"), Fee: d("0.00001")}}}
	if _, err := checkFill(request, s.Legs[0], result); err == nil {
		t.Fatal("wrong fee accepted")
	}
}

type exhaustedHedge struct{ bought atomic.Bool }

func (p *exhaustedHedge) Status() papermcp.Status {
	return papermcp.Status{Ready: true, Session: 1}
}

func (p *exhaustedHedge) Reconnect() {}

func (p *exhaustedHedge) Snapshot(ctx context.Context) (papermcp.Snapshot, error) {
	snapshot := testSnapshot(time.Now())
	if p.bought.Load() {
		snapshot.Legs[0].Bids = nil
	}
	return snapshot, ctx.Err()
}

func (p *exhaustedHedge) Fill(ctx context.Context, request papermcp.FillRequest) (papermcp.FillResult, error) {
	if request.Token == "b" {
		<-ctx.Done()
		return papermcp.FillResult{}, ctx.Err()
	}
	snapshot := testSnapshot(time.Now())
	sweep, err := bookSweep(snapshot.Legs[0], request.Side, request.Shares, request.Limit, request.Intent)
	if err != nil {
		return papermcp.FillResult{}, err
	}
	p.bought.Store(true)
	return papermcp.FillResult{RequestID: request.RequestID, Generation: request.Generation,
		SnapshotID: request.SnapshotID, Token: request.Token, Side: request.Side, Status: "filled", Fills: sweep.Fills}, nil
}

func TestRunnerHedgeDepthDisappearsAfterA(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	views := make(chan View, 1)
	cfg := Config{Balance: d("5"), Shares: d("5"), PayoutFloor: one,
		MaxAge: time.Second, HedgeTimeout: time.Second, Reviewed: true}
	go Run(ctx, &exhaustedHedge{}, cfg, nil, views)
	for {
		select {
		case <-ctx.Done():
			t.Fatal("hedge failure did not stop the runner")
		case view := <-views:
			if view.Phase != failed {
				continue
			}
			if !view.Paused || !strings.Contains(strings.Join(view.Logs, "\n"), "[HEDGE_FAILED]") {
				t.Fatalf("unexpected failure state: %+v", view)
			}
			requireAmount(t, view.Ledger.Cash, "2.53760")
			requireAmount(t, view.Ledger.Exposure.Shares, "5")
			requireAmount(t, view.Ledger.Realized, "0")
			return
		}
	}
}

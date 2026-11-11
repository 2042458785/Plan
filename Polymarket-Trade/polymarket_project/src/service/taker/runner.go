package taker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Polymarket/src/pkg/papermcp"

	"github.com/shopspring/decimal"
)

type MarketIO interface {
	Status() papermcp.Status
	Snapshot(context.Context) (papermcp.Snapshot, error)
	Fill(context.Context, papermcp.FillRequest) (papermcp.FillResult, error)
	Reconnect()
}

type Config struct {
	Balance      decimal.Decimal
	Shares       decimal.Decimal
	PayoutFloor  decimal.Decimal
	MaxAge       time.Duration
	HedgeTimeout time.Duration
	Reviewed     bool
	ObserveOnly  bool
}

type View struct {
	Quote         *Quote
	SkewRejected  int
	StaleRejected int
	MCP           papermcp.Status
	Snapshot      papermcp.Snapshot
	Ledger        Ledger
	Phase         string
	Paused        bool
	Sync          time.Duration
	Reason        string
	LocalTS       bool
	Checked       int
	Eligible      int
	Profitable    int
	TotalRTT      time.Duration
	MaxRTT        time.Duration
	Logs          []string
}

type Command int

const (
	TogglePause Command = iota
	Reconnect
)

const (
	idle     = "IDLE"
	pendingA = "A PENDING"
	pendingB = "B PENDING"
	hedging  = "HEDGING"
	failed   = "HEDGE FAILED"
)

type event struct {
	kind     string
	id       uint64
	session  uint64
	snapshot papermcp.Snapshot
	sweep    Sweep
	finished time.Time
	err      error
}

type runState struct {
	rules         [2]string
	rulesChanged  bool
	lastFeedError string
	View
	orderID         uint64
	deadline        time.Time
	timer           *time.Timer
	cancelB         context.CancelFunc
	lastSnapshot    string
	lastSignal      time.Time
	lastPollError   string
	lastMCPError    string
	pending         pendingTrade
	polling         bool
	recoverExposure bool
	bProcessed      bool
}

func (s *runState) log(format string, args ...any) {
	line := time.Now().Format("15:04:05.000") + " " + fmt.Sprintf(format, args...)
	s.Logs = append(s.Logs, line)
	if len(s.Logs) > 80 {
		s.Logs = append([]string(nil), s.Logs[len(s.Logs)-80:]...)
	}
}

func Run(ctx context.Context, io MarketIO, cfg Config, controls <-chan Command, output chan View) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()
	events := make(chan event, 8)
	state := runState{View: View{Phase: idle, Ledger: Ledger{Cash: cfg.Balance}}}
	defer state.clearDeadline()
	send := func(result event) {
		select {
		case events <- result:
		case <-ctx.Done():
		}
	}
	publish := func() {
		view := state.View
		view.Logs = append([]string(nil), view.Logs...)
		select {
		case output <- view:
		default:
			select {
			case <-output:
			default:
			}
			select {
			case output <- view:
			default:
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case control, open := <-controls:
			if !open {
				controls = nil
				continue
			}
			switch control {
			case TogglePause:
				if state.rulesChanged {
					state.log("Market rules changed; review again before restarting")
				} else if state.Phase == failed {
					state.log("Risk paused; reconnect to retry outstanding hedge")
				} else {
					state.Paused = !state.Paused
					state.log("New signals paused=%t", state.Paused)
				}
			case Reconnect:
				io.Reconnect()
				state.recoverExposure = state.Phase == failed
				state.Snapshot = papermcp.Snapshot{}
				state.Quote = nil
				state.log("MCP reconnect requested")
			}
		case <-ticker.C:
			status := io.Status()
			if status.Error != "" && status.Error != state.lastMCPError {
				state.log("MCP: DEAD %s", status.Error)
			}
			state.lastMCPError = status.Error
			if status.Session != state.MCP.Session || !status.Ready {
				state.Snapshot = papermcp.Snapshot{}
				state.Quote = nil
				state.Reason = "waiting for new session"
				state.LocalTS = false
			}
			state.MCP = status
			if status.Ready && !state.polling {
				state.polling = true
				go func() {
					call, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
					defer cancel()
					s, err := io.Snapshot(call)
					send(event{kind: "snapshot", snapshot: s, finished: time.Now(), err: err})
				}()
			}
		case result := <-events:
			if result.kind == "snapshot" {
				state.polling = false
				if result.err != nil {
					state.Snapshot = papermcp.Snapshot{}
					state.Quote = nil
					state.Reason = "snapshot unavailable"
					message := result.err.Error()
					if message != state.lastPollError && !errors.Is(result.err, papermcp.ErrUnavailable) {
						state.log("Snapshot unavailable: %s", message)
					}
					state.lastPollError = message
				} else if status := io.Status(); status.Ready && status.Session == result.snapshot.Session {
					state.MCP = status
					state.lastPollError = ""
					state.onSnapshot(ctx, io, cfg, result, send)
				}
			} else if result.id == state.orderID {
				if result.session != 0 {
					status := io.Status()
					if !status.Ready || status.Session != result.session {
						result.err = papermcp.ErrOldSession
					}
				}
				state.onExecution(ctx, io, cfg, result, send)
			}
		}
		publish()
	}
}

func (s *runState) onSnapshot(ctx context.Context, io MarketIO, cfg Config, result event, send func(event)) {
	snapshot := result.snapshot
	s.Snapshot = snapshot
	s.Quote = nil
	if snapshot.FeedError != "" && snapshot.FeedError != s.lastFeedError {
		s.log("WS: DOWN %s", snapshot.FeedError)
	}
	s.lastFeedError = snapshot.FeedError
	if cfg.Reviewed && len(snapshot.Legs) == 2 {
		rules := [2]string{snapshot.Legs[0].RulesFingerprint, snapshot.Legs[1].RulesFingerprint}
		if s.rules == [2]string{} {
			s.rules = rules
		} else if rules != s.rules && !s.rulesChanged {
			s.rulesChanged, s.Paused = true, true
			s.log("Market rules changed; new signals paused")
		}
	}
	if pnl, ok := s.Ledger.settle(snapshot); ok {
		s.log("[SETTLED] pair realized %s", pnl.StringFixed(5))
	}
	s.Checked++
	rtt := result.finished.Sub(snapshot.RequestedAt)
	if rtt >= 0 {
		s.TotalRTT += rtt
		s.MaxRTT = max(s.MaxRTT, rtt)
	}
	now := time.Now()
	skew, err := Sync(snapshot)
	s.Sync = skew
	previousLocalTS := s.LocalTS
	s.LocalTS = errors.Is(err, ErrLocalTS)
	if s.LocalTS && !previousLocalTS {
		s.log("[LOCAL_TS] exchange timestamp missing; strategy skipped")
	}
	if s.recoverExposure && s.Phase == failed && snapshot.WS && len(snapshot.Legs) == 2 {
		if ValidateLeg(snapshot, snapshot.Legs[0], now, cfg.MaxAge) == nil {
			s.recoverExposure = false
			s.startHedge(ctx, io, cfg, send)
		}
	}
	if err == nil {
		for _, leg := range snapshot.Legs {
			if err = ValidateLeg(snapshot, leg, now, cfg.MaxAge); err != nil {
				break
			}
		}
	}
	if err != nil {
		if errors.Is(err, ErrSkew) {
			s.SkewRejected++
		}
		if errors.Is(err, ErrStale) {
			s.StaleRejected++
		}
		s.Reason = err.Error()
		return
	}
	s.Eligible++
	if s.rulesChanged || s.Paused || s.Phase == failed {
		s.Reason = "new signals paused"
		return
	}
	if !cfg.Reviewed {
		s.Reason = "observe only: verify settlement rules before simulating"
		return
	}
	quote, err := CalculateVWAP(snapshot, cfg.Shares, cfg.PayoutFloor, s.Ledger.Cash, now, cfg.MaxAge)
	if err != nil {
		s.Reason = err.Error()
		return
	}
	s.Quote = &quote
	if !quote.NetFloor.IsPositive() {
		s.Reason = "no positive net payout floor"
		return
	}
	s.Profitable++
	s.Reason = "net floor $" + quote.NetFloor.StringFixed(5)
	if s.lastSignal.IsZero() || now.Sub(s.lastSignal) >= 10*time.Second {
		s.lastSignal = now
		s.log("[SIGNAL] net floor=%s | A VWAP=%s B VWAP=%s", quote.NetFloor.StringFixed(5),
			quote.Legs[0].VWAP().String(), quote.Legs[1].VWAP().String())
	}
	key := snapshot.Generation + "/" + snapshot.ID
	if cfg.ObserveOnly || s.Paused || s.Phase != idle || !s.Ledger.Paired.Shares.IsZero() || key == s.lastSnapshot {
		return
	}
	s.lastSnapshot = key
	s.orderID++
	s.Phase = pendingA
	request := fillRequest(s.orderID, "a", snapshot, snapshot.Legs[0], "buy", cfg.Shares, quote.Legs[0].Limit())
	tokens := [2]string{snapshot.Legs[0].Token, snapshot.Legs[1].Token}
	// Preserve the target and B price limit across async calls.
	s.pending = pendingTrade{tokens: tokens, bLimit: quote.Legs[1].Limit()}
	id := s.orderID
	go func() {
		call, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		sweep, err := execute(call, io, request, snapshot.Legs[0])
		send(event{kind: "a", id: id, session: request.Session, sweep: sweep, finished: time.Now(), err: err})
	}()
}

type pendingTrade struct {
	tokens [2]string
	bLimit decimal.Decimal
}

func fillRequest(id uint64, suffix string, snapshot papermcp.Snapshot, leg papermcp.Leg, side string, shares, limit decimal.Decimal) papermcp.FillRequest {
	return papermcp.FillRequest{RequestID: fmt.Sprintf("%d-%s", id, suffix), Session: snapshot.Session,
		Generation: snapshot.Generation, SnapshotID: snapshot.ID, Token: leg.Token,
		Side: side, Shares: shares, Limit: limit, Intent: "taker"}
}

func execute(ctx context.Context, io MarketIO, request papermcp.FillRequest, leg papermcp.Leg) (Sweep, error) {
	result, err := io.Fill(ctx, request)
	if err != nil {
		return Sweep{}, err
	}
	return checkFill(request, leg, result)
}

func (s *runState) onExecution(ctx context.Context, io MarketIO, cfg Config, result event, send func(event)) {
	switch result.kind {
	case "a":
		if s.Phase != pendingA {
			return
		}
		if result.err != nil || result.sweep.Shares.IsZero() {
			s.Phase = idle
			if result.err != nil {
				s.log("[A REJECTED] %s", result.err)
			}
			return
		}
		s.Ledger.confirmA(s.pending.tokens, result.sweep)
		s.log("[A FILL] shares=%s cost=%s", result.sweep.Shares, result.sweep.Cost())
		s.Phase = pendingB
		s.bProcessed = false
		s.deadline = time.Now().Add(HedgeAfter)
		id := s.orderID
		s.timer = time.AfterFunc(time.Until(s.deadline), func() {
			send(event{kind: "deadline", id: id})
		})
		s.startB(ctx, io, cfg, send)
	case "b":
		if s.Phase != pendingB || s.bProcessed || !result.finished.Before(s.deadline) {
			return
		}
		s.bProcessed = true
		if result.err != nil {
			s.log("[B UNCONFIRMED] %s", result.err)
			return
		}
		if result.sweep.Cost().GreaterThan(s.Ledger.Cash) {
			s.log("[B REJECTED] virtual balance insufficient")
			return
		}
		s.Ledger.confirmB(result.sweep)
		s.log("[B FILL] shares=%s unpaired=%s", result.sweep.Shares, s.Ledger.Exposure.Shares)
		if s.Ledger.Exposure.Shares.IsZero() {
			s.clearDeadline()
			s.Phase = idle
		}
	case "deadline":
		if s.Phase == pendingB {
			s.startHedge(ctx, io, cfg, send)
		}
	case "hedge":
		if s.Phase != hedging {
			return
		}
		if result.err != nil || !result.sweep.Shares.Equal(s.Ledger.Exposure.Shares) {
			s.Phase, s.Paused = failed, true
			s.log("[HEDGE_FAILED] unpaired=%s reason=%v", s.Ledger.Exposure.Shares, result.err)
			return
		}
		pnl := s.Ledger.hedge(result.sweep)
		s.Phase = idle
		s.log("[HEDGE_LOSS] realized=%s", pnl.StringFixed(5))
	}
}

func (s *runState) clearDeadline() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.cancelB != nil {
		s.cancelB()
		s.cancelB = nil
	}
}

func (s *runState) startB(ctx context.Context, io MarketIO, cfg Config, send func(event)) {
	call, cancel := context.WithDeadline(ctx, s.deadline)
	s.cancelB = cancel
	id, target, pending := s.orderID, s.Ledger.Exposure.Shares, s.pending
	session := io.Status().Session
	go func() {
		defer cancel()
		sweep, err := executeB(call, io, cfg, id, target, pending)
		send(event{kind: "b", id: id, session: session, sweep: sweep, finished: time.Now(), err: err})
	}()
}

func executeB(ctx context.Context, io MarketIO, cfg Config, id uint64, target decimal.Decimal, pending pendingTrade) (Sweep, error) {
	snapshot, err := io.Snapshot(ctx)
	if err != nil {
		return Sweep{}, err
	}
	if _, err := Sync(snapshot); err != nil {
		return Sweep{}, err
	}
	for i, leg := range snapshot.Legs {
		if leg.Token != pending.tokens[i] {
			return Sweep{}, errors.New("market token changed")
		}
		if err := ValidateLeg(snapshot, leg, time.Now(), cfg.MaxAge); err != nil {
			return Sweep{}, err
		}
	}
	leg := snapshot.Legs[1]
	request := fillRequest(id, "b", snapshot, leg, "buy", target, pending.bLimit)
	expected, err := bookSweep(leg, "buy", target, pending.bLimit, "taker")
	if err != nil {
		return Sweep{}, err
	}
	if expected.Notional.LessThan(one) {
		return Sweep{}, errors.New("available B execution below $1")
	}
	return execute(ctx, io, request, leg)
}

func (s *runState) startHedge(ctx context.Context, io MarketIO, cfg Config, send func(event)) {
	s.clearDeadline()
	s.Phase = hedging
	id, exposure := s.orderID, s.Ledger.Exposure
	session := io.Status().Session
	go func() {
		call, cancel := context.WithTimeout(ctx, cfg.HedgeTimeout)
		defer cancel()
		sweep, err := executeHedge(call, io, cfg, id, exposure)
		send(event{kind: "hedge", id: id, session: session, sweep: sweep, finished: time.Now(), err: err})
	}()
}

func executeHedge(ctx context.Context, io MarketIO, cfg Config, id uint64, exposure Position) (Sweep, error) {
	snapshot, err := io.Snapshot(ctx)
	if err != nil {
		return Sweep{}, err
	}
	if len(snapshot.Legs) != 2 || snapshot.Legs[0].Token != exposure.Tokens[0] {
		return Sweep{}, errors.New("missing current hedge book")
	}
	leg := snapshot.Legs[0]
	if err := ValidateLeg(snapshot, leg, time.Now(), cfg.MaxAge); err != nil {
		return Sweep{}, err
	}
	sweep, err := bookSweep(leg, "sell", exposure.Shares, decimal.Zero, "taker")
	if err != nil {
		return Sweep{}, err
	}
	if !sweep.Shares.Equal(exposure.Shares) {
		return Sweep{}, ErrDepth
	}
	request := fillRequest(id, "hedge", snapshot, leg, "sell", exposure.Shares, sweep.Limit())
	return execute(ctx, io, request, leg)
}

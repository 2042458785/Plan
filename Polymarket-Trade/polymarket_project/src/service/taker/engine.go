package taker

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"Polymarket/src/pkg/papermcp"

	"github.com/shopspring/decimal"
)

const (
	PollInterval = 100 * time.Millisecond
	MaxSkew      = 200 * time.Millisecond
	HedgeAfter   = 500 * time.Millisecond
	MakerBlocked = "[ERROR] Maker blocked: paper-trader does not simulate FIFO queue"
)

const maxExchangeMS int64 = 8_000_000_000_000

var (
	one        = decimal.NewFromInt(1)
	half       = decimal.New(5, -1)
	ErrLocalTS = errors.New("[LOCAL_TS] missing exchange timestamp")
	ErrSkew    = errors.New("exchange timestamps differ by 200ms or more")
	ErrStale   = errors.New("stale order book")
	ErrDepth   = errors.New("insufficient depth")
)

type Sweep struct {
	Fills    []papermcp.Fill
	Shares   decimal.Decimal
	Notional decimal.Decimal
	Fees     decimal.Decimal
}

func (s Sweep) Cost() decimal.Decimal { return s.Notional.Add(s.Fees) }

func (s Sweep) VWAP() decimal.Decimal {
	if s.Shares.IsZero() {
		return decimal.Zero
	}
	return s.Notional.DivRound(s.Shares, 10)
}

func (s Sweep) Limit() decimal.Decimal {
	return s.Fills[len(s.Fills)-1].Price
}

type Quote struct {
	Legs     [2]Sweep
	NetFloor decimal.Decimal
}

func Sync(s papermcp.Snapshot) (time.Duration, error) {
	if len(s.Legs) != 2 || s.Legs[0].ReceiveAgeMS == nil || s.Legs[1].ReceiveAgeMS == nil {
		return 0, errors.New("waiting for both books")
	}
	a, b := s.Legs[0].ExchangeMS, s.Legs[1].ExchangeMS
	if a == nil || b == nil || *a <= 0 || *b <= 0 || *a >= maxExchangeMS || *b >= maxExchangeMS {
		return 0, ErrLocalTS
	}
	skew := (time.Duration(*a-*b) * time.Millisecond).Abs()
	if skew >= MaxSkew {
		return skew, ErrSkew
	}
	return skew, nil
}

func ValidateLeg(s papermcp.Snapshot, leg papermcp.Leg, now time.Time, maxAge time.Duration) error {
	if !s.WS || s.Generation == "" || s.ID == "" || !leg.Accepting || leg.Token == "" {
		return errors.New("market feed or trading unavailable")
	}
	if leg.ExchangeMS == nil || *leg.ExchangeMS <= 0 || *leg.ExchangeMS >= maxExchangeMS {
		return ErrLocalTS
	}
	if leg.ReceiveAgeMS == nil || *leg.ReceiveAgeMS < 0 || *leg.ReceiveAgeMS > maxAge.Milliseconds() ||
		s.RequestedAt.IsZero() || now.Before(s.RequestedAt) {
		return ErrStale
	}
	receiveAge := time.Duration(*leg.ReceiveAgeMS) * time.Millisecond
	exchangeAge := now.UnixMilli() - *leg.ExchangeMS
	if now.Sub(s.RequestedAt) > maxAge-receiveAge || exchangeAge > maxAge.Milliseconds() || exchangeAge < -maxAge.Milliseconds() {
		return ErrStale
	}
	if !leg.Tick.IsPositive() || leg.Tick.GreaterThanOrEqual(one) || !leg.Minimum.IsPositive() ||
		leg.Rate == nil || leg.Rate.IsNegative() || leg.Rate.GreaterThanOrEqual(one) {
		return errors.New("invalid or unknown tick, minimum, or fee rate")
	}
	return nil
}

func bookSweep(leg papermcp.Leg, side string, shares, limit decimal.Decimal, intent string) (Sweep, error) {
	var result Sweep
	if intent != "taker" {
		return result, errors.New(MakerBlocked)
	}
	if side != "buy" && side != "sell" {
		return result, errors.New("invalid side")
	}
	if !leg.Tick.IsPositive() || !leg.Minimum.IsPositive() || !shares.IsPositive() ||
		shares.LessThan(leg.Minimum) || leg.Rate == nil || leg.Rate.IsNegative() || leg.Rate.GreaterThanOrEqual(one) {
		return result, errors.New("invalid market parameters or quantity below minimum")
	}
	if !limit.IsZero() && (!limit.IsPositive() || !limit.LessThan(one) || !limit.Mod(leg.Tick).IsZero()) {
		return result, errors.New("price limit is off tick")
	}
	levels := leg.Asks
	if side == "sell" {
		levels = leg.Bids
	}
	levels = slices.Clone(levels)
	slices.SortFunc(levels, func(a, b papermcp.Level) int {
		if side == "sell" {
			return b.Price.Cmp(a.Price)
		}
		return a.Price.Cmp(b.Price)
	})
	remaining := shares
	var previous decimal.Decimal
	for _, level := range levels {
		if !level.Price.IsPositive() || !level.Price.LessThan(one) || !level.Price.Mod(leg.Tick).IsZero() ||
			!level.Size.IsPositive() || level.Price.Equal(previous) {
			return Sweep{}, errors.New("invalid, duplicate, or off-tick level")
		}
		previous = level.Price
		if !limit.IsZero() && ((side == "buy" && level.Price.GreaterThan(limit)) ||
			(side == "sell" && level.Price.LessThan(limit))) {
			break
		}
		fill := decimal.Min(remaining, level.Size)
		fee := fill.Mul(*leg.Rate).Mul(level.Price).Mul(one.Sub(level.Price)).Round(5)
		result.Fills = append(result.Fills, papermcp.Fill{Price: level.Price, Shares: fill, Fee: fee})
		result.Notional = result.Notional.Add(fill.Mul(level.Price))
		result.Fees = result.Fees.Add(fee)
		result.Shares = result.Shares.Add(fill)
		remaining = remaining.Sub(fill)
		if remaining.IsZero() {
			break
		}
	}
	return result, nil
}

func CalculateVWAP(s papermcp.Snapshot, shares, payoutFloor, balance decimal.Decimal, now time.Time, maxAge time.Duration) (Quote, error) {
	var quote Quote
	if _, err := Sync(s); err != nil {
		return quote, err
	}
	if s.Legs[0].Token == s.Legs[1].Token {
		return quote, errors.New("legs must have distinct tokens")
	}
	if !payoutFloor.IsPositive() || payoutFloor.GreaterThan(decimal.NewFromInt(2)) {
		return quote, errors.New("invalid reviewed payout floor")
	}
	for i, leg := range s.Legs {
		if err := ValidateLeg(s, leg, now, maxAge); err != nil {
			return quote, fmt.Errorf("leg %d: %w", i+1, err)
		}
		sweep, err := bookSweep(leg, "buy", shares, decimal.Zero, "taker")
		if err != nil {
			return quote, fmt.Errorf("leg %d: %w", i+1, err)
		}
		if !sweep.Shares.Equal(shares) {
			return quote, fmt.Errorf("leg %d: %w", i+1, ErrDepth)
		}
		if sweep.Notional.LessThan(one) {
			return quote, fmt.Errorf("leg %d: market buy below $1", i+1)
		}
		quote.Legs[i] = sweep
	}
	cost := quote.Legs[0].Cost().Add(quote.Legs[1].Cost())
	if cost.GreaterThan(balance) {
		return quote, errors.New("virtual balance insufficient")
	}
	quote.NetFloor = shares.Mul(payoutFloor).Sub(cost)
	return quote, nil
}

func checkFill(request papermcp.FillRequest, leg papermcp.Leg, result papermcp.FillResult) (Sweep, error) {
	if result.RequestID != request.RequestID || result.Generation != request.Generation ||
		result.SnapshotID != request.SnapshotID || result.Token != request.Token || result.Side != request.Side {
		return Sweep{}, errors.New("fill identity mismatch")
	}
	expected, err := bookSweep(leg, request.Side, request.Shares, request.Limit, request.Intent)
	if err != nil {
		return Sweep{}, err
	}
	status := "partial"
	if expected.Shares.Equal(request.Shares) {
		status = "filled"
	} else if expected.Shares.IsZero() {
		status = "unfilled"
	}
	if status != result.Status || len(result.Fills) != len(expected.Fills) {
		return Sweep{}, errors.New("simulated fill does not match pinned book")
	}
	for i, fill := range result.Fills {
		match := expected.Fills[i]
		if !fill.Price.Equal(match.Price) || !fill.Shares.Equal(match.Shares) || !fill.Fee.Equal(match.Fee) {
			return Sweep{}, errors.New("simulated price, quantity, or fee mismatch")
		}
	}
	return expected, nil
}

type Position struct {
	Tokens [2]string
	Shares decimal.Decimal
	Cost   decimal.Decimal
}

type Ledger struct {
	Cash     decimal.Decimal
	Realized decimal.Decimal
	Fees     decimal.Decimal
	Paired   Position
	Exposure Position
}

func (l *Ledger) confirmA(tokens [2]string, sweep Sweep) {
	l.Cash = l.Cash.Sub(sweep.Cost())
	l.Fees = l.Fees.Add(sweep.Fees)
	l.Exposure = Position{Tokens: tokens, Shares: sweep.Shares, Cost: sweep.Cost()}
}

func (l *Ledger) confirmB(sweep Sweep) {
	if sweep.Shares.IsZero() {
		return
	}
	basis := l.Exposure.Cost
	if sweep.Shares.LessThan(l.Exposure.Shares) {
		basis = basis.Mul(sweep.Shares).DivRound(l.Exposure.Shares, 18)
	}
	l.Cash = l.Cash.Sub(sweep.Cost())
	l.Fees = l.Fees.Add(sweep.Fees)
	l.Paired = Position{Tokens: l.Exposure.Tokens, Shares: sweep.Shares, Cost: basis.Add(sweep.Cost())}
	l.Exposure.Shares = l.Exposure.Shares.Sub(sweep.Shares)
	l.Exposure.Cost = l.Exposure.Cost.Sub(basis)
}

func (l *Ledger) hedge(sweep Sweep) decimal.Decimal {
	net := sweep.Notional.Sub(sweep.Fees)
	pnl := net.Sub(l.Exposure.Cost)
	l.Cash = l.Cash.Add(net)
	l.Fees = l.Fees.Add(sweep.Fees)
	l.Realized = l.Realized.Add(pnl)
	l.Exposure = Position{}
	return pnl
}

func (l *Ledger) settle(s papermcp.Snapshot) (decimal.Decimal, bool) {
	position := l.Paired
	if position.Shares.IsZero() || len(s.Legs) != 2 {
		return decimal.Zero, false
	}
	payout := decimal.Zero
	for i, leg := range s.Legs {
		if leg.Token != position.Tokens[i] || leg.Payout == nil ||
			!(leg.Payout.IsZero() || leg.Payout.Equal(one) || leg.Payout.Equal(half)) {
			return decimal.Zero, false
		}
		payout = payout.Add(*leg.Payout)
	}
	payout = payout.Mul(position.Shares)
	pnl := payout.Sub(position.Cost)
	l.Cash = l.Cash.Add(payout)
	l.Realized = l.Realized.Add(pnl)
	l.Paired = Position{}
	return pnl, true
}

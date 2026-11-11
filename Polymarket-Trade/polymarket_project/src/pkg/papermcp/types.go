package papermcp

import (
	"time"

	"github.com/shopspring/decimal"
)

type Level struct {
	Price decimal.Decimal `json:"price"`
	Size  decimal.Decimal `json:"size"`
}

type Leg struct {
	RulesFingerprint string           `json:"rules_fingerprint"`
	Slug             string           `json:"slug"`
	Outcome          string           `json:"outcome"`
	Token            string           `json:"token"`
	ExchangeMS       *int64           `json:"exchange_ms"`
	ReceiveAgeMS     *int64           `json:"receive_age_ms"`
	Tick             decimal.Decimal  `json:"tick_size"`
	Minimum          decimal.Decimal  `json:"min_order_size"`
	Rate             *decimal.Decimal `json:"fee_rate"`
	Accepting        bool             `json:"accepting"`
	Payout           *decimal.Decimal `json:"payout"`
	Asks             []Level          `json:"asks"`
	Bids             []Level          `json:"bids"`
}

type MarketTrade struct {
	Token      string           `json:"token"`
	Side       string           `json:"side"`
	Price      decimal.Decimal  `json:"price"`
	Size       *decimal.Decimal `json:"size"`
	ExchangeMS int64            `json:"exchange_ms"`
}

type Snapshot struct {
	Trades      []MarketTrade `json:"trades"`
	WSRTTMS     *int64        `json:"ws_rtt_ms"`
	FeedError   string        `json:"feed_error"`
	Session     uint64        `json:"-"`
	RequestedAt time.Time     `json:"-"`
	Generation  string        `json:"generation"`
	ID          string        `json:"snapshot_id"`
	WS          bool          `json:"ws_ok"`
	Legs        []Leg         `json:"legs"`
}

type FillRequest struct {
	RequestID  string          `json:"request_id"`
	Session    uint64          `json:"-"`
	Generation string          `json:"generation"`
	SnapshotID string          `json:"snapshot_id"`
	Token      string          `json:"token"`
	Side       string          `json:"side"`
	Shares     decimal.Decimal `json:"shares"`
	Limit      decimal.Decimal `json:"limit"`
	Intent     string          `json:"intent"`
}

type Fill struct {
	Price  decimal.Decimal `json:"price"`
	Shares decimal.Decimal `json:"shares"`
	Fee    decimal.Decimal `json:"fee"`
}

type FillResult struct {
	RequestID  string `json:"request_id"`
	Generation string `json:"generation"`
	SnapshotID string `json:"snapshot_id"`
	Token      string `json:"token"`
	Side       string `json:"side"`
	Status     string `json:"status"`
	Fills      []Fill `json:"fills"`
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"Polymarket/src/pkg/papermcp"

	tea "charm.land/bubbletea/v2"
)

type bookPreview struct {
	market  catalogMarket
	yes     bool
	loading bool
	book    *papermcp.Leg
	err     string
	next    time.Time
	cancel  context.CancelFunc
}

type previewResult struct {
	id   uint64
	book *papermcp.Leg
	err  error
}

func (m *model) showPreview(market catalogMarket) tea.Cmd {
	m.clearPreview()
	m.preview = &bookPreview{market: market, yes: true}
	m.bookOffset, m.bookFocus = 0, true
	return m.refreshPreview()
}

func (m *model) clearPreview() {
	if m.preview != nil && m.preview.cancel != nil {
		m.preview.cancel()
	}
	m.previewID++
	m.preview = nil
}

func (m *model) refreshPreview() tea.Cmd {
	p := m.preview
	if p == nil || p.loading {
		return nil
	}
	ctx, cancel := context.WithTimeout(m.ctx, 12*time.Second)
	p.loading, p.cancel, p.err = true, cancel, ""
	m.previewID++
	id, slug, side, l := m.previewID, p.market.Slug, outcome(p.yes), m.form.language
	return func() tea.Msg {
		defer cancel()
		output, err := runCatalog(ctx, l, "--book", slug, side)
		if err != nil {
			return previewResult{id: id, err: err}
		}
		var book papermcp.Leg
		if err := json.Unmarshal(output, &book); err != nil {
			return previewResult{id: id, err: err}
		}
		if book.Slug != slug || book.Outcome != side || !book.Tick.IsPositive() {
			return previewResult{id: id, err: fmt.Errorf("invalid preview book")}
		}
		slices.SortFunc(book.Asks, func(a, b papermcp.Level) int { return a.Price.Cmp(b.Price) })
		slices.SortFunc(book.Bids, func(a, b papermcp.Level) int { return b.Price.Cmp(a.Price) })
		return previewResult{id: id, book: &book}
	}
}

func (m *model) flipPreview() tea.Cmd {
	if m.preview == nil {
		return nil
	}
	p := m.preview
	if p.cancel != nil {
		p.cancel()
	}
	p.yes, p.book, p.loading = !p.yes, nil, false
	m.bookOffset = 0
	return m.refreshPreview()
}

func (m *model) displayedLeg() papermcp.Leg {
	if m.preview != nil {
		if m.preview.book != nil {
			return *m.preview.book
		}
		return papermcp.Leg{}
	}
	return m.leg(m.bookLeg)
}

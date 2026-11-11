package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"Polymarket/src/pkg/papermcp"
	"Polymarket/src/service/taker"

	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
)

func TestSetupUsesEnteredBalanceAndDemoSettings(t *testing.T) {
	form := newSetup()
	form.move(1)
	form.edit(form.value(), "5")
	params, err := form.params()
	if err != nil {
		t.Fatal(err)
	}
	if params.config.Balance.String() != "5" || !params.config.Reviewed || !slices.Contains(params.args, "--fixture") {
		t.Fatalf("unexpected session: %+v", params)
	}
}

func TestLiveSetupRequiresReviewBeforePaper(t *testing.T) {
	form := newSetup()
	form.mode, form.a, form.b, form.yesA = liveObserve, "market-a", "market-b", true
	params, err := form.params()
	if err != nil {
		t.Fatal(err)
	}
	if !params.config.ObserveOnly || params.config.Reviewed || params.args[5] != "Yes" {
		t.Fatalf("unexpected observation session: %+v", params)
	}
	form.mode, form.payout = livePaper, "1"
	if _, err := form.params(); err == nil {
		t.Fatal("unreviewed paper setup accepted")
	}
	form.reviewed = true
	params, err = form.params()
	if err != nil || params.config.ObserveOnly || !params.config.Reviewed {
		t.Fatalf("reviewed paper setup: %+v %v", params, err)
	}
	form.focus = 1
	form.edit(form.value(), "-changed")
	if form.reviewed {
		t.Fatal("edited market retained prior review")
	}
}

func TestOrderBookDepthShowsTickPrecisionAndNotional(t *testing.T) {
	value := decimal.RequireFromString
	rows := depthSide([]papermcp.Level{
		{Price: value("0.4"), Size: value("20")},
		{Price: value("0.399"), Size: value("30")},
	}, 0, 5, 28, 3, styles.buy)
	if len(rows) != 5 || !strings.Contains(ansi.Strip(rows[0]), "0.400") || !strings.Contains(ansi.Strip(rows[0]), "8.00") ||
		!strings.Contains(ansi.Strip(rows[1]), "0.399") || !strings.Contains(rows[0], styles.buy.Render(" 0.400")) {
		t.Fatalf("unexpected depth display: %q", rows)
	}
}

func TestDashboardShowsIndependentNetworkRTT(t *testing.T) {
	rtt, age := int64(27), int64(0)
	stampA, stampB := int64(1800000000000), int64(1800000000300)
	m := model{width: 120, height: 24, session: &session{}, form: newSetup(), value: taker.View{
		Snapshot: papermcp.Snapshot{WS: true, WSRTTMS: &rtt, Legs: []papermcp.Leg{
			{ExchangeMS: &stampA, ReceiveAgeMS: &age},
			{ExchangeMS: &stampB, ReceiveAgeMS: &age},
		}},
	}}
	wide := ansi.Strip(m.dashboardHeader())
	if !strings.Contains(wide, "时间差:300ms") || !strings.Contains(wide, "行情往返:27ms") {
		t.Fatalf("missing independent metrics: %s", wide)
	}
	m.width = 66
	narrow := m.dashboardHeader()
	if strings.Contains(narrow, "\n") || !strings.Contains(narrow, "行情往返:27ms") {
		t.Fatalf("latency line does not fit: %v", narrow)
	}
	m.value.Snapshot.WS = false
	if lines := m.dashboardHeader(); !strings.Contains(lines, "行情往返:--") {
		t.Fatalf("disconnected feed exposed an old measurement: %s", lines)
	}
}

func TestWorkspaceFitsTerminal(t *testing.T) {
	price, size, tick := decimal.New(48, -2), decimal.NewFromInt(100), decimal.New(1, -2)
	leg := papermcp.Leg{Slug: "bitcoin-above-74k-on-november-11-2026", Outcome: "Yes", Tick: tick,
		Asks: []papermcp.Level{{Price: price, Size: size}}, Bids: []papermcp.Level{{Price: price, Size: size}}}
	for _, width := range []int{66, 82, 136} {
		m := model{width: width, height: 26, session: &session{}, form: newSetup(), value: taker.View{
			Snapshot: papermcp.Snapshot{WS: true, Legs: []papermcp.Leg{leg, leg}},
			Logs:     []string{"22:10:01.000 [A FILL] shares=5 cost=2.46240"},
		}}
		view := m.workspaceView()
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("%d cols: line wraps: %q", width, ansi.Strip(line))
			}
		}
		if len(strings.Split(view, "\n")) != m.height || !strings.Contains(ansi.Strip(view), "订单簿 · A") ||
			!strings.Contains(ansi.Strip(view), "模拟账户") {
			t.Fatalf("%d cols: incomplete workspace", width)
		}
	}
}

func TestCatalogAssignsSelectedMarketAndInvalidatesOldQuery(t *testing.T) {
	m := model{form: newSetup()}
	m.form.mode, m.form.a, m.form.reviewed = livePaper, "keep-a", true
	m.form.focus = 3
	m.openCatalog()
	m.catalog.events = []catalogEvent{{Markets: []catalogMarket{{Slug: "selected-b"}}}}
	m.catalog.pane = 2
	id := m.catalogID
	m.catalogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.catalogActive || m.form.a != "keep-a" || m.form.b != "selected-b" || m.form.reviewed {
		t.Fatalf("bad assignment: %+v", m.form)
	}
	m.openCatalog()
	m.Update(catalogResult{id: id, events: []catalogEvent{{Title: "obsolete"}}})
	if len(m.catalog.events) != 1 || m.catalog.events[0].Markets[0].Slug != "selected-b" {
		t.Fatal("reopened catalog accepted an obsolete search")
	}
}

func TestMarketUpdatesCoalesceUntilUITick(t *testing.T) {
	m := model{width: 136, height: 26, languageChosen: true, session: &session{}, form: newSetup()}
	start := m.View().Content
	for _, slug := range []string{"first-book", "latest-book"} {
		m.Update(viewMessage(taker.View{Snapshot: papermcp.Snapshot{
			Legs: []papermcp.Leg{{Slug: slug, Outcome: "Yes"}},
		}}))
		if m.View().Content != start {
			t.Fatal("market message updated the view before the UI tick")
		}
	}
	m.Update(uiTick(time.Now()))
	after := m.View()
	if !strings.Contains(after.Content, "latest-book") || strings.Contains(after.Content, "first-book") || after.MouseMode != tea.MouseModeNone {
		t.Fatal("the UI tick did not publish the latest snapshot")
	}
}

func TestActionHistoryKeepsLast50RelevantEntries(t *testing.T) {
	var entries []string
	for range 60 {
		entries = append(entries, "12:00:00.001 [SIGNAL] trade", "12:00:00.002 quote updated")
	}
	last := "12:00:00.003 [HEDGE_FAILED] reason"
	entries = append(entries, last)
	actions := recentActions(entries)
	if len(actions) != 50 || actions[49] != last || strings.Contains(strings.Join(actions, "\n"), "quote updated") {
		t.Fatal("viewport retained noise or discarded the most recent risk action")
	}
}

func TestControlStatusChangesWithoutWaitingForMarketTick(t *testing.T) {
	m := model{width: 120, height: 24, languageChosen: true,
		session: &session{controls: make(chan taker.Command, 2)}, form: newSetup()}
	for _, test := range []struct{ key, want string }{{"p", "暂停请求"}, {"r", "重连中"}, {"q", "正在退出"}} {
		m.Update(tea.KeyPressMsg{Code: rune(test.key[0]), Text: test.key})
		if !strings.Contains(m.View().Content, test.want) {
			t.Fatalf("%s did not immediately show %s", test.key, test.want)
		}
	}
	if <-m.session.controls != taker.TogglePause || <-m.session.controls != taker.Reconnect {
		t.Fatal("unexpected control commands")
	}
}

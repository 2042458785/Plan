package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"Polymarket/src/pkg/papermcp"
	"Polymarket/src/service/taker"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/charmbracelet/x/ansi"
	"github.com/shopspring/decimal"
)

type terminalStyles struct {
	screen, text, title, border, buy, sell, warning lipgloss.Style
}

var styles = terminalStyles{
	screen:  lipgloss.NewStyle().Background(lipgloss.Color("#000000")),
	text:    lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")),
	title:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
	border:  lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")),
	buy:     lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
	sell:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")),
	warning: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")),
}

func (m *model) View() tea.View {
	if m.frameValid {
		return m.frame
	}
	var content string
	switch {
	case !m.languageChosen:
		content = m.languageView()
	case m.session == nil && !m.catalogActive:
		content = m.setupView()
	default:
		content = m.workspaceView()
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.BackgroundColor = styles.screen.GetBackground()
	view.ForegroundColor = styles.text.GetForeground()
	view.MouseMode = tea.MouseModeNone
	m.frame, m.frameValid = view, true
	return view
}

func fillStyled(value string, width int) string {
	value = ansi.Truncate(value, max(0, width), "")
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

func drawPane(title string, rows []string, width, height int, focused bool) []string {
	if focused {
		title = "> " + title
	}
	heading := " " + trim(title, width-6) + " "
	border := styles.border.Render
	lines := []string{border("┌─") + styles.title.Render(heading) +
		border(strings.Repeat("─", width-3-ansi.StringWidth(heading))+"┐")}
	edge := border("│")
	for i := range height - 2 {
		row := ""
		if i < len(rows) {
			row = rows[i]
		}
		lines = append(lines, edge+fillStyled(row, width-2)+edge)
	}
	return append(lines, border("└"+strings.Repeat("─", width-2)+"┘"))
}

func joinPanes(panes ...[]string) []string {
	rows := make([]string, len(panes[0]))
	for i := range rows {
		line := make([]string, len(panes))
		for j, pane := range panes {
			line[j] = pane[i]
		}
		rows[i] = strings.Join(line, " ")
	}
	return rows
}

func (m *model) workspaceView() string {
	l := m.form.language
	header, footer := m.dashboardHeader(), m.workspaceFooter()
	topHeight, bottomHeight := m.paneHeights()
	if m.width < 66 || topHeight < 8 || bottomHeight < 7 {
		lines := []string{header}
		if m.height >= 3 {
			lines = append(lines, trim(l.text("请放大终端以显示盘口和账户。", "Enlarge terminal to show the book and account."), m.width))
			lines = append(lines, make([]string, m.height-3)...)
		}
		return strings.Join(append(lines, footer), "\n")
	}
	bookWidth := m.bookWidth()
	sideWidth := m.width - bookWidth - 1
	bookTitle := l.text("订单簿", "ORDER BOOK") + fmt.Sprintf(" · %c", 'A'+m.bookLeg)
	if m.preview != nil {
		bookTitle = l.text("订单簿 · 只读预览 [O]", "ORDER BOOK · PREVIEW [O]")
	}
	book := drawPane(bookTitle, m.depthRows(bookWidth-2, topHeight-5), bookWidth, topHeight, !m.catalogActive && m.bookFocus)
	marketsTitle := l.text("市场列表", "MARKETS")
	marketFocus := m.catalogActive && m.catalog.pane == 2
	var top []string
	if m.width >= 110 {
		searchWidth := m.width * 29 / 100
		marketWidth := sideWidth - searchWidth - 1
		top = joinPanes(m.searchRows(searchWidth, topHeight),
			drawPane(marketsTitle, m.browseRows(marketWidth-2, topHeight-2), marketWidth, topHeight, marketFocus),
			book)
	} else if m.catalogActive && !marketFocus {
		top = joinPanes(m.searchRows(sideWidth, topHeight), book)
	} else {
		rows := m.browseRows(sideWidth-2, topHeight-3)
		top = joinPanes(drawPane(marketsTitle, rows, sideWidth, topHeight, marketFocus), book)
	}

	bottom := joinPanes(
		drawPane(l.text("成交与风控 [Tab ↑↓]", "EXECUTION & RISK [Tab ↑↓]"), m.executionRows(sideWidth-2, bottomHeight-2), sideWidth, bottomHeight, m.session != nil && !m.catalogActive && !m.bookFocus),
		drawPane(l.text("模拟账户 / 持仓", "PAPER ACCOUNT / POSITIONS"), m.accountRows(), bookWidth, bottomHeight, false),
	)
	lines := append([]string{header}, top...)
	lines = append(lines, bottom...)
	return strings.Join(append(lines, footer), "\n")
}

func (m *model) dashboardHeader() string {
	l, v := m.form.language, m.value
	if m.session == nil {
		return trim(l.text("POLYMARKET  /  选择模拟市场", "POLYMARKET / SELECT PAPER MARKETS"), m.width)
	}

	phase := l.diagnostic(v.Phase)
	if phase == "" {
		phase = l.text("等待信号", "IDLE")
	}
	switch {
	case m.quitting:
		phase = styles.warning.Render(l.text("正在退出", "EXITING"))
	case !m.pauseUntil.IsZero():
		phase = styles.warning.Render(l.text("暂停请求", "PAUSE REQUESTED"))
		if v.Paused {
			phase = styles.warning.Render(l.text("恢复请求", "RESUME REQUESTED"))
		}
	case v.Phase == "HEDGE FAILED" || v.Phase == "HEDGING" || v.Ledger.Exposure.Shares.IsPositive():
		phase = styles.warning.Render(phase)
	case v.Paused:
		phase = styles.warning.Render(l.text("已暂停", "PAUSED"))
	}

	mcp := styles.sell.Render("MCP:DOWN")
	switch {
	case !m.reconnectUntil.IsZero():
		mcp = styles.warning.Render(l.text("MCP:重连中", "MCP:RECONNECTING"))
	case v.MCP.Ready:
		mcp = "MCP:OK"
	case v.MCP.Session == 0 && v.MCP.Error == "":
		mcp = styles.warning.Render(l.text("MCP:启动中", "MCP:STARTING"))
	}
	ws := "WS:OK"
	if !m.reconnectUntil.IsZero() {
		ws = "WS:WAIT"
	} else if !v.Snapshot.WS {
		ws = styles.sell.Render("WS:DOWN")
	}
	syncValue := "--"
	syncStyle := styles.text
	if m.reconnectUntil.IsZero() {
		if v.LocalTS {
			syncValue, syncStyle = "[LOCAL_TS]", styles.warning
		} else if skew, err := taker.Sync(v.Snapshot); err == nil || errors.Is(err, taker.ErrSkew) {
			syncValue = skew.String()
			if err != nil {
				syncStyle = styles.warning
			}
		}
	}
	rtt := "--"
	if m.reconnectUntil.IsZero() && v.Snapshot.WS && v.Snapshot.WSRTTMS != nil {
		rtt = fmt.Sprintf("%dms", *v.Snapshot.WSRTTMS)
	}

	line := phase + " | " + mcp + " | " + ws
	for _, item := range []string{
		l.text("时间差:", "Sync:") + syncStyle.Render(syncValue),
		l.text("行情往返:", "Feed RTT:") + rtt,
		l.text("余额:$", "Cash:$") + money(v.Ledger.Cash),
		l.text("已实现:$", "Realized:$") + signedValue(v.Ledger.Realized, money(v.Ledger.Realized)),
	} {
		if ansi.StringWidth(line+" | "+item) <= m.width {
			line += " | " + item
		}
	}
	return ansi.Truncate(line, max(0, m.width), "")
}

func (m *model) workspaceFooter() string {
	l := m.form.language
	if m.session != nil {
		return trim(l.text("[P] 暂停/恢复 | [R] 重连 | [Q] 退出", "[P] Pause/Resume | [R] Reconnect | [Q] Quit"), m.width)
	}
	return trim(m.catalogHelp(), m.width)
}

func (m *model) catalogHelp() string {
	l := m.form.language
	if m.session != nil {
		return l.text("[Enter] 搜索/打开/预览 [O] 是/否 [Tab] 切换 [Esc] 返回", "[Enter] Search/Open/Preview [O] Yes/No [Tab] Focus [Esc] Back")
	}
	leg := "A"
	if m.catalog != nil && m.catalog.target == fieldB {
		leg = "B"
	}
	return l.text("[Enter] 搜索/打开/填入 "+leg+" 腿  [O] 预览  [Esc] 返回", "[Enter] Search/Open/Assign "+leg+"  [O] Preview  [Esc] Back")
}

func (m *model) bookWidth() int {
	if m.width >= 110 {
		return m.width - m.width*29/100 - m.width*34/100 - 2
	}
	return max(40, m.width/2)
}

func (m *model) paneHeights() (top, bottom int) {
	available := m.height - 2
	bottom = max(7, available/4)
	return available - bottom, bottom
}

func (m *model) executionSummary() string {
	l, q := m.form.language, m.value.Quote
	if q == nil {
		return ""
	}
	cost := q.Legs[0].Cost().Add(q.Legs[1].Cost())
	net := q.NetFloor.StringFixed(5)
	return l.text("含费成本 $", "All-in cost $") + cost.StringFixed(5) +
		l.text("  最低净额 ", "  Net floor ") + signedValue(q.NetFloor, net)
}

func (m *model) accountRows() []string {
	l := m.form.language
	account := m.value.Ledger
	return []string{
		l.text("可用  $", "Cash  $") + money(account.Cash),
		l.text("已实现  $", "Realized  $") + signedValue(account.Realized, account.Realized.StringFixed(5)),
		l.text("手续费  $", "Fees  $") + account.Fees.StringFixed(5),
		l.text("配对 / 单腿  ", "Paired / exposed  ") + account.Paired.Shares.String() + " / " + exposureValue(account.Exposure.Shares),
		l.text("持仓成本  $", "Position cost  $") + account.Paired.Cost.Add(account.Exposure.Cost).StringFixed(5),
	}
}

func wrapPlain(text string, width int) []string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	return strings.Split(ansi.Wrap(clean, max(1, width), ""), "\n")
}

const maxActionLogs = 50

// Filter the engine's rolling history into the UI viewport without changing engine logs.
func recentActions(entries []string) []string {
	rows := make([]string, 0, min(maxActionLogs, len(entries)))
	for i := len(entries) - 1; i >= 0 && len(rows) < maxActionLogs; i-- {
		if _, ok := actionStyle(entries[i]); ok {
			rows = append(rows, entries[i])
		}
	}
	slices.Reverse(rows)
	return rows
}

func actionStyle(entry string) (lipgloss.Style, bool) {
	_, message, _ := strings.Cut(entry, " ")
	switch {
	case strings.HasPrefix(message, "[HEDGE"), strings.HasPrefix(message, "[WARNING]"),
		strings.HasPrefix(message, "[LOCAL_TS]"), strings.HasPrefix(message, "Market rules changed"),
		strings.HasPrefix(message, "Risk paused"), strings.HasPrefix(message, "WS: DOWN"),
		strings.HasPrefix(message, "MCP: DEAD"), strings.HasPrefix(message, "[A REJECTED]"),
		strings.HasPrefix(message, "[B REJECTED]"), strings.HasPrefix(message, "[B UNCONFIRMED]"):
		return styles.warning, true
	case strings.HasPrefix(message, "[A FILL]"), strings.HasPrefix(message, "[B FILL]"):
		return styles.buy, true
	case strings.HasPrefix(message, "[SIGNAL]"), strings.HasPrefix(message, "[SETTLED]"),
		strings.HasPrefix(message, "New signals paused="), message == "MCP reconnect requested":
		return styles.text, true
	default:
		return styles.text, false
	}
}

func (m *model) executionSummaryRows(width int) []string {
	var rows []string
	if m.catalogActive && m.session != nil {
		rows = append(rows, trim(m.catalogHelp(), width))
	} else if reason := m.value.Reason; reason != "" && m.value.Quote == nil {
		style := styles.text
		if reason == taker.ErrSkew.Error() || strings.Contains(reason, taker.ErrStale.Error()) || m.value.LocalTS {
			style = styles.warning
		}
		rows = append(rows, style.Render(trim(m.form.language.diagnostic(reason), width)))
	}
	if summary := m.executionSummary(); summary != "" {
		rows = append(rows, summary)
	}
	return rows
}

func (m *model) executionRows(width, limit int) []string {
	l := m.form.language
	rows := m.executionSummaryRows(width)
	logs := m.value.Logs
	if len(logs) == 0 {
		return append(rows, l.text("等待信号、成交或风控", "Waiting for signal, fill, or risk event"))
	}
	limit = max(0, limit-len(rows))
	end := len(logs) - min(m.logOffset, max(0, len(logs)-limit))
	for _, entry := range logs[max(0, end-limit):end] {
		ink, _ := actionStyle(entry)
		rows = append(rows, ink.Render(trim(l.diagnostic(entry), width)))
	}
	return rows
}

func (m *model) leg(index int) papermcp.Leg {
	if index < len(m.value.Snapshot.Legs) {
		return m.value.Snapshot.Legs[index]
	}
	return papermcp.Leg{}
}

func (m *model) depthRows(width, count int) []string {
	l := m.form.language
	leg := m.displayedLeg()
	if leg.Slug == "" {
		if p := m.preview; p != nil {
			prompt := l.text("正在获取盘口…", "Loading order book…")
			if p.err != "" {
				prompt = styles.warning.Render(trim(p.err, width))
			}
			return []string{l.yesNo(p.yes) + " · " + trim(p.market.Label, width-6), prompt}
		}
		return []string{l.text("等待盘口", "Waiting for order book")}
	}
	precision := tickPrecision(leg.Tick)
	label := leg.Slug
	if m.preview != nil {
		label = m.preview.market.Label
	}
	name := l.yesNo(leg.Outcome == "Yes") + " · " + label
	sideWidth := (width - 1) / 2
	rows := []string{trim(name, width), styles.buy.Render(pad(l.text("买盘", "BIDS"), sideWidth)) + " " + styles.sell.Render(l.text("卖盘", "ASKS"))}
	columns := depthColumns(l, sideWidth, precision)
	rows = append(rows, styles.title.Render(columns+" "+columns))
	maximumOffset := max(0, max(len(leg.Bids), len(leg.Asks))-count)
	offset := min(m.bookOffset, maximumOffset)
	bids := depthSide(leg.Bids, offset, count, sideWidth, precision, styles.buy)
	asks := depthSide(leg.Asks, offset, count, sideWidth, precision, styles.sell)
	return append(rows, joinPanes(bids, asks)...)
}

func tickPrecision(tick decimal.Decimal) int32 {
	_, fraction, _ := strings.Cut(tick.String(), ".")
	return int32(len(fraction))
}

func depthColumns(l uiLanguage, width int, precision int32) string {
	return depthCells(l.text("价格", "PX"), l.text("份额", "SIZE"), l.text("金额", "USD"), width, precision, lipgloss.NewStyle())
}

func depthCells(price, size, usd string, width int, precision int32, priceStyle lipgloss.Style) string {
	priceWidth := min(width-5, max(6, int(precision)+2))
	sizeWidth := width - priceWidth - 1
	notional := ""
	if sizeWidth >= 14 {
		notional = " " + depthNumber(usd, 8)
		sizeWidth -= 9
	}
	return priceStyle.Render(depthNumber(price, priceWidth)) + " " + depthNumber(size, sizeWidth) + notional
}

func depthNumber(value string, width int) string {
	return right(ansi.Truncate(value, width, "…"), width)
}

func depthSide(levels []papermcp.Level, offset, count, width int, precision int32, priceStyle lipgloss.Style) []string {
	rows := make([]string, count)
	for i := range count {
		at := i + offset
		if at >= len(levels) {
			rows[i] = strings.Repeat(" ", width)
			continue
		}
		level := levels[at]
		rows[i] = depthCells(level.Price.StringFixed(precision), level.Size.StringFixed(2),
			level.Price.Mul(level.Size).StringFixed(2), width, precision, priceStyle)
	}
	return rows
}

func (m *model) scrollPane(delta int) {
	top, bottom := m.paneHeights()
	if m.bookFocus {
		leg := m.displayedLeg()
		last := max(0, max(len(leg.Bids), len(leg.Asks))-max(1, top-5))
		m.bookOffset = min(last, max(0, m.bookOffset+delta))
	} else {
		width := max(1, m.width-m.bookWidth()-3)
		limit := max(1, bottom-2-len(m.executionSummaryRows(width)))
		last := max(0, len(m.value.Logs)-limit)
		m.logOffset = min(last, max(0, m.logOffset-delta))
	}
}

func signedValue(value decimal.Decimal, display string) string {
	switch {
	case value.IsPositive():
		return styles.buy.Render(display)
	case value.IsNegative():
		return styles.sell.Render(display)
	default:
		return display
	}
}

func exposureValue(value decimal.Decimal) string {
	if value.IsPositive() {
		return styles.warning.Render(value.String())
	}
	return value.String()
}

func (m *model) languageView() string {
	lines := []string{styles.title.Render("选择界面语言 / Choose a language")}
	for i, name := range []string{"简体中文", "English"} {
		if uiLanguage(i) == m.form.language {
			lines = append(lines, styles.title.Render("> "+name))
		} else {
			lines = append(lines, "  "+name)
		}
	}
	return strings.Join(append(lines, "[↑/↓] 选择 / Select   [Enter] 确认 / Confirm"), "\n")
}

func (m *model) setupView() string {
	l := m.form.language
	lines := []string{styles.title.Render(l.text("POLYMARKET  /  模拟交易设置", "POLYMARKET / PAPER SESSION"))}
	width := min(90, max(20, m.width))
	for i, id := range m.form.fields() {
		label, value := m.form.display(id)
		if value == "" {
			value = "-"
		}
		line := pad(label, 24) + " " + value
		if i == m.form.focus {
			lines = append(lines, styles.title.Render(trim("> "+line, width)))
		} else {
			lines = append(lines, trim("  "+line, width))
		}
		if id == fieldMode {
			mode := modes[m.form.mode]
			lines = append(lines, ansi.Wrap(l.text(mode.zhHelp, mode.enHelp), width, ""))
		}
	}
	if m.form.mode == liveObserve || m.form.mode == livePaper {
		lines = append(lines, ansi.Wrap(l.text("在 A/B 市场字段按 Ctrl+F 搜索或粘贴事件链接。", "On an A/B field, press Ctrl+F to search or paste an event URL."), width, ""))
	}
	if m.setupError != "" {
		lines = append(lines, styles.warning.Render(trim(m.setupError, width)))
	}
	lines = append(lines, ansi.Wrap(l.text("[Tab] 下一项  [←/→] 切换  [Enter] 开始  [Esc] 语言",
		"[Tab] Next  [Left/Right] Change  [Enter] Start  [Esc] Language"), width, ""))
	return strings.Join(lines, "\n")
}

func money(value decimal.Decimal) string {
	whole, fractional, _ := strings.Cut(value.StringFixed(2), ".")
	for at := len(whole) - 3; at > 0 && whole[at-1] != '-'; at -= 3 {
		whole = whole[:at] + "," + whole[at:]
	}
	return whole + "." + fractional
}

func trim(value string, width int) string {
	safe := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	return ansi.Truncate(safe, max(0, width), "")
}

func pad(value string, width int) string {
	value = trim(value, width)
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

func right(value string, width int) string {
	return strings.Repeat(" ", max(0, width-ansi.StringWidth(value))) + value
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

type catalogMarket struct {
	Slug      string `json:"slug"`
	Label     string `json:"label"`
	Accepting bool   `json:"accepting"`
}

type catalogEvent struct {
	Title   string          `json:"title"`
	Markets []catalogMarket `json:"markets"`
}

type catalogResult struct {
	id     uint64
	events []catalogEvent
	err    error
}

type catalog struct {
	target   fieldID
	query    string
	pane     int
	event    int
	market   int
	events   []catalogEvent
	cancel   context.CancelFunc
	loading  bool
	searched bool
	err      string
}

func (m *model) openCatalog() {
	target := fieldA
	if m.session == nil {
		switch m.form.fields()[m.form.focus] {
		case fieldB, fieldOutcomeB:
			target = fieldB
		}
	}
	if m.catalog == nil {
		m.catalog = &catalog{}
	}
	m.catalog.target, m.catalog.pane, m.catalogActive = target, 0, true
}

func (m *model) closeCatalog() {
	if m.catalog.cancel != nil {
		m.catalog.cancel()
	}
	m.catalog.loading, m.catalog.cancel, m.catalogActive = false, nil, false
	m.catalogID++
}

func (m *model) searchCatalog() tea.Cmd {
	c := m.catalog
	query := strings.TrimSpace(c.query)
	if query == "" {
		return nil
	}
	if c.cancel != nil {
		c.cancel()
	}
	ctx, cancel := context.WithTimeout(m.ctx, 12*time.Second)
	c.cancel, c.loading, c.searched, c.err = cancel, true, true, ""
	c.events, c.event, c.market = nil, 0, 0
	m.catalogID++
	id, l := m.catalogID, m.form.language
	return func() tea.Msg {
		defer cancel()
		output, err := runCatalog(ctx, l, query)
		if err != nil {
			return catalogResult{id: id, err: err}
		}
		var events []catalogEvent
		err = json.Unmarshal(output, &events)
		return catalogResult{id: id, events: events, err: err}
	}
}

func runCatalog(ctx context.Context, l uiLanguage, args ...string) ([]byte, error) {
	python, err := pythonExecutable(l)
	if err != nil {
		return nil, err
	}
	script, err := filepath.Abs("../python_adapter/catalog.py")
	if err != nil {
		return nil, err
	}
	output, err := exec.CommandContext(ctx, python, append([]string{script}, args...)...).Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) != 0 {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(exit.Stderr)))
	}
	return output, err
}

func (m *model) catalogKey(msg tea.KeyPressMsg) tea.Cmd {
	c := m.catalog
	switch msg.String() {
	case "ctrl+c":
		m.closeCatalog()
		return tea.Quit
	case "esc":
		m.closeCatalog()
		if m.session == nil {
			m.clearPreview()
		}
	case "enter":
		switch c.pane {
		case 0:
			return m.searchCatalog()
		case 1:
			if len(c.events) != 0 {
				c.pane = 2
			}
		case 2:
			markets := c.markets()
			if len(markets) != 0 {
				market := markets[c.market]
				if m.session != nil {
					m.closeCatalog()
					return m.showPreview(market)
				}
				m.selectMarket(market.Slug)
			}
		}
	case "o", "O":
		if c.pane != 2 {
			c.edit(msg.Key().Text)
			return nil
		}
		markets := c.markets()
		if len(markets) != 0 {
			selected := markets[c.market]
			if m.preview != nil && m.preview.market.Slug == selected.Slug {
				return m.flipPreview()
			}
			return m.showPreview(selected)
		}
	case "tab", "right":
		if len(c.events) != 0 {
			c.pane = (c.pane + 1) % 3
		}
	case "shift+tab", "left":
		if len(c.events) != 0 {
			c.pane = (c.pane + 2) % 3
		}
	case "up":
		c.move(-1)
	case "down":
		if c.pane == 0 && len(c.events) != 0 {
			c.pane = 1
		} else {
			c.move(1)
		}
	case "ctrl+u":
		if c.pane == 0 {
			c.query = ""
		}
	case "backspace":
		if c.pane == 0 && c.query != "" {
			runes := []rune(c.query)
			c.query = string(runes[:len(runes)-1])
		}
	default:
		c.edit(msg.Key().Text)
	}
	return nil
}

func (c *catalog) edit(text string) {
	if c.pane != 0 {
		return
	}
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
	if len(c.query)+len(text) <= 600 {
		c.query += text
	}
}

func (c catalog) markets() []catalogMarket {
	if len(c.events) == 0 {
		return nil
	}
	return c.events[c.event].Markets
}

func (c *catalog) move(delta int) {
	switch c.pane {
	case 1:
		c.event = max(0, min(len(c.events)-1, c.event+delta))
		c.market = 0
	case 2:
		c.market = max(0, min(len(c.markets())-1, c.market+delta))
	}
}

func (m *model) selectMarket(slug string) {
	if !slugPattern.MatchString(slug) {
		return
	}
	target := m.catalog.target
	if target == fieldA {
		m.form.a = slug
	} else {
		m.form.b = slug
	}
	m.form.reviewed = false
	for i, id := range m.form.fields() {
		if id == target {
			m.form.focus = i
			break
		}
	}
	m.form.replace = true
	m.setupError = ""
	m.closeCatalog()
	m.clearPreview()
}

func (m *model) searchRows(width, height int) []string {
	l, c := m.form.language, m.catalog
	prompt := l.text("Ctrl+F 输入关键词或事件链接", "Ctrl+F to enter keywords or a URL")
	if c != nil && c.query != "" {
		prompt = c.query
	}
	focused := m.catalogActive && c.pane == 0
	query := drawPane(l.text("搜索盘口", "SEARCH"), []string{styles.title.Render("> ") + trim(prompt, width-4)}, width, 3, focused)
	var results []string
	if c != nil {
		switch {
		case c.loading:
			results = append(results, l.text("正在搜索…", "Searching…"))
		case c.err != "":
			results = append(results, wrapPlain(l.text("搜索失败  ", "Search failed  ")+c.err, width-2)...)
		case len(c.events) > 0:
			labels := make([]string, len(c.events))
			for i, event := range c.events {
				labels[i] = fmt.Sprintf("%d. %s", i+1, event.Title)
			}
			results = choiceRows(labels, c.event, height-5, width-2)
		case c.searched:
			results = append(results, l.text("没有匹配事件", "No matching events"))
		}
	}
	return append(query, drawPane(l.text("事件结果", "RESULTS"), results, width, height-3, m.catalogActive && c.pane == 1)...)
}

func (m *model) browseRows(width, limit int) []string {
	l, c := m.form.language, m.catalog
	var rows []string
	if m.session != nil {
		rows = append(rows, styles.title.Render(l.text("策略 A/B  [A/B] 选腿", "STRATEGY A/B  [A/B] View leg")))
		for i := range 2 {
			leg := m.leg(i)
			name := leg.Slug
			if name == "" {
				name = []string{m.form.a, m.form.b}[i]
			}
			side := l.yesNo(leg.Outcome == "Yes")
			if leg.Outcome == "" {
				side = "–"
			}
			line := fmt.Sprintf("  %c %s  %s", 'A'+i, side, name)
			if m.preview == nil && m.bookLeg == i {
				line = styles.title.Render("> " + trim(line[2:], width-2))
			}
			rows = append(rows, line)
		}
		if trade := m.latestTradeLine(width); trade != "" {
			rows = append(rows, trade)
		}
		if !m.catalogActive {
			rows = append(rows, l.text("[Ctrl+F] 搜索", "[Ctrl+F] Search"))
		}
	}
	if c == nil || len(c.events) == 0 {
		return rows
	}
	event := c.events[c.event]
	rows = append(rows, styles.title.Render(trim(event.Title, width)))
	labels := make([]string, len(event.Markets))
	for i, market := range event.Markets {
		labels[i] = fmt.Sprintf("%d. %s", i+1, market.Label)
		if !market.Accepting {
			labels[i] += l.text(" · 停止接单", " · closed")
		}
	}
	return append(rows, choiceRows(labels, c.market, limit-len(rows), width)...)
}

func choiceRows(values []string, selected, limit, width int) []string {
	if limit <= 0 || width <= 2 {
		return nil
	}
	first := max(0, selected-limit+1)
	rows := make([]string, 0, min(limit, len(values)))
	for i := first; i < min(len(values), first+limit); i++ {
		label := trim(values[i], width-2)
		if i == selected {
			rows = append(rows, styles.title.Render("> "+label))
		} else {
			rows = append(rows, "  "+label)
		}
	}
	return rows
}

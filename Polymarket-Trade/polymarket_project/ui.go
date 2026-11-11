package main

import (
	"context"
	"time"

	"Polymarket/src/service/taker"

	tea "charm.land/bubbletea/v2"
)

const refreshInterval = 100 * time.Millisecond

type uiTick time.Time
type exitTick struct{}

func nextUITick() tea.Cmd {
	return tea.Tick(refreshInterval, func(now time.Time) tea.Msg { return uiTick(now) })
}

type viewMessage taker.View
type stopped struct{}

type model struct {
	ctx            context.Context
	form           setup
	languageChosen bool
	setupError     string
	session        *session
	width, height  int
	logOffset      int
	bookOffset     int
	bookLeg        int
	bookFocus      bool
	catalog        *catalog
	catalogID      uint64
	catalogActive  bool
	preview        *bookPreview
	previewID      uint64
	value          taker.View
	pending        *taker.View
	frame          tea.View
	frameValid     bool
	pauseUntil     time.Time
	reconnectUntil time.Time
	reconnectFrom  uint64
	quitting       bool
}

func (m *model) Init() tea.Cmd { return nextUITick() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case viewMessage:
		value := taker.View(msg)
		m.pending = &value
		return m, m.listen()
	case uiTick:
		return m, m.refresh(time.Time(msg))
	case tea.MouseMsg:
		return m, nil
	}
	m.frameValid = false
	switch msg := msg.(type) {
	case stopped, exitTick:
		return m, tea.Quit
	case previewResult:
		if p := m.preview; p != nil && msg.id == m.previewID {
			p.loading, p.cancel, p.next = false, nil, time.Now().Add(5*time.Second)
			p.book = msg.book
			if msg.err != nil {
				p.err = m.form.language.diagnostic(msg.err.Error())
			}
		}
	case catalogResult:
		if m.catalogActive && msg.id == m.catalogID {
			c := m.catalog
			c.loading, c.cancel = false, nil
			if msg.err != nil {
				c.err = m.form.language.diagnostic(msg.err.Error())
			} else {
				c.events = msg.events
				if len(c.events) != 0 {
					c.pane = 1
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.PasteMsg:
		if m.catalogActive {
			m.catalog.edit(msg.Content)
			return m, nil
		}
		if m.session == nil && m.languageChosen {
			if value := m.form.value(); value != nil {
				m.form.edit(value, msg.Content)
				m.setupError = ""
			}
		}
	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		if m.catalogActive {
			if m.session != nil {
				key := msg.String()
				switch key {
				case "ctrl+p":
					return m, m.sessionKey("p")
				case "ctrl+r":
					return m, m.sessionKey("r")
				case "ctrl+q", "ctrl+c":
					return m, m.sessionKey("q")
				}
				if m.catalog.pane != 0 {
					switch key {
					case "p", "P", "r", "R", "q", "Q":
						return m, m.dashboardKey(msg)
					}
				}
			}
			return m, m.catalogKey(msg)
		}
		if !m.languageChosen {
			return m, m.languageKey(msg)
		}
		if m.session == nil {
			return m, m.setupKey(msg)
		}
		return m, m.dashboardKey(msg)
	}
	return m, nil
}

func (m *model) refresh(now time.Time) tea.Cmd {
	if m.pending != nil {
		value := *m.pending
		value.Logs = recentActions(value.Logs)
		if value.Paused != m.value.Paused {
			m.pauseUntil = time.Time{}
		}
		if value.MCP.Ready && value.MCP.Session > m.reconnectFrom {
			m.reconnectUntil = time.Time{}
		}
		m.value, m.pending, m.frameValid = value, nil, false
	}
	if !m.pauseUntil.IsZero() && !now.Before(m.pauseUntil) {
		m.pauseUntil, m.frameValid = time.Time{}, false
	}
	if !m.reconnectUntil.IsZero() && !now.Before(m.reconnectUntil) {
		m.reconnectUntil, m.frameValid = time.Time{}, false
	}
	if p := m.preview; p != nil && !p.loading && !now.Before(p.next) {
		m.frameValid = false
		return tea.Batch(nextUITick(), m.refreshPreview())
	}
	return nextUITick()
}

func (m *model) languageKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		return tea.Quit
	case "tab", "up", "down", "left", "right", "space":
		if m.form.language == chinese {
			m.form.language = english
		} else {
			m.form.language = chinese
		}
	case "enter":
		m.languageChosen = true
	}
	return nil
}

func (m *model) setupKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		m.languageChosen = false
		m.setupError = ""
		return nil
	case "ctrl+f":
		if m.form.mode == liveObserve || m.form.mode == livePaper {
			m.openCatalog()
		}
		return nil
	case "enter":
		params, err := m.form.params()
		if err != nil {
			m.setupError = err.Error()
			return nil
		}
		active, err := startSession(m.ctx, params, m.form.language)
		if err != nil {
			m.setupError = err.Error()
			return nil
		}
		m.session = active
		m.value = taker.View{Phase: "STARTING", Ledger: taker.Ledger{Cash: params.config.Balance}}
		return m.listen()
	default:
		m.form.key(msg)
		m.setupError = ""
		return nil
	}
}

func (m *model) dashboardKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m.sessionKey("q")
	case "p", "P":
		return m.sessionKey("p")
	case "r", "R":
		return m.sessionKey("r")
	case "ctrl+f":
		m.openCatalog()
	case "o", "O":
		return m.flipPreview()
	case "enter":
		return m.refreshPreview()
	case "a", "A":
		m.clearPreview()
		m.bookLeg, m.bookOffset, m.bookFocus = 0, 0, true
	case "b", "B":
		m.clearPreview()
		m.bookLeg, m.bookOffset, m.bookFocus = 1, 0, true
	case "tab":
		m.bookFocus = !m.bookFocus
	case "up":
		m.scrollPane(-1)
	case "down":
		m.scrollPane(1)
	case "end":
		m.logOffset, m.bookOffset = 0, 0
	}
	return nil
}

func (m *model) listen() tea.Cmd {
	s := m.session
	return func() tea.Msg {
		select {
		case <-s.ctx.Done():
			return stopped{}
		case v := <-s.views:
			return viewMessage(v)
		}
	}
}

func (m *model) sessionKey(key string) tea.Cmd {
	switch key {
	case "p":
		if m.pauseUntil.IsZero() && m.control(taker.TogglePause) {
			m.pauseUntil = time.Now().Add(time.Second)
		}
	case "r":
		if m.control(taker.Reconnect) {
			m.reconnectFrom = m.value.MCP.Session
			if m.pending != nil {
				m.reconnectFrom = max(m.reconnectFrom, m.pending.MCP.Session)
			}
			m.reconnectUntil = time.Now().Add(20 * time.Second)
		}
	case "q":
		m.quitting = true
		return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return exitTick{} })
	}
	return nil
}

func (m *model) control(command taker.Command) bool {
	select {
	case m.session.controls <- command:
		return true
	default:
		return false
	}
}

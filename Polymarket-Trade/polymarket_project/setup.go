package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"Polymarket/src/service/taker"

	tea "charm.land/bubbletea/v2"
	"github.com/shopspring/decimal"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,249}$`)

type runMode int

const (
	demo runMode = iota
	demoTimeout
	liveObserve
	livePaper
)

var modes = [...]struct {
	zh, en         string
	zhHelp, enHelp string
}{
	{"演示成交", "Demo fills",
		"用固定样例盘口练习双腿成交，适合第一次使用。",
		"Practice filling both legs on fixed sample books. Start here if you're new."},
	{"演示单腿超时", "Demo one-leg timeout",
		"用固定样例让 B 腿故意超时，查看系统如何平掉 A 腿。",
		"Deliberately time out leg B on sample books and watch leg A unwind."},
	{"真实盘口观察", "Watch live books",
		"输入两个市场，查看 Polymarket 实时盘口和时间差，只看行情。",
		"Enter two markets to watch live Polymarket books and their time difference. Viewing only."},
	{"真实盘口模拟", "Simulate on live books",
		"用真实盘口模拟成交，使用虚拟资金。先核对结算规则，再填每对最低兑付。",
		"Simulate fills against live books with virtual funds. Review the rules and enter the minimum payout per pair."},
}

type fieldID int

const (
	fieldMode fieldID = iota
	fieldA
	fieldOutcomeA
	fieldB
	fieldOutcomeB
	fieldBalance
	fieldShares
	fieldPayout
	fieldReviewed
)

type setup struct {
	language uiLanguage
	mode     runMode
	a, b     string
	yesA     bool
	yesB     bool
	balance  string
	shares   string
	payout   string
	reviewed bool
	focus    int
	replace  bool
}

func newSetup() setup { return setup{balance: "10000", shares: "5"} }

func (s setup) fields() []fieldID {
	if s.mode == demo || s.mode == demoTimeout {
		return []fieldID{fieldMode, fieldBalance, fieldShares}
	}
	fields := []fieldID{fieldMode, fieldA, fieldOutcomeA, fieldB, fieldOutcomeB, fieldBalance, fieldShares}
	if s.mode == livePaper {
		fields = append(fields, fieldPayout, fieldReviewed)
	}
	return fields
}

func (s *setup) move(delta int) {
	count := len(s.fields())
	s.focus = (s.focus + delta + count) % count
	s.replace = true
}

func (s *setup) value() *string {
	switch s.fields()[s.focus] {
	case fieldA:
		return &s.a
	case fieldB:
		return &s.b
	case fieldBalance:
		return &s.balance
	case fieldShares:
		return &s.shares
	case fieldPayout:
		return &s.payout
	default:
		return nil
	}
}

func (s *setup) changeChoice(delta int) {
	switch s.fields()[s.focus] {
	case fieldMode:
		next := runMode((int(s.mode) + delta + len(modes)) % len(modes))
		if next != s.mode {
			s.reviewed = false
		}
		s.mode = next
	case fieldOutcomeA:
		s.yesA = !s.yesA
		s.reviewed = false
	case fieldOutcomeB:
		s.yesB = !s.yesB
		s.reviewed = false
	case fieldReviewed:
		s.reviewed = !s.reviewed
	}
}

func (s *setup) edit(value *string, input string) {
	input = strings.TrimSpace(input)
	input = strings.Map(func(r rune) rune {
		if r < 33 || r > 126 {
			return -1
		}
		return r
	}, input)
	if input == "" {
		return
	}
	if s.replace {
		*value = ""
	}
	s.replace = false
	if len(*value)+len(input) <= 250 {
		*value += input
		s.resetReviewForMarketEdit()
	}
}

func (s *setup) resetReviewForMarketEdit() {
	switch s.fields()[s.focus] {
	case fieldA, fieldB, fieldPayout:
		s.reviewed = false
	}
}

func (s *setup) key(key tea.KeyPressMsg) {
	switch key.String() {
	case "tab", "down":
		s.move(1)
	case "shift+tab", "up":
		s.move(-1)
	default:
		value := s.value()
		if value == nil {
			switch key.String() {
			case "left":
				s.changeChoice(-1)
			case "right", "space":
				s.changeChoice(1)
			}
			return
		}
		switch key.String() {
		case "ctrl+u":
			*value = ""
			s.replace = false
			s.resetReviewForMarketEdit()
		case "backspace":
			if s.replace {
				*value = ""
			} else if len(*value) > 0 {
				*value = (*value)[:len(*value)-1]
			}
			s.replace = false
			s.resetReviewForMarketEdit()
		default:
			s.edit(value, key.Key().Text)
		}
	}
}

type sessionParams struct {
	args   []string
	config taker.Config
}

func (s setup) params() (sessionParams, error) {
	balance, err := s.positive(s.language.text("虚拟本金", "Virtual balance"), s.balance)
	if err != nil {
		return sessionParams{}, err
	}
	shares, err := s.positive(s.language.text("每腿份额", "Shares per leg"), s.shares)
	if err != nil {
		return sessionParams{}, err
	}

	cfg := taker.Config{Balance: balance, Shares: shares, MaxAge: 2 * time.Second, HedgeTimeout: 300 * time.Millisecond}
	a, b, outcomeA, outcomeB := s.a, s.b, outcome(s.yesA), outcome(s.yesB)
	var flags []string
	switch s.mode {
	case demo, demoTimeout:
		a, b, outcomeA, outcomeB = "fixture-a", "fixture-b", "No", "No"
		cfg.Reviewed, cfg.PayoutFloor = true, decimal.NewFromInt(1)
		flags = append(flags, "--fixture")
		if s.mode == demoTimeout {
			flags = append(flags, "--miss-b")
		}
	case liveObserve, livePaper:
		if !slugPattern.MatchString(a) || !slugPattern.MatchString(b) || a == b {
			return sessionParams{}, fmt.Errorf("%s", s.language.text("请填写两个不同的市场标识（链接最后一段）", "Enter two different market slugs (the last part of each link)."))
		}
		cfg.ObserveOnly = s.mode == liveObserve
		if s.mode == livePaper {
			floor, err := s.positive(s.language.text("每对最低兑付", "Minimum payout per pair"), s.payout)
			if err != nil || floor.GreaterThan(decimal.NewFromInt(2)) {
				return sessionParams{}, fmt.Errorf("%s", s.language.text("每对最低兑付必须大于 0 且不超过 $2", "Minimum payout per pair must be greater than $0 and at most $2."))
			}
			if !s.reviewed {
				return sessionParams{}, fmt.Errorf("%s", s.language.text("请先核对结算规则，再把「已核对规则」切换为「是」", "Review the settlement rules, then set Rules reviewed to Yes."))
			}
			cfg.Reviewed, cfg.PayoutFloor = true, floor
		}
	}
	args := []string{"--a", a, "--b", b, "--outcome-a", outcomeA, "--outcome-b", outcomeB}
	return sessionParams{args: append(args, flags...), config: cfg}, nil
}

func (s setup) positive(name, value string) (decimal.Decimal, error) {
	if len(value) > 20 || strings.ContainsAny(value, "eE+-") {
		return decimal.Zero, fmt.Errorf(s.language.text("%s 格式无效", "%s has an invalid format."), name)
	}
	result, err := decimal.NewFromString(value)
	if err != nil || !result.IsPositive() || !result.Equal(result.Truncate(8)) {
		return decimal.Zero, fmt.Errorf(s.language.text("%s 必须为正数，最多八位小数", "%s must be positive with at most eight decimal places."), name)
	}
	return result, nil
}

func outcome(yes bool) string {
	if yes {
		return "Yes"
	}
	return "No"
}

func (s setup) display(id fieldID) (label, value string) {
	l := s.language
	switch id {
	case fieldMode:
		mode := modes[s.mode]
		return l.text("模式", "Mode"), "< " + l.text(mode.zh, mode.en) + " >"
	case fieldA:
		return l.text("市场 A 标识", "Market A slug"), s.a
	case fieldOutcomeA:
		return l.text("A 腿方向", "Outcome A"), "< " + l.yesNo(s.yesA) + " >"
	case fieldB:
		return l.text("市场 B 标识", "Market B slug"), s.b
	case fieldOutcomeB:
		return l.text("B 腿方向", "Outcome B"), "< " + l.yesNo(s.yesB) + " >"
	case fieldBalance:
		return l.text("虚拟本金（美元）", "Virtual balance"), s.balance
	case fieldShares:
		return l.text("每腿份额", "Shares per leg"), s.shares
	case fieldPayout:
		return l.text("每对最低兑付", "Minimum payout"), s.payout
	case fieldReviewed:
		return l.text("已核对规则", "Rules reviewed"), "< " + l.yesNo(s.reviewed) + " >"
	}
	return "", ""
}

func (l uiLanguage) yesNo(yes bool) string {
	if yes {
		return l.text("是", "Yes")
	}
	return l.text("否", "No")
}

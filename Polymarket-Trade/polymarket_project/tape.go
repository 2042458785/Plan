package main

import (
	"fmt"
	"time"
)

func (m *model) latestTradeLine(width int) string {
	l := m.form.language
	if m.form.mode == demo || m.form.mode == demoTimeout {
		return ""
	}

	for _, trade := range m.value.Snapshot.Trades {
		for i, leg := range m.value.Snapshot.Legs {
			if trade.Token != leg.Token {
				continue
			}
			action, ink := l.text("买", "BUY"), styles.buy
			if trade.Side == "SELL" {
				action, ink = l.text("卖", "SELL"), styles.sell
			}
			size := "—"
			if trade.Size != nil {
				size = trade.Size.StringFixed(2)
			}
			label := fmt.Sprintf("%s %s %c %s × %s", l.text("公开成交", "Public fill"),
				time.UnixMilli(trade.ExchangeMS).Format("15:04:05"), 'A'+i, trade.Price.String(), size)
			return ink.Render(trim(action+" "+label, width))
		}
	}
	return ""
}

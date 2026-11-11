package main

import "strings"

type uiLanguage uint8

const (
	chinese uiLanguage = iota
	english
)

func (l uiLanguage) text(zh, en string) string {
	if l == english {
		return en
	}
	return zh
}

var dashboardChinese = strings.NewReplacer(
	"STARTING", "正在启动",
	"IDLE", "等待信号",
	"A PENDING", "等待 A 腿成交",
	"B PENDING", "等待 B 腿成交",
	"HEDGING", "正在平掉单腿",
	"HEDGE FAILED", "单腿平仓失败",
	"waiting for new session", "等待连接",
	"waiting for both books", "等待双腿盘口",
	"snapshot unavailable", "暂时读不到盘口",
	"market feed or trading unavailable", "行情断开或市场已停止交易",
	"exchange timestamps differ by 200ms or more", "两腿盘口时间相差至少 200 毫秒",
	"[LOCAL_TS] missing exchange timestamp", "[LOCAL_TS] 缺交易所时间戳",
	"stale order book", "盘口已过期",
	"observe only: verify settlement rules before simulating", "只观察盘口",
	"new signals paused", "新信号已暂停",
	"no positive net payout floor", "计入手续费后，最低兑付未超过成本",
	"net floor $", "估算最低净收益 $",
	"virtual balance insufficient", "虚拟余额不足",
	"invalid or unknown tick, minimum, or fee rate", "价格步长、最小份额或费率无效",
	"invalid market parameters or quantity below minimum", "市场参数无效或份额低于下限",
	"invalid, duplicate, or off-tick level", "盘口档位重复或价格精度无效",
	"market buy below $1", "买入金额低于 $1",
	"insufficient depth", "盘口可成交份额不足",
	"missing current hedge book", "缺少最新平仓盘口",
	"Market rules changed; review again before restarting", "市场规则已变更，重新核对后再启动",
	"Market rules changed; new signals paused", "市场规则已变更，新信号已暂停",
	"Risk paused; reconnect to retry outstanding hedge", "风控已暂停，重连后可重试平仓",
	"New signals paused=true", "已暂停新信号",
	"New signals paused=false", "已恢复新信号",
	"MCP reconnect requested", "已请求重连",
	"MCP: DEAD", "模拟连接断开",
	"WS: DOWN", "行情断开",
	"Snapshot unavailable:", "暂时读不到盘口",
	"[LOCAL_TS] exchange timestamp missing; strategy skipped", "[LOCAL_TS] 缺交易所时间戳，已跳过本轮",
	"[SETTLED] pair realized", "[结算] 已实现盈亏",
	"[SIGNAL] net floor=", "[信号] 最低净收益=",
	"[A FILL] shares=", "[A腿成交] 份额=",
	"[B FILL] shares=", "[B腿成交] 份额=",
	"[A REJECTED]", "[A腿未成交]",
	"[B REJECTED]", "[B腿未成交]",
	"[B UNCONFIRMED]", "[B腿未确认]",
	"[HEDGE_LOSS] realized=", "[HEDGE_LOSS] 平仓盈亏=",
	"[HEDGE_FAILED] unpaired=", "[HEDGE_FAILED] 平仓失败，未配对份额=",
	"unpaired=", "未配对=",
	"cost=", "成本=",
	"reason=", "原因=",
	"context deadline exceeded", "请求超时",
	"leg 1: ", "A 腿 ",
	"leg 2: ", "B 腿 ",
)

func (l uiLanguage) diagnostic(message string) string {
	if l == english {
		return message
	}
	return dashboardChinese.Replace(message)
}

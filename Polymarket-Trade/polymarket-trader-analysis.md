# Polymarket 两位交易员分析

2026 年 11 月 11 日，北京时间。依据两人的 X 原帖、回复、持仓截图，以及 Polymarket 官方钱包流水和市场规则。

**两人确实进入全类别预测市场月榜前五。** e46m3 有自动对冲平仓，也公开展示过方向仓。balthazar 有跨盘口互补成交和多结果仓位转换。现有证据能复核这些操作，仍无法确认哪种策略贡献了最多利润。

## 榜单与钱包盈亏

| 交易员 | 官方月榜快照 | 官方钱包已实现市场盈亏变化 |
| --- | ---: | ---: |
| [@love_u_4ever（e46m3）](https://polymarket.com/@e46m3) | 第 2，+$3,746,454 | [−$194,785](https://data-api.polymarket.com/v2/user-pnl?user=0x4f1d5ae26fc31472966e951af3183308736d8de2&interval=1m&fidelity=1d) |
| [@balthazarpoly（balthazar）](https://polymarket.com/@balthazar) | 第 4，+$2,122,527 | [+$14,469](https://data-api.polymarket.com/v2/user-pnl?user=0x5a218c7ad04135830a45c41aaed7294df7809318&interval=1m&fidelity=1d) |

钱包列取 10 月 13 日至 11 月 11 日官方曲线首尾的 `realized_market_pnl` 差值。**置信度高**的是两个接口各自显示的数值。两套数字的差额尚未对账，实际到手净利未知。 [官方月榜](https://polymarket.com/leaderboard/overall/monthly/profit) · [钱包数据说明](https://docs.polymarket.com/trading/wallet-activity#wallet-pnl-history)

## e46m3 · 买反面平仓，也持有方向仓

- **已核实自动平仓的做法。** 他在 9 月 20 日说，退出时由程序买入反面。11 月 11 日，官方流水里同一个「BTC 高于 $72,000」合约出现三次买入 Yes 后合并同等份数的记录，间隔 **1–4 秒**。其中一次以 **90.9¢** 买入 **5.28 份 Yes**，与已有 No 合并收回 **$5.28**。原有 No 的成本未知，这一对仓位的盈亏无法确定。 [平仓原帖](https://x.com/love_u_4ever/status/2101625974244909104) · [官方成交与合并流水](https://data-api.polymarket.com/v2/activity?user=0x4f1d5ae26fc31472966e951af3183308736d8de2&limit=20&condition=0xa50b960bdd6fcae61f53a5144857e410b4bbeb753245fee7900b5f520c570a9c)
- **公开晒过 $15,417 的方向仓。** 11 月 3 日晒出的三张图是「共和党控制参议院」No（$8,725）、「民主党拿下两院」Yes（$3,291）、「油价至少 $80 且民主党拿下参议院」Yes（$3,401）。三笔都押向相近的选举结果，截图无法说明其他仓位有没有对冲。 [中期选举原帖及三张持仓图](https://x.com/love_u_4ever/status/2117297239072862479)
- **公开讲过两种思路。** 99.9¢ 接尾盘卖单的案例来自他举例的 @newtradeway。用 BTC 期权配合 Polymarket 的案例，他随后承认两边结算时间不同，也要算期权费用。两条都缺少证据归入 e46m3 的主要利润来源。 [尾盘挂单](https://x.com/love_u_4ever/status/1981379332363997192) · [案例账号](https://x.com/love_u_4ever/status/1981379334331125986) · [期权案例补充](https://x.com/love_u_4ever/status/1937903683666415774)

## balthazar · 跨盘口配对与多结果转换

- **已核实一组跨盘口成交。** 11 月 11 日 11:50:10，他在《饥饿游戏》首周末票房的两个盘口各买 No。「低于 $1.05 亿」买了 **208 份**，「至少 $1.05 亿」买了 **198.04 份**。两个盘口的数据源、截止和兜底结算规则一致。可配对的 **198.04 对**按流水分摊成本约 **$190.56**，若持有到按规则结算，兑付约 **$198.04**，差额约 **$7.48**。同秒成交还多出 **9.96 份**未配对，整个钱包后来是否平仓、是否兑现这笔差额，尚未核实。 [同秒四笔官方流水](https://data-api.polymarket.com/v2/activity?user=0x5a218c7ad04135830a45c41aaed7294df7809318&start=1794369009&end=1794369011&limit=10) · [低于门槛的规则](https://gamma-api.polymarket.com/markets/slug/will-the-hunger-games-sunrise-on-the-reaping-opening-weekend-box-office-be-less-than-105m) · [至少达到门槛的规则](https://gamma-api.polymarket.com/markets/slug/will-the-hunger-games-sunrise-on-the-reaping-opening-weekend-box-office-be-at-least-105m)
- **已核实他用过多结果转换。** 有人问他怎么成交，他回了「convert」。官方流水也记录了他在英雄联盟世界赛击杀数、马斯克发帖数等市场做过转换。单凭转换记录算不出套利盈亏，还要找到转换前的成本和之后的卖出或兑付。 [本人回复](https://x.com/balthazarpoly/status/2064212743951892546) · [官方转换流水](https://data-api.polymarket.com/v2/activity?user=0x5a218c7ad04135830a45c41aaed7294df7809318&type=CONVERSION&start=1794368559&end=1794368561&limit=10)
- **原帖显示他关注组合报价。** 他谈过组合模块的报价透明度，晒过地缘政治组合的报价示例，也说自己有其他账户。报价图无法证明成交，单个榜单钱包也无法代表他的全部交易。 [组合模块原帖](https://x.com/balthazarpoly/status/2065205016336507071) · [组合报价图](https://x.com/balthazarpoly/status/2102331221443326451/photo/1) · [其他账户的回复](https://x.com/balthazarpoly/status/2113021304299208872)

## 套利研究顺序

| 顺序 | 做法 | 怎么验 |
| --- | --- | --- |
| 1 | **同平台跨盘口互补** | 先复现票房这种同事件、同门槛的配对。核对结算条件，按两边能成交的相同份数计算，扣费后成本须低于最低兑付额。 |
| 2 | **NegRisk 多结果转换** | 买一个结果的 No，转换成其他结果的 Yes。按其他结果的实际买盘和深度算卖出净额，确认能覆盖成本。 [官方转换规则](https://docs.polymarket.com/concepts/negative-risk) |
| 3 | **同盘口 Yes + No 合并** | 按两边的卖盘深度，算等量买入的全成本。每对成本低于 $1 才有差价，合并也可以用于退出旧仓位。 [合并规则](https://docs.polymarket.com/concepts/positions-tokens) · [手续费](https://docs.polymarket.com/trading/fees) |

先用 **7 天只读模拟**测试第一种。记录每天有几次扣费后仍为正的机会、同价能做多大、价差能维持多久、资金锁多久，以及只成交一边时退出要亏多少。第一种排在前面是因为有可复核的真实成交，当前最赚钱的是哪种仍缺连续盘口数据。

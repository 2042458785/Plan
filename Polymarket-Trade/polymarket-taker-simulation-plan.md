# Polymarket 双腿 Taker 模拟验证方案

Go 项目以 TUI 为入口。在 `polymarket_project` 目录运行 `go run .`，启动参数在界面填写。下方记录接口、风控和验收结果。 [README](README.md)

## 目录与核心依赖

```text
Polymarket-Trade/
├── polymarket_project/
│   ├── go.mod                         # 在现有模块内新增依赖
│   ├── main.go                        # 启动 TUI 和 MCP 会话
│   ├── setup.go                       # TUI 参数表单
│   ├── dashboard.go                   # Bubble Tea 界面与键盘事件
│   ├── terminal.go                    # 市场、配对、盘口和账户分区
│   ├── catalog.go                     # 事件搜索与盘口选择
│   └── src/
│       ├── pkg/papermcp/provider.go     # 第二步，MCP、超时、子进程守护
│       └── service/taker/             # engine.go 算账，runner.go 执行与风控
└── python_adapter/
    ├── bin/pm-trader-mcp               # 项目内固定路径的适配入口
    ├── pyproject.toml                  # Python 版本与依赖
    ├── requirements.lock              # Python 依赖锁定
    ├── catalog.py                     # 读取公开事件和盘口
    └── adapter.py                     # 订阅、缓存与指定快照模拟
```

现有 `module Polymarket` 及 Go 1.26.2 延用。以下是 TUI 路径涉及的核心依赖，`decimal` 已在原模块中。

```go
require (
    charm.land/bubbletea/v2 v2.1.1
    github.com/modelcontextprotocol/go-sdk v1.8.0
    github.com/shopspring/decimal v1.4.0
)
```

Go 依赖最低版本分别是 Go 1.26.0、1.25.0 和 1.10。采用官方 MCP Go SDK 管理 stdio 与 JSON-RPC，不另写一套协议。 [Bubble Tea](https://github.com/charmbracelet/bubbletea/blob/v2.1.1/go.mod) · [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/go.mod) · [decimal](https://github.com/shopspring/decimal/blob/v1.4.0/go.mod)

Python 适配层固定 `polymarket-paper-trader==0.4.13`。项目入口复用上游的市场和 token 解析能力，扩展 WebSocket 缓存与指定快照撮合。默认 `python3` 实测为 3.9.6。本机已使用项目内的 Python 3.12.16 虚拟环境完成真实握手。 [上游依赖声明](https://github.com/agent-next/polymarket-paper-trader/blob/v0.4.13/pyproject.toml)

## Go 和 Python 的契约

| 工具 | 输入与输出 |
| --- | --- |
| `get_pair_snapshot` | 一次读取同一缓存视图中的两腿。返回连接代次、快照 ID、两腿 token、各自交易所时间戳、盘口接收年龄、完整买卖深度、最低份额、价格步长、费率和行情状态。 |
| `simulate_fill` | 输入请求 ID、连接代次、快照 ID、token、买卖方向、份额和价格上限或下限。返回成交、部分成交或未成交，以及逐档价量和费用。固定快照已失效时返回错误。 |

价格、份额、费率和金额均用十进制字符串传输。两个工具均通过本地 stdio 调用。Python 只对指定盘口做纯模拟，Go 用 `decimal` 复核金额并记录状态。现成 `buy` 按美元金额重新取价，且用浮点数表示金额，所以需要新增适配工具。模拟成交响应本身不改变余额，Go 只对通过状态校验的响应记账一次。 [上游盘口读取](https://github.com/agent-next/polymarket-paper-trader/blob/v0.4.13/pm_trader/api.py#L319) · [上游撮合](https://github.com/agent-next/polymarket-paper-trader/blob/v0.4.13/pm_trader/engine.py#L269)

## 数据、报价和风控

1. **同步与新鲜度。** Python 持续订阅两腿，先收到完整盘口再应用增量，断线后重新建立。Go 每 100ms 读取缓存，同类轮询只允许一个请求在途。只用两腿的交易所时间戳比较 `|A-B| < 200ms`。缺时间戳的行情标 `[LOCAL_TS]`，可显示，跳过算账。另查连接状态、交易所事件年龄及缓存接收年龄。同步判据只比较交易所时间戳，年龄校验单独使用时钟。新鲜度门槛默认 2 秒，独立于 200ms 同步阈值。观察模式显示同步状态和行情校验结果。 [Polymarket 实时行情](https://docs.polymarket.com/market-data/realtime-data)
2. **可成交净收益。** 已核对两市场结算条款和例外的组合才进入报价。两腿同份额，逐档吃卖盘，检查深度、余额、动态最低份额和 `price % tickSize == 0`。VWAP 用总名义额除以成交份额。手续费按各档份额及成交价计算 `C × feeRate × p × (1-p)`，按官方精度保留五位小数。净收益下限等于配对最低兑付减去两腿成本及手续费。费率未知或市场规格无效则拒绝。遇到 Maker 意图直接记 `[ERROR] Maker blocked: paper-trader does not simulate FIFO queue`。 [市场参数](https://docs.polymarket.com/api-reference/markets/get-clob-market-info) · [手续费](https://docs.polymarket.com/trading/fees)
3. **500ms 单边风控。** Go 确认 A 腿模拟成交时，用单调时钟起算。`time.AfterFunc` 只向状态循环投递超时事件。截止前用 B 腿确认的份额冲抵。到期时只处理未配对份额，按届时的新鲜买盘从最佳买价逐档平仓。对冲行情请求有独立超时。缺盘口、请求超时或深度不足时记 `[HEDGE_FAILED]`，保留未平敞口并停止新交易计算。完成平仓记 `[HEDGE_LOSS]` 和实际模拟损益。超时、成交和记账由同一状态循环处理，迟到或重复回执不得再次扣款、补仓或平仓。
4. **守护与显示。** Go 使用 `os/exec` 启动项目内的适配入口。SDK 管理独立的读写路径和请求关联，应用通过有界 channel 收结果，设请求超时，持续排空 stderr。断线后显示 `MCP: DEAD`，作废旧快照与待处理回执，保留已确认的成交和敞口，按 1、2、4、8、16 秒退避。新连接先握手，再验证两份新盘口。TUI 用深色直角面板与红绿累计深度条。顶部显示连接、余额、已实现盈亏、两腿时间差与行情往返延迟。中部显示市场、配对检查及可滚动的 A/B 订单簿，底部显示模拟成交、风控日志和账户。P 暂停新信号，Q 退出，R 重连。暂停期间继续处理已有敞口。常规行情不刷日志，价差信号每 10 秒最多打印一次，风控事件完整记录。 [MCP stdio 规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)

## 分步验收

| 步骤 | 交付及通过条件 |
| --- | --- |
| 1 | 本方案、目录结构、版本清单及依赖检查。 |
| 2 | `provider.go` 和 Python 行情适配。验证真实 stdio 握手、完整盘口和增量、时间戳透传、并发读写超时、子进程退出与重连、旧代次拒绝。 |
| 3 | `engine.go` 和 Python 的快照模拟工具。验证跨档滑点、手续费、最低份额、tick、余额不足及 Maker 拒绝。覆盖时间差 199ms 通过、200ms 拒绝，以及缺失和过期盘口。用可控时钟回放全成、部分成交、超时、迟到、重复回执和对冲失败。 |
| 4 | Bubble Tea `main.go`、配置表单和盘口面板。验证无参数启动、关键词及链接搜索、盘口填入、A/B 深度切换、日志滚动及 P/R/Q。 |

## 验收与置信度

- **已核实**。Go 单测、静态检查、竞态检测、Python 单测和实际 stdio 联调通过。真实终端验证了事件链接搜索、盘口填入、A/B 深度切换、事件日志和模拟成交。真实子进程被终止后成功重连，旧会话回执被拒绝。慢速模拟请求超时期间，快照请求仍可返回。离线回放覆盖 199/200ms 边界、陈旧行情、跨档费用、最小份额、价格步长、重复与迟到回执以及买盘消失后的风控暂停。
- **固定样例**。$5 账本每腿各买 5 份，双腿总成本 $4.92480，配对净兑付下限高于成本 $0.07520。双腿完成时这部分仍待结算。B 腿故意迟到的回放在 500ms 后对冲，已实现亏损 $0.27428，剩余现金 $4.72572。
- **公开行情**。读到此前分析的《饥饿游戏》$1.05 亿票房双 No 组合的两腿 WebSocket 盘口。首轮 25 秒观察中收到 241 次快照响应，0 次通过全部行情门槛。复测 25 秒收到 236 次，25 次通过全部行情门槛，187 次因两腿相差至少 200ms 被拒。复测本地 MCP 往返均值约 6.91ms，最高约 38.14ms。该延迟是本机 Go 到 Python 的请求时间。两轮均为观察模式。
- **仍未知**。这段公开行情不足以判断两腿通常能否同步、策略有无稳定净收益或 $50 实盘表现。适配器按可见深度估算，成交以模拟回执为准。终端账本在内存中，并限制同一组合同时最多一组已配对持仓。

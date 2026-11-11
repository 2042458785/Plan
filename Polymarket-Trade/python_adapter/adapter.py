"""Public Polymarket market-data cache and stateless paper fills over MCP stdio."""
from __future__ import annotations

import argparse
import asyncio
from collections import OrderedDict, deque
from dataclasses import dataclass, field
from decimal import Decimal, ROUND_HALF_UP, localcontext
import json
import hashlib
import logging
import time
from urllib.parse import quote
from uuid import uuid4

import httpx
from mcp.server import MCPServer
from pm_trader.api import GAMMA_BASE, _parse_market
from websockets.asyncio.client import connect

WS = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
LOG = logging.getLogger("taker.market")
ONE = Decimal(1)
ZERO = Decimal(0)
FEE_PRECISION = Decimal("0.00001")
MAKER_BLOCKED = "[ERROR] Maker blocked: paper-trader does not simulate FIFO queue"


def decimal(value: object) -> Decimal:
    if not isinstance(value, (str, int, Decimal)):
        raise ValueError("invalid decimal")
    result = Decimal(value)
    if not result.is_finite():
        raise ValueError("non-finite decimal")
    return result


def text(value: Decimal) -> str:
    return format(value, "f")


def millis(value: object) -> int | None:
    if isinstance(value, (str, int)) and str(value).isascii() and str(value).isdigit():
        stamp = int(value)
        if 0 < stamp < 10**14:
            return stamp
    return None


@dataclass(frozen=True)
class Market:
    slug: str
    outcome: str
    token: str
    condition: str
    tick: Decimal
    minimum: Decimal
    rate: Decimal | None
    accepting: bool
    payout: Decimal | None = None
    rules: str = "fixture-v1"


def parse_market(raw: dict, slug: str, outcome: str) -> Market:
    market = _parse_market({key: raw.get(key) for key in ("slug", "conditionId", "outcomes", "clobTokenIds")})
    tokens = [entry["token_id"] for entry in market.tokens]
    outcomes = market.outcomes
    if raw.get("slug") != slug or set(outcomes) != {"Yes", "No"} or len(tokens) != 2 or len(outcomes) != 2:
        raise ValueError("expected a binary market with distinct Yes and No tokens")
    index = outcomes.index(outcome)
    # The pinned parser resolves market identity. Numeric execution parameters
    # are taken from the raw JSON decoded with Decimal.
    condition = market.condition_id
    if not condition or not tokens[index] or tokens[0] == tokens[1]:
        raise ValueError("invalid market identity")
    rate = None
    schedule = raw.get("feeSchedule")
    if raw.get("feesEnabled") is False:
        rate = ZERO
    elif raw.get("feesEnabled") is True and isinstance(schedule, dict) and schedule.get("exponent") == 1:
        rate = decimal(schedule["rate"])
        if not ZERO < rate < ONE:
            rate = None
    closed = raw.get("closed") is True
    payout = None
    if closed and str(raw.get("umaResolutionStatus", "")).lower() == "resolved":
        prices = [decimal(value) for value in json.loads(raw["outcomePrices"])]
        if len(prices) == 2 and sum(prices) == ONE and prices[0] in (ZERO, ONE, Decimal("0.5")):
            payout = prices[index]
    rules = hashlib.sha256(json.dumps([
        str(raw.get(key, "")) for key in ("question", "description", "resolutionSource", "endDate")
    ], ensure_ascii=False).encode()).hexdigest()
    return Market(slug, outcome, str(tokens[index]), condition,
                  decimal(raw["orderPriceMinTickSize"]), decimal(raw["orderMinSize"]),
                  rate, raw.get("acceptingOrders") is True and not closed, payout, rules)


async def markets(client: httpx.AsyncClient, selections: list[tuple[str, str]]) -> list[Market]:
    async def get(slug: str, outcome: str) -> Market:
        response = await client.get(f"{GAMMA_BASE}/markets/slug/{quote(slug, safe='')}")
        response.raise_for_status()
        return parse_market(json.loads(response.content, parse_float=Decimal), slug, outcome)

    result = await asyncio.gather(*(get(slug, outcome) for slug, outcome in selections))
    if result[0].token == result[1].token or result[0].condition == result[1].condition:
        raise ValueError("expected two different markets")
    return list(result)


@dataclass
class Book:
    asks: dict[Decimal, Decimal] = field(default_factory=dict)
    bids: dict[Decimal, Decimal] = field(default_factory=dict)
    exchange_ms: int | None = None
    last_exchange_ms: int | None = None
    received_ns: int = 0
    tick: Decimal | None = None
    needs_full_book: bool = False


def update_level(levels: dict[Decimal, Decimal], item: dict) -> None:
    price, size = decimal(item["price"]), decimal(item["size"])
    if not ZERO < price < ONE or size < ZERO:
        raise ValueError("invalid book level")
    if size:
        levels[price] = size
    else:
        levels.pop(price, None)


class Feed:
    def __init__(self, selections: list[tuple[str, str]]):
        self.selections = selections
        self.markets: list[Market] = []
        self.books: dict[str, Book] = {}
        self.generation = ""
        self.online = False
        self.ws_rtt_ms: int | None = None
        self.trades: deque[dict] = deque(maxlen=16)
        self.last_error = ""
        self.closed_conditions: set[str] = set()
        self.revision = ""
        self.history: OrderedDict[str, tuple[dict, dict[str, int]]] = OrderedDict()

    def reset(self) -> None:
        self.online = False
        self.ws_rtt_ms = None
        self.trades.clear()
        self.books.clear()
        self.closed_conditions.clear()
        self.history.clear()
        self.generation = uuid4().hex
        self.revision = uuid4().hex

    def change(self) -> None:
        self.revision = uuid4().hex

    def apply(self, message: dict) -> None:
        kind = message.get("event_type")
        if kind == "last_trade_price":
            self.record_trade(message)
            return
        if kind not in ("book", "price_change", "tick_size_change", "market_resolved"):
            return
        stamp = millis(message.get("timestamp"))
        seen = time.monotonic_ns()
        by_token = {market.token: market for market in self.markets}
        if kind == "market_resolved":
            condition = message.get("market")
            if isinstance(condition, str):
                self.closed_conditions.add(condition)
            self.markets = [Market(m.slug, m.outcome, m.token, m.condition, m.tick, m.minimum, m.rate, False, m.payout, m.rules)
                            if m.condition == condition else m for m in self.markets]
            self.change()
            return
        if kind == "price_change":
            for item in message["price_changes"]:
                token = str(item["asset_id"])
                book = self.books.get(token)
                if book is None:
                    continue
                if message.get("market") != by_token[token].condition:
                    raise ValueError("price change condition does not match token")
                if stamp is not None and book.last_exchange_ms is not None and stamp < book.last_exchange_ms:
                    continue
                side = item["side"]
                if side not in ("BUY", "SELL"):
                    raise ValueError("invalid price change side")
                update_level(book.bids if side == "BUY" else book.asks, item)
                self.touch(book, stamp, seen)
            return
        token = str(message["asset_id"])
        market = by_token.get(token)
        if market is None:
            return
        if message.get("market") != market.condition:
            raise ValueError("book condition does not match token")
        prior = self.books.get(token)
        if stamp is not None and prior and prior.last_exchange_ms is not None and stamp < prior.last_exchange_ms:
            return
        if kind == "tick_size_change":
            if prior is not None:
                prior.tick = decimal(message["new_tick_size"])
                self.change()
            return
        book = Book(tick=prior.tick if prior else None)
        for key, levels in (("bids", book.bids), ("asks", book.asks)):
            for item in message[key]:
                update_level(levels, item)
        self.books[token] = book
        self.touch(book, stamp, seen)

    def record_trade(self, message: dict) -> None:
        token = str(message.get("asset_id", ""))
        side = message.get("side")
        if side not in ("BUY", "SELL") or not any(m.token == token and m.condition == message.get("market")
                                                   for m in self.markets):
            return
        stamp = millis(message.get("timestamp"))
        try:
            price = decimal(message.get("price"))
            size = decimal(message.get("size")) if message.get("size") is not None else None
        except (ValueError, ArithmeticError):
            return
        if stamp is None or not ZERO < price < ONE or (size is not None and size <= ZERO):
            return
        self.trades.appendleft({"token": token, "side": side, "price": text(price),
                                "size": text(size) if size is not None else None, "exchange_ms": stamp})

    def touch(self, book: Book, stamp: int | None, seen: int) -> None:
        if stamp is None:
            book.needs_full_book = True
        else:
            book.last_exchange_ms = stamp
        book.exchange_ms = None if book.needs_full_book else stamp
        book.received_ns = seen
        self.change()

    def snapshot(self) -> dict:
        if self.revision not in self.history:
            received = {}
            legs = []
            for market in self.markets:
                book = self.books.get(market.token)
                received[market.token] = book.received_ns if book else 0
                tick = book.tick if book and book.tick is not None else market.tick
                leg = {"slug": market.slug, "outcome": market.outcome, "token": market.token, "rules_fingerprint": market.rules,
                       "exchange_ms": book.exchange_ms if book else None, "tick_size": text(tick),
                       "min_order_size": text(market.minimum), "fee_rate": text(market.rate) if market.rate is not None else None,
                       "accepting": market.accepting, "payout": text(market.payout) if market.payout is not None else None,
                       "asks": self.levels(book.asks) if book else [], "bids": self.levels(book.bids, reverse=True) if book else []}
                legs.append(leg)
            record = {"generation": self.generation, "snapshot_id": self.revision, "legs": legs}
            self.history[self.revision] = (record, received)
            if len(self.history) > 64:
                self.history.popitem(last=False)
        record, received = self.history[self.revision]
        now = time.monotonic_ns()
        return {**record, "ws_ok": self.online, "ws_rtt_ms": self.ws_rtt_ms, "feed_error": self.last_error,
                "trades": list(self.trades), "legs": [
            {**leg, "receive_age_ms": (now - received[leg["token"]]) // 1_000_000
             if received[leg["token"]] else None} for leg in record["legs"]
        ]}

    @staticmethod
    def levels(levels: dict[Decimal, Decimal], reverse: bool = False) -> list[dict]:
        return [{"price": text(p), "size": text(levels[p])} for p in sorted(levels, reverse=reverse)]

    def fill(self, request_id: str, generation: str, snapshot_id: str, token: str,
             side: str, shares: str, limit: str, intent: str) -> dict:
        if intent != "taker":
            raise ValueError(MAKER_BLOCKED)
        if not self.online or generation != self.generation or snapshot_id not in self.history:
            raise ValueError("snapshot no longer available")
        record, _ = self.history[snapshot_id]
        leg = next((value for value in record["legs"] if value["token"] == token), None)
        current = next((value for value in self.markets if value.token == token), None)
        if not leg or not current or not current.accepting or leg["fee_rate"] is None or leg["exchange_ms"] is None:
            raise ValueError("market or fee not available")
        current_book = self.books.get(token)
        current_tick = current_book.tick if current_book and current_book.tick is not None else current.tick
        if decimal(leg["tick_size"]) != current_tick or decimal(leg["min_order_size"]) != current.minimum or decimal(leg["fee_rate"]) != current.rate or leg["rules_fingerprint"] != current.rules:
            raise ValueError("market parameters changed after snapshot")
        with localcontext() as ctx:
            ctx.prec = 80
            quantity, bound, rate = decimal(shares), decimal(limit), decimal(leg["fee_rate"])
            if not ZERO < quantity or quantity < current.minimum or not ZERO < bound < ONE or bound % current_tick:
                raise ValueError("invalid quantity or price limit")
            if side not in ("buy", "sell"):
                raise ValueError("invalid side")
            remaining, fills = quantity, []
            for level in leg["asks" if side == "buy" else "bids"]:
                price = decimal(level["price"])
                if (side == "buy" and price > bound) or (side == "sell" and price < bound):
                    break
                if price % current_tick:
                    raise ValueError("off-tick book level")
                amount = min(remaining, decimal(level["size"]))
                fee = (amount * rate * price * (ONE - price)).quantize(FEE_PRECISION, rounding=ROUND_HALF_UP)
                fills.append({"price": text(price), "shares": text(amount), "fee": text(fee)})
                remaining -= amount
                if not remaining:
                    break
            status = "filled" if not remaining else "partial" if fills else "unfilled"
            return {"request_id": request_id, "generation": generation, "snapshot_id": snapshot_id,
                    "token": token, "side": side, "status": status, "fills": fills}

    async def stream(self, client: httpx.AsyncClient) -> None:
        self.reset()
        self.markets = await markets(client, self.selections)
        self.change()
        if not any(market.accepting for market in self.markets):
            await asyncio.sleep(15)
            return
        async with connect(WS, ping_interval=None, open_timeout=8, close_timeout=2, proxy=None) as ws:
            await ws.send(json.dumps({"type": "market", "assets_ids": [m.token for m in self.markets],
                                      "custom_feature_enabled": True}))
            self.online = True
            self.last_error = ""
            tasks = [
                asyncio.create_task(self.receive(ws)),
                asyncio.create_task(self.refresh(client)),
                asyncio.create_task(self.measure_rtt(ws)),
            ]
            try:
                await asyncio.gather(*tasks)
            finally:
                for task in tasks:
                    task.cancel()
                await asyncio.gather(*tasks, return_exceptions=True)

    async def measure_rtt(self, ws) -> None:
        while True:
            # Protocol-level pings have a unique payload, independent of the text heartbeat.
            pong = await ws.ping()
            try:
                latency = await asyncio.wait_for(asyncio.shield(pong), timeout=3)
            except asyncio.TimeoutError:
                self.ws_rtt_ms = None
                # Keep at most one unanswered latency probe per connection.
                await pong
            else:
                self.ws_rtt_ms = round(latency * 1000)
            await asyncio.sleep(10)

    async def receive(self, ws) -> None:
        last_frame = last_ping = time.monotonic()
        while True:
            try:
                payload = await asyncio.wait_for(ws.recv(), 1)
            except asyncio.TimeoutError:
                payload = None
            now = time.monotonic()
            if now - last_ping >= 10:
                await ws.send("PING")
                last_ping = now
            if payload is None:
                if now - last_frame > 25:
                    raise TimeoutError("market WebSocket heartbeat expired")
                continue
            last_frame = now
            if payload == "PONG":
                continue
            data = json.loads(payload, parse_float=Decimal)
            for event in data if isinstance(data, list) else [data]:
                if isinstance(event, dict):
                    self.apply(event)

    async def refresh(self, client: httpx.AsyncClient) -> None:
        while True:
            await asyncio.sleep(15)
            updated = await markets(client, self.selections)
            if [m.token for m in updated] != [m.token for m in self.markets]:
                raise ValueError("market token changed")
            updated = [Market(m.slug, m.outcome, m.token, m.condition, m.tick, m.minimum, m.rate, False, m.payout, m.rules)
                       if m.condition in self.closed_conditions else m for m in updated]
            if updated != self.markets:
                self.markets = updated
                self.change()

    async def run(self) -> None:
        async with httpx.AsyncClient(timeout=8) as client:
            while True:
                try:
                    await self.stream(client)
                except Exception as exc:
                    self.last_error = str(exc)[:240] or type(exc).__name__
                    LOG.warning("market feed disconnected: %s", exc)
                finally:
                    self.reset()
                await asyncio.sleep(1)

    async def fixture(self) -> None:
        self.reset()
        self.markets = [Market(slug, outcome, f"fixture-{i}", f"condition-{i}", Decimal("0.01"),
                               Decimal(5), Decimal("0.05"), True)
                        for i, (slug, outcome) in enumerate(self.selections)]
        sizes = [100, 35, 95, 160, 240, 66, 80, 137, 210, 85, 260, 100, 320, 125, 420, 170, 155, 225, 310, 500]
        asks = [{"price": text(Decimal(48+i).scaleb(-2)), "size": str(q)} for i, q in enumerate(sizes)]
        bids = [{"price": text(Decimal(45-i).scaleb(-2)), "size": str(q if i == 0 else q+24)}
                for i, q in enumerate(sizes)]
        self.online = True
        while True:
            stamp = time.time_ns() // 1_000_000
            for market in self.markets:
                self.apply({"event_type": "book", "asset_id": market.token, "market": market.condition,
                            "timestamp": str(stamp), "asks": asks, "bids": bids})
            await asyncio.sleep(0.1)


async def serve(args) -> None:
    feed = Feed([(args.a, args.outcome_a), (args.b, args.outcome_b)])
    server = MCPServer("polymarket-taker-paper")

    @server.tool()
    async def get_pair_snapshot() -> dict:
        return feed.snapshot()

    @server.tool()
    async def simulate_fill(request_id: str, generation: str, snapshot_id: str, token: str,
                      side: str, shares: str, limit: str, intent: str) -> dict:
        if args.miss_b and token == "fixture-1" and side == "buy":
            await asyncio.sleep(1)
        return feed.fill(request_id, generation, snapshot_id, token, side, shares, limit, intent)

    task = asyncio.create_task(feed.fixture() if args.fixture else feed.run())
    try:
        await server.run_stdio_async()
    finally:
        task.cancel()
        await asyncio.gather(task, return_exceptions=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--a", required=True)
    parser.add_argument("--b", required=True)
    parser.add_argument("--outcome-a", choices=("Yes", "No"), required=True)
    parser.add_argument("--outcome-b", choices=("Yes", "No"), required=True)
    parser.add_argument("--fixture", action="store_true")
    parser.add_argument("--miss-b", action="store_true")
    logging.basicConfig(level=logging.WARNING)
    asyncio.run(serve(parser.parse_args()))

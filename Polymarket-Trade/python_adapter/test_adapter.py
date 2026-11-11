import unittest
from decimal import Decimal

from adapter import Feed, Market


class FeedTests(unittest.TestCase):
    def setUp(self):
        self.feed = Feed([("a", "No"), ("b", "No")])
        self.feed.reset()
        self.feed.online = True
        self.feed.markets = [
            Market(slug, "No", slug, "condition-" + slug, Decimal(".01"), Decimal(5), Decimal(".05"), True)
            for slug in ("a", "b")
        ]
        for token in ("a", "b"):
            self.full_book(token, "1800000000000")

    def full_book(self, token, stamp):
        self.feed.apply({"event_type": "book", "asset_id": token, "market": "condition-" + token,
                         "timestamp": stamp, "asks": [{"price": ".48", "size": "8"}],
                         "bids": [{"price": ".45", "size": "10"}]})

    def delta(self, stamp, size):
        self.feed.apply({"event_type": "price_change", "market": "condition-a", "timestamp": stamp,
                         "price_changes": [{"asset_id": "a", "side": "SELL", "price": ".48", "size": size}]})

    def fill(self, snapshot):
        return self.feed.fill("req", snapshot["generation"], snapshot["snapshot_id"], "a",
                              "buy", "5", ".48", "taker")

    def test_snapshot_pins_depth_and_exchange_time(self):
        first = self.feed.snapshot()
        self.delta("1800000000001", "3")
        second = self.feed.snapshot()
        self.assertNotEqual(first["snapshot_id"], second["snapshot_id"])
        self.assertEqual(first["legs"][0]["exchange_ms"], 1800000000000)
        self.assertEqual(second["legs"][0]["exchange_ms"], 1800000000001)
        self.assertEqual(self.fill(first)["fills"], [{"price": "0.48", "shares": "5", "fee": "0.06240"}])
        self.assertEqual(self.fill(second)["status"], "partial")
        self.delta("1800000000000", "0")
        self.assertEqual(self.feed.snapshot()["snapshot_id"], second["snapshot_id"])
        self.delta("1800000000002", "0")
        self.assertEqual(self.feed.snapshot()["legs"][0]["asks"], [])

    def test_missing_exchange_time_requires_full_book(self):
        self.delta(None, "6")
        self.assertIsNone(self.feed.snapshot()["legs"][0]["exchange_ms"])
        self.delta("1800000000001", "4")
        self.assertIsNone(self.feed.snapshot()["legs"][0]["exchange_ms"])
        self.full_book("a", "1800000000002")
        self.assertEqual(self.feed.snapshot()["legs"][0]["exchange_ms"], 1800000000002)

    def test_old_generation_and_old_tick_rejected(self):
        first = self.feed.snapshot()
        self.feed.apply({"event_type": "tick_size_change", "asset_id": "a", "market": "condition-a",
                         "timestamp": "1800000000001", "new_tick_size": ".001"})
        with self.assertRaisesRegex(ValueError, "parameters changed"):
            self.fill(first)
        self.feed.reset()
        with self.assertRaisesRegex(ValueError, "snapshot no longer"):
            self.fill(first)

    def test_latency_is_live_metadata_and_cleared_on_reconnect(self):
        first = self.feed.snapshot()
        self.assertIsNone(first["ws_rtt_ms"])
        self.feed.ws_rtt_ms = 25
        second = self.feed.snapshot()
        self.assertEqual(second["ws_rtt_ms"], 25)
        self.assertEqual(second["snapshot_id"], first["snapshot_id"])
        self.assertEqual(second["legs"][0]["exchange_ms"], first["legs"][0]["exchange_ms"])
        self.feed.reset()
        self.assertIsNone(self.feed.snapshot()["ws_rtt_ms"])

    def test_public_trade_is_independent_of_order_book_snapshot(self):
        first = self.feed.snapshot()
        trade = {"event_type": "last_trade_price", "asset_id": "a", "side": "BUY",
                 "market": "condition-a", "price": ".48", "size": "8", "timestamp": "1800000000999"}
        self.feed.apply(trade)
        self.feed.apply({**trade, "market": "unrelated"})
        self.feed.apply({**trade, "size": "-1"})
        second = self.feed.snapshot()
        self.assertEqual(second["trades"], [{"token": "a", "side": "BUY", "price": "0.48",
                                             "size": "8", "exchange_ms": 1800000000999}])
        self.assertEqual(second["snapshot_id"], first["snapshot_id"])
        self.assertEqual(second["legs"][0]["exchange_ms"], first["legs"][0]["exchange_ms"])
        self.feed.reset()
        self.assertEqual(self.feed.snapshot()["trades"], [])

    def test_maker_blocked(self):
        s = self.feed.snapshot()
        with self.assertRaisesRegex(ValueError, "Maker blocked"):
            self.feed.fill("req", s["generation"], s["snapshot_id"], "a", "buy", "5", ".48", "maker")


if __name__ == "__main__":
    unittest.main()

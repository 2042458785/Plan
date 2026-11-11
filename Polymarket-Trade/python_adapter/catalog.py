"""Read-only event and market discovery for the terminal setup screen."""

import json
import re
import sys
from urllib.parse import parse_qs, urlsplit

import httpx

GAMMA = "https://gamma-api.polymarket.com"
SLUG = re.compile(r"[a-z0-9][a-z0-9-]{0,249}\Z")


def rows(query):
    url = urlsplit(query)
    target = ""
    selected = ""
    if url.scheme:
        path = url.path.strip("/").split("/")
        if url.scheme != "https" or url.hostname not in ("polymarket.com", "www.polymarket.com") or "event" not in path:
            raise ValueError("expected a Polymarket event link")
        index = path.index("event") + 1
        if index >= len(path) or not SLUG.fullmatch(path[index]):
            raise ValueError("event identifier missing from link")
        target = path[index]
        selected = parse_qs(url.query).get("marketSlug", [""])[0]
    elif "-" in query and SLUG.fullmatch(query):
        target = query

    with httpx.Client(timeout=8) as client:
        events = None
        if target:
            response = client.get(f"{GAMMA}/events/slug/{target}")
            if response.is_success:
                events = [response.json()]
            elif url.scheme or response.status_code != 404:
                response.raise_for_status()
        if events is None:
            response = client.get(f"{GAMMA}/public-search", params={
                "q": query,
                "limit_per_type": 12,
                "search_profiles": "false",
                "search_tags": "false",
            })
            response.raise_for_status()
            events = response.json().get("events") or []
    result = []
    for event in events[:12]:
        markets = []
        for market in event.get("markets") or []:
            if not SLUG.fullmatch(market.get("slug") or ""):
                continue
            outcomes = market.get("outcomes") or []
            if isinstance(outcomes, str):
                try:
                    outcomes = json.loads(outcomes)
                except ValueError:
                    continue
            if len(outcomes) != 2 or set(outcomes) != {"Yes", "No"}:
                continue
            markets.append({
                "slug": market["slug"],
                "label": market.get("groupItemTitle") or market.get("question") or market["slug"],
                "accepting": market.get("acceptingOrders") is True and not market.get("closed", False),
            })
        if selected:
            markets.sort(key=lambda market: market["slug"] != selected)
        if markets:
            result.append({"title": event.get("title") or event.get("slug") or "", "markets": markets})
    result.sort(key=lambda event: not any(market["accepting"] for market in event["markets"]))
    return result


def book_preview(slug, outcome):
    if not SLUG.fullmatch(slug) or outcome not in ("Yes", "No"):
        raise ValueError("invalid market or outcome")
    with httpx.Client(timeout=7) as client:
        response = client.get(f"{GAMMA}/markets/slug/{slug}")
        response.raise_for_status()
        market = response.json()
        outcomes = json.loads(market["outcomes"])
        tokens = json.loads(market["clobTokenIds"])
        if len(outcomes) != 2 or len(tokens) != 2 or set(outcomes) != {"Yes", "No"}:
            raise ValueError("unsupported market outcomes")
        token = tokens[outcomes.index(outcome)]
        response = client.get("https://clob.polymarket.com/book", params={"token_id": token})
        response.raise_for_status()
        book = response.json()
    if str(book.get("asset_id")) != token or book.get("market") != market.get("conditionId"):
        raise ValueError("order book identity does not match the selected market")
    return {
        "slug": slug,
        "outcome": outcome,
        "token": token,
        "exchange_ms": int(book["timestamp"]),
        "tick_size": str(book["tick_size"]),
        "min_order_size": str(book["min_order_size"]),
        "accepting": bool(market.get("acceptingOrders")),
        "bids": book["bids"],
        "asks": book["asks"],
    }


if __name__ == "__main__":
    try:
        args = sys.argv[1:]
        if len(args) == 3 and args[0] == "--book":
            result = book_preview(args[1], args[2])
        elif len(args) == 1:
            result = rows(args[0])
        else:
            raise ValueError("expected a search query or --book SLUG OUTCOME")
        print(json.dumps(result, ensure_ascii=False))
    except (httpx.HTTPError, ValueError, KeyError, TypeError) as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)

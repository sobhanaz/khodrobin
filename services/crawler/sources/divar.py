"""Divar — https://divar.ir/s/{city}/car

Divar server-renders a JSON-LD array of schema.org Car objects into the listing
page. That is a documented, machine-readable format the site publishes on
purpose for search engines, so we read it instead of scraping the DOM or
replaying an internal API: it is stabler, cheaper, and unambiguous about intent.

Note what it does *not* give us: prices are in **rials**, the year arrives twice
(Jalali in productionDate, Gregorian in vehicleModelDate), and mileage is a
string inside a QuantitativeValue. Reconciling that is the normalizer's job.
"""
from __future__ import annotations

import json
import re
from typing import Any

import httpx

from politeness import USER_AGENT, HostLimiter

HOST = "divar.ir"
LIST_URL = "https://divar.ir/s/{city}/car"
BRAND_URL = "https://divar.ir/s/{city}/car/{brand}"

# Brand-filtered pages return 48 cars against 24 for the unfiltered feed, so
# fanning out over brands is both deeper coverage and fewer requests per car.
# Ordered by how common the marque is on the Iranian used market.
BRANDS = [
    "pride", "peugeot", "samand", "tiba", "quik", "dena", "tara", "shahin",
    "rana", "arisan", "saina", "kia", "hyundai", "toyota", "renault",
    "mvm", "chery", "jac", "kmc", "bmw", "benz", "mazda", "nissan",
]
_LD = re.compile(r'<script[^>]*application/ld\+json[^>]*>(.*?)</script>', re.S)


def _cars_from_html(html: str) -> list[dict[str, Any]]:
    for block in _LD.findall(html):
        try:
            data = json.loads(block)
        except json.JSONDecodeError:
            continue
        if isinstance(data, list) and data and data[0].get("@type") == "Car":
            return data
    return []


def _urls(cities: list[str], brands: list[str] | None) -> list[tuple[str, str]]:
    """(city, url) pairs to fetch: the whole-category feed plus one per brand."""
    pairs = [(city, LIST_URL.format(city=city)) for city in cities]
    for city in cities:
        for brand in (brands if brands is not None else BRANDS):
            pairs.append((city, BRAND_URL.format(city=city, brand=brand)))
    return pairs


# Measured, not guessed: at ~1 req/s Divar starts returning empty 200s within a
# dozen requests. At 4s it keeps answering. Slower and complete beats fast and
# silently empty.
MIN_INTERVAL = 4.0


def fetch(limiter: HostLimiter, cities: list[str],
          brands: list[str] | None = None) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    limiter.min_interval_for[HOST] = MIN_INTERVAL
    out: list[dict[str, Any]] = []
    with httpx.Client(headers={"User-Agent": USER_AGENT}, timeout=30, follow_redirects=True) as c:
        for city, url in _urls(cities, brands):
            if limiter.is_open(HOST):
                break
            limiter.wait(HOST)
            try:
                r = c.get(url)
            except httpx.HTTPError:
                limiter.record(HOST, 599)
                continue
            limiter.record(HOST, r.status_code)
            if r.status_code != 200:
                continue
            cars = _cars_from_html(r.text)
            if not cars:
                # A 200 with no JSON-LD is either a brand slug Divar does not
                # know or a soft block. Both mean "stop asking so fast".
                limiter.record_empty(HOST)
                continue
            for car in cars:
                out.append({"source": "divar", "city": city, "list_url": url, "payload": car})
    return out


def source_id(raw: dict[str, Any]) -> str:
    """Divar's per-listing token, taken from the tail of the ad URL."""
    url = raw["payload"].get("url", "")
    return url.rstrip("/").rsplit("/", 1)[-1]

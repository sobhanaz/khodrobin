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


def fetch(limiter: HostLimiter, cities: list[str]) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    out: list[dict[str, Any]] = []
    with httpx.Client(headers={"User-Agent": USER_AGENT}, timeout=30, follow_redirects=True) as c:
        for city in cities:
            limiter.wait(HOST)
            url = LIST_URL.format(city=city)
            try:
                r = c.get(url)
            except httpx.HTTPError:
                limiter.record(HOST, 599)
                continue
            limiter.record(HOST, r.status_code)
            if r.status_code != 200:
                continue
            for car in _cars_from_html(r.text):
                out.append({"source": "divar", "city": city, "list_url": url, "payload": car})
    return out


def source_id(raw: dict[str, Any]) -> str:
    """Divar's per-listing token, taken from the tail of the ad URL."""
    url = raw["payload"].get("url", "")
    return url.rstrip("/").rsplit("/", 1)[-1]

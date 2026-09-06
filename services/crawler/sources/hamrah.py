"""Hamrah-Mechanic — https://www.hamrah-mechanic.com/cars-for-sale/

A Next.js app, so the full listing payload is embedded in __NEXT_DATA__. These
cars are dealer-inspected, which makes this the cleanest of the three sources —
and the most useful contradiction detector: when Hamrah-Mechanic and Divar
disagree about a car's mileage, the inspected figure is usually the honest one.

Prices are in **rials**, like Divar. Years are Gregorian only.
"""
from __future__ import annotations

import json
import re
from typing import Any

import httpx

from politeness import USER_AGENT, HostLimiter

HOST = "www.hamrah-mechanic.com"
LIST_URL = "https://www.hamrah-mechanic.com/cars-for-sale/"
_NEXT = re.compile(r'<script id="__NEXT_DATA__"[^>]*>(.*?)</script>', re.S)


def fetch(limiter: HostLimiter, pages: int = 2) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    out: list[dict[str, Any]] = []
    with httpx.Client(headers={"User-Agent": USER_AGENT}, timeout=30, follow_redirects=True) as c:
        for page in range(1, pages + 1):
            limiter.wait(HOST)
            url = LIST_URL if page == 1 else f"{LIST_URL}?page={page}"
            try:
                r = c.get(url)
            except httpx.HTTPError:
                limiter.record(HOST, 599)
                break
            limiter.record(HOST, r.status_code)
            if r.status_code != 200:
                break
            m = _NEXT.search(r.text)
            if not m:
                break
            try:
                data = json.loads(m.group(1))
            except json.JSONDecodeError:
                break
            cars = data.get("props", {}).get("pageProps", {}).get("cars", {}).get("list", [])
            if not cars:
                break
            for car in cars:
                out.append({"source": "hamrah", "city": None, "list_url": url, "payload": car})
    return out


def source_id(raw: dict[str, Any]) -> str:
    return str(raw["payload"]["orderId"])

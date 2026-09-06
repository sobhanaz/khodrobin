"""Khodro45 — https://khodro45.com/api/v2/car_listing/

The cleanest of the four sources. It is a Django REST Framework endpoint with
ordinary limit/offset pagination, and it already publishes canonical slugs
(`chery-tiggo5`) alongside Persian and English titles, plus an explicit trim.

Two fields the other sources do not have:

* ``car_specifications.is_klm_matched`` — whether the odometer reading was
  verified against service records. That is a data-integrity signal the other
  three simply do not offer, and it feeds the plausibility flags.
* ``guaranteed`` — dealer-backed listings, useful context on a price.

Prices are **tomans**. Confirmed the same way as the others: a 1395 Chery
Tiggo 5 at 2,079,000,000 is ~2.08 billion tomans, which matches the market;
read as rials it would be an implausible 208 million.
"""
from __future__ import annotations

from typing import Any

import httpx

from politeness import USER_AGENT, HostLimiter

HOST = "khodro45.com"
API_URL = "https://khodro45.com/api/v2/car_listing/"
PAGE_SIZE = 30


def fetch(limiter: HostLimiter, pages: int = 2) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    out: list[dict[str, Any]] = []
    headers = {
        "User-Agent": USER_AGENT,
        "Accept": "application/json",
        "Referer": "https://khodro45.com/used-car/",
    }
    with httpx.Client(headers=headers, timeout=30, follow_redirects=True) as c:
        for page in range(pages):
            limiter.wait(HOST)
            params = {
                "limit": PAGE_SIZE,
                "offset": page * PAGE_SIZE,
                "ordering": "-created_time",
            }
            try:
                r = c.get(API_URL, params=params)
            except httpx.HTTPError:
                limiter.record(HOST, 599)
                break
            limiter.record(HOST, r.status_code)
            if r.status_code != 200:
                break
            body = r.json()
            results = body.get("results") or []
            if not results:
                limiter.record_empty(HOST)
                break
            for ad in results:
                out.append({"source": "khodro45", "city": None, "list_url": API_URL, "payload": ad})
            if not body.get("next"):
                break
    return out


def source_id(raw: dict[str, Any]) -> str:
    p = raw["payload"]
    return str(p.get("slug") or p.get("identifier"))

"""Bama — https://bama.ir/cad/api/search

Bama exposes a public JSON search API that the site's own front end calls. It
returns richer car attributes than the other two sources — trim, body_status
(«بدون رنگ»), fuel, body type, cylinder count — which makes it the best source
for attributes and a useful arbiter when Divar and Hamrah-Mechanic disagree.

Prices here are **tomans**, formatted with commas, while Divar and
Hamrah-Mechanic use rials. That single inconsistency is worth a slide.
"""
from __future__ import annotations

from typing import Any

import httpx

from politeness import USER_AGENT, HostLimiter

HOST = "bama.ir"
API_URL = "https://bama.ir/cad/api/search"


def fetch(limiter: HostLimiter, pages: int = 2) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    out: list[dict[str, Any]] = []
    headers = {"User-Agent": USER_AGENT, "Accept": "application/json", "Referer": "https://bama.ir/car"}
    with httpx.Client(headers=headers, timeout=30, follow_redirects=True) as c:
        for page in range(pages):
            limiter.wait(HOST)
            try:
                r = c.get(API_URL, params={"pageIndex": page})
            except httpx.HTTPError:
                limiter.record(HOST, 599)
                break
            limiter.record(HOST, r.status_code)
            if r.status_code != 200:
                break
            body = r.json()
            for ad in body.get("data", {}).get("ads", []):
                # The feed interleaves banners with real ads.
                if ad.get("type") != "ad" or not ad.get("detail"):
                    continue
                out.append({"source": "bama", "city": None, "list_url": API_URL, "payload": ad})
            if not body.get("metadata", {}).get("has_next"):
                break
    return out


def source_id(raw: dict[str, Any]) -> str:
    return raw["payload"]["detail"]["code"]

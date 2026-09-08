"""Sheypoor — https://www.sheypoor.com/s/{city}/car

Sheypoor is a Next.js App Router site, so the first instinct is to reach for
``__NEXT_DATA__`` the way the Hamrah-Mechanic adapter does. That script tag does
not exist here: the App Router streams its payload as ``self.__next_f.push``
fragments, which is an internal serialization format with no stability promise.

We do not need it. Every listing page also server-renders a schema.org
``CollectionPage`` whose ``mainEntity.itemListElement`` is an array of ``Vehicle``
objects — the same class of published, machine-readable data Divar gives us, and
published for the same reason. Read that instead. No browser automation.

Two things this feed does *not* tell you, both of which cost a re-crawl if you
learn them late:

* **Prices are rials**, like Divar, unlike Bama/Hamrah/Khodro45. The payload
  says ``priceCurrency: "IRR"``, but a field name is not evidence — Bama's own
  field naming would tell you the same story and Bama is tomans. Verified
  against live market prices instead; see normalize.from_sheypoor.
* **There is no model field and no trim field.** Only ``brand.name`` in Persian
  and the seller's free-text ``name``. The model has to be resolved out of the
  title, which extract.resolve already does — but only because the title is fed
  in. Drop the title and this source resolves to a brand and nothing else.
"""
from __future__ import annotations

import json
import re
from typing import Any

import httpx

from politeness import USER_AGENT, HostLimiter

HOST = "www.sheypoor.com"
LIST_URL = "https://www.sheypoor.com/s/{city}/car"
_LD = re.compile(r'<script[^>]*application/ld\+json[^>]*>(.*?)</script>', re.S)

# robots.txt disallows `/*?` and then re-allows `/*page_num=`. That pair is not
# an accident — it is the site saying pagination is the one query string it
# wants crawled — so page_num is the only parameter we ever append.
PAGE_PARAM = "page_num"


def _blocks(html: str) -> list[Any]:
    out = []
    for block in _LD.findall(html):
        try:
            out.append(json.loads(block))
        except json.JSONDecodeError:
            continue
    return out


def _cars(blocks: list[Any]) -> list[dict[str, Any]]:
    """The Vehicle array out of the CollectionPage block.

    Filtered on ``@type`` rather than trusting position: the page carries a
    FAQPage and a BreadcrumbList in the same ld+json shape, and the ordering of
    the three is a rendering detail nobody promised us.
    """
    for data in blocks:
        main = data.get("mainEntity") if isinstance(data, dict) else None
        if not isinstance(main, dict):
            continue
        cars = [i for i in main.get("itemListElement") or []
                if isinstance(i, dict) and i.get("@type") == "Vehicle"]
        if cars:
            return cars
    return []


def _city_fa(blocks: list[Any], city: str) -> str | None:
    """The city's Persian name, from the page's own breadcrumb.

    A Vehicle item carries no location at all — the city is a property of the
    page, not of the ad. We know the slug because we built the URL, but the
    product shows «تهران», not «tehran», and hardcoding a slug→Persian table
    here would be a second copy of something the page already states. Matched on
    the breadcrumb's own URL rather than on its position, because position 2 is
    the city only for province-level cities.
    """
    want = f"/s/{city}"
    for data in blocks:
        if not isinstance(data, dict) or data.get("@type") != "BreadcrumbList":
            continue
        for entry in data.get("itemListElement") or []:
            item = entry.get("item")
            url = item if isinstance(item, str) else (item or {}).get("@id", "")
            if str(url).rstrip("/").endswith(want):
                return entry.get("name") or (item or {}).get("name")
    return None


def fetch(limiter: HostLimiter, cities: list[str], pages: int = 2) -> list[dict[str, Any]]:
    if limiter.is_open(HOST):
        return []
    out: list[dict[str, Any]] = []
    with httpx.Client(headers={"User-Agent": USER_AGENT}, timeout=30, follow_redirects=True) as c:
        for city in cities:
            base = LIST_URL.format(city=city)
            for page in range(1, pages + 1):
                if limiter.is_open(HOST):
                    return out
                limiter.wait(HOST)
                url = base if page == 1 else f"{base}?{PAGE_PARAM}={page}"
                try:
                    r = c.get(url)
                except httpx.HTTPError:
                    limiter.record(HOST, 599)
                    break
                limiter.record(HOST, r.status_code)
                if r.status_code != 200:
                    break
                blocks = _blocks(r.text)
                cars = _cars(blocks)
                if not cars:
                    # An unknown city or brand slug 404s here, so a 200 that
                    # carries no Vehicle array is not "you asked for nothing" —
                    # it is either a soft block or the end of the results, and
                    # both mean stop asking this host for now.
                    limiter.record_empty(HOST)
                    break
                city_fa = _city_fa(blocks, city)
                for car in cars:
                    # The only edit made to a raw payload anywhere in this
                    # crawler, and it is here because the fact genuinely lives
                    # on the page rather than in the record. Named loudly so
                    # nobody later mistakes it for something Sheypoor sent.
                    out.append({"source": "sheypoor", "city": city, "list_url": url,
                                "payload": {**car, "sheypoor_city_fa": city_fa}})
    return out


def source_id(raw: dict[str, Any]) -> str:
    """Sheypoor's listing id, from offers.sku.

    Also the numeric tail of the ad URL, but the URL carries a slug built from
    the seller's title — an editable field. The sku survives an edit; the slug
    does not, and a source_id that changes when a seller fixes a typo turns one
    ad into two offers on the card.
    """
    return str((raw["payload"].get("offers") or {}).get("sku") or "")

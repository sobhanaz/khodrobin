"""Fetch per-listing detail pages for the shortlist, and write /data/details.json.

Selectivity is the whole design, not an optimisation. The index holds ~8,800
offers; one detail request each on a three-hour cycle is ~70,000 requests a day,
which is not a crawl, it is an outage someone else has to absorb. Divar already
answers HTTP 200 with the JSON-LD simply missing when it decides we are asking
too fast, so the punishment for getting this wrong is silent and looks like
success.

Four things keep the footprint honest, and all four are measured rather than
guessed:

* **Rank, then cut.** Only offers belonging to the top N specs, ranked exactly
  the way warm_explanations ranks them, because those are the cards a visitor
  sees first. Measured on the live index: the top 60 specs hold 288 offers.
* **A TTL.** A seller's paragraph does not change every three hours. Seven days
  by default, so a steady-state cycle fetches almost nothing.
* **Yield, not status.** ``politeness.record_empty`` treats a 200 with no data
  as a 429, which is the only way a soft block is visible at all.
* **A wall-clock budget.** Running out is the normal case on a first run and
  costs nothing: the shortlist is stable between cycles, so the next run
  continues where this one stopped.

The output merges into whatever is already on disk. A source that yields
nothing this cycle must not blank what it wrote last cycle — a blocked host
would otherwise erase a working feature and the logs would show a clean run.

Shape, one structure for five sources (see details.py):

    { "built_at": "<iso8601>",
      "details": { "<source>:<source_id>": { ...detail..., "fetched_at": ... } } }

What this job must never do is write a price, a mileage, or anything else the
index already holds. details.py's docstring has the two real Divar ads that
explain why, in the units they were published in.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import time
from collections import defaultdict
from datetime import datetime, timedelta, timezone

import httpx

import details
from politeness import USER_AGENT, HostLimiter
from sources import bama, divar, hamrah, khodro45, sheypoor
from warm_explanations import log, rank_for_warming

DEFAULT_INDEX = "/data/index.json"
DEFAULT_OUT = "/data/details.json"

# Per-source fetch plan. `url` defaults to the listing URL the index already
# stores; only a source whose detail data lives somewhere other than its own ad
# page needs to override it.
SOURCES: dict[str, dict] = {
    "divar": {"host": divar.HOST, "interval": divar.MIN_INTERVAL},
    "bama": {"host": bama.HOST},
    "sheypoor": {"host": sheypoor.HOST},
    "hamrah": {"host": hamrah.HOST},
    # Off, and this is a decision rather than an oversight. Khodro45's detail
    # page at /view/{slug}/ is a 2.3 KB client-rendered shell with no car data
    # in it, and robots.txt disallows both that path and /api, where the only
    # copy of the payload lives. Its sitemap-usedcar.xml publishes 991 URLs and
    # not one of them is a listing. Flip `enabled` if that policy changes; the
    # extractor and its test are already here and passing.
    "khodro45": {
        "host": khodro45.HOST,
        "enabled": False,
        "why": "robots.txt disallows /view/ (the page) and /api (the payload)",
        "url": lambda offer: f"https://khodro45.com/api/v1/car_listing/{offer['source_id']}/",
    },
}


def load_existing(path: pathlib.Path) -> dict:
    """Whatever is already on disk, or nothing if it cannot be read.

    Unlike history.json this file is regenerable — every entry is one HTTP
    request away — so a corrupt one is started over rather than moved aside for
    salvage. It is still logged, because starting over means several hundred
    requests we did not need to make and that should be visible.
    """
    if not path.exists():
        return {}
    try:
        stored = json.loads(path.read_text(encoding="utf-8")).get("details")
        if not isinstance(stored, dict):
            raise ValueError("no details map")
        return {k: v for k, v in stored.items() if isinstance(v, dict)}
    except (json.JSONDecodeError, OSError, ValueError) as exc:
        log(f"details at {path} unreadable ({exc!r}); refetching from scratch")
        return {}


def key_for(offer: dict) -> str | None:
    """"<source>:<source_id>", the same identity the index and the API use."""
    source, sid = offer.get("source"), offer.get("source_id")
    return f"{source}:{sid}" if source and sid else None


def is_fresh(entry: dict, ttl: timedelta, now: datetime) -> bool:
    try:
        fetched = datetime.fromisoformat(entry["fetched_at"])
    except (KeyError, TypeError, ValueError):
        # An entry with no readable timestamp cannot be aged, so it is refetched
        # rather than kept forever on the strength of a field nobody can parse.
        return False
    return now - fetched < ttl


def plan(specs: list[dict], stored: dict, ttl: timedelta, now: datetime) -> list[dict]:
    """The fetch order, and it is two interleaves rather than a flat list.

    Round by offer position first: every top spec gets its cheapest clean offer
    detailed before any spec gets its second. When the budget runs out — the
    normal case on a first run — that leaves 60 cards each with one enriched
    offer instead of 8 cards fully enriched and 52 with nothing.

    Then round-robin by source inside each round, because the limiter's waits
    are per-host: 4 seconds between two Divar requests, 1 between anyone else's.
    Spreading a round across hosts means one host's cooldown elapses while
    another host is being fetched instead of while nothing happens.
    """
    rounds: dict[int, dict[str, list[dict]]] = defaultdict(lambda: defaultdict(list))
    for spec in specs:
        for position, offer in enumerate(spec.get("offers") or []):
            source = offer.get("source")
            config = SOURCES.get(source or "")
            if not config or not config.get("enabled", True):
                continue
            key = key_for(offer)
            if not key or is_fresh(stored.get(key) or {}, ttl, now):
                continue
            rounds[position][source].append({"key": key, "source": source, "offer": offer})

    jobs: list[dict] = []
    for position in sorted(rounds):
        queues = list(rounds[position].values())
        for i in range(max(len(q) for q in queues)):
            jobs += [q[i] for q in queues if i < len(q)]
    return jobs


def fetch_url(job: dict) -> str:
    config = SOURCES[job["source"]]
    builder = config.get("url")
    return builder(job["offer"]) if builder else job["offer"]["url"]


def enrich(index_path: pathlib.Path, out_path: pathlib.Path, limit: int,
           budget_seconds: float, ttl_days: float) -> int:
    index = json.loads(index_path.read_text(encoding="utf-8"))
    all_specs = index.get("specs", [])
    now = datetime.now(timezone.utc)
    ttl = timedelta(days=ttl_days)
    stored = load_existing(out_path)

    # Merge, never replace. An entry survives for as long as its offer is still
    # somewhere in the index, not just while it is in this run's top N — the
    # shortlist moves every cycle and a card dropping to rank 61 must not lose
    # the detail panel it had at rank 59. What does expire is an offer that has
    # left the index entirely: the listing is gone and nobody can open it.
    offers_seen = sum(len(spec.get("offers") or []) for spec in all_specs)
    live = {k for spec in all_specs for offer in spec.get("offers") or []
            if (k := key_for(offer))}
    if stored and offers_seen and not live:
        # Offers exist and not one of them is addressable, which means an index
        # built before detail enrichment existed, not a catalogue that emptied
        # itself. An index with no offers at all is a different thing and does
        # expire, which is why offers_seen is in the condition.
        log(f"index has no addressable offers; keeping {len(stored)} stored details untouched")
        return 0
    fresh = {k: v for k, v in stored.items() if k in live}
    expired = len(stored) - len(fresh)

    jobs = plan(rank_for_warming(all_specs)[:limit], fresh, ttl, now)

    limiter = HostLimiter()
    for config in SOURCES.values():
        if config.get("interval"):
            limiter.min_interval_for[config["host"]] = config["interval"]

    counts: dict[str, dict[str, int]] = defaultdict(
        lambda: {"attempted": 0, "extracted": 0, "empty": 0, "blocked": 0})
    for source, config in SOURCES.items():
        if not config.get("enabled", True):
            counts[source]["blocked"] = 0  # materialise the row so the log says so

    started = time.monotonic()
    ran_out = False
    with httpx.Client(headers={"User-Agent": USER_AGENT}, timeout=30,
                      follow_redirects=True) as client:
        for job in jobs:
            if time.monotonic() - started > budget_seconds:
                ran_out = True
                break
            source = job["source"]
            host = SOURCES[source]["host"]
            if limiter.is_open(host):
                counts[source]["blocked"] += 1
                continue

            limiter.wait(host)
            counts[source]["attempted"] += 1
            try:
                response = client.get(fetch_url(job))
            except httpx.HTTPError as exc:
                limiter.record(host, 599)
                counts[source]["blocked"] += 1
                log(f"{source} {job['key']} fetch failed: {exc!r}")
                fresh.setdefault(job["key"], {"fetched_at": now.isoformat()})
                continue
            status = response.status_code
            if status != 200:
                # Only a real push-back is reported. `record` clears the strike
                # counter for every status it does not consider an error, and a
                # 404 must not be allowed to do that: an index three hours old
                # is full of sold listings, and letting each one wipe the
                # accumulated soft-block evidence is exactly how a host that
                # stopped answering keeps looking healthy.
                if status == 429 or status >= 500:
                    limiter.record(host, status)
                # A tombstone, so a permanently dead listing is not re-requested
                # every three hours forever. Only successes were stamped before,
                # so a 410 stayed unstamped and therefore always "due": measured
                # at 16 of 186 offers on the local index, about 128 wasted
                # requests a day against hosts that already said no.
                fresh.setdefault(job["key"], {"fetched_at": now.isoformat()})
                counts[source]["blocked"] += 1
                continue

            detail = details.extract(source, response.text)
            if not detail:
                # The measurement that matters. A 200 carrying no structured
                # data is either a soft block or an expired ad, and there is no
                # way to tell them apart from here — so both back the host off,
                # because guessing wrong in the other direction gets us banned.
                limiter.record_empty(host)
                fresh.setdefault(job["key"], {"fetched_at": now.isoformat()})
                counts[source]["empty"] += 1
                continue

            # Success reports NOTHING to the limiter, deliberately.
            #
            # This used to call record(host, 200) here, under a comment claiming
            # the ordering prevented an alternating success/empty pair from
            # cancelling out. It did not: record() zeroes the strike counter for
            # any non-error status, so one good page erased the evidence of the
            # empty one before it. Divar's soft block is intermittent, so that
            # is the shape it actually takes, and the breaker could never open.
            # A run of 40 requests with 20 soft blocks never tripped it.
            #
            # 200 is the resting state and needs no vote. Strikes accumulate
            # from record_empty alone and the breaker opens on the third,
            # whatever order the responses arrive in.
            fresh[job["key"]] = detail | {"fetched_at": now.isoformat()}
            counts[source]["extracted"] += 1

    payload = {
        "built_at": now.isoformat(),
        "count": len(fresh),
        "details": fresh,
    }
    out_path.parent.mkdir(parents=True, exist_ok=True)
    tmp = out_path.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(payload, ensure_ascii=False, separators=(",", ":")),
                   encoding="utf-8")
    tmp.replace(out_path)

    for source in SOURCES:
        c = counts[source]
        note = "" if SOURCES[source].get("enabled", True) else \
            f"  SKIPPED ({SOURCES[source].get('why')})"
        log(f"  {source:9s} attempted={c['attempted']:4d} extracted={c['extracted']:4d} "
            f"empty={c['empty']:4d} blocked={c['blocked']:4d}{note}")
    log(f"details: planned={len(jobs)} stored={len(fresh)} expired={expired} "
        f"{'budget exhausted; ' if ran_out else ''}"
        f"took={time.monotonic() - started:.0f}s -> {out_path} "
        f"({out_path.stat().st_size / 1e6:.1f} MB)")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(
        description="Fetch detail pages for the top specs' offers.")
    ap.add_argument("--index", default=DEFAULT_INDEX)
    ap.add_argument("--out", default=DEFAULT_OUT)
    ap.add_argument("--limit", type=int, default=60,
                    help="how many top specs' offers to enrich")
    ap.add_argument("--budget-seconds", type=float, default=900,
                    help="stop fetching after this long; the rest wait for the next cycle")
    ap.add_argument("--ttl-days", type=float, default=7,
                    help="do not refetch an entry younger than this")
    args = ap.parse_args()

    index_path = pathlib.Path(args.index)
    if not index_path.exists():
        log(f"no index at {index_path}; nothing to enrich")
        return 0
    try:
        return enrich(index_path, pathlib.Path(args.out), args.limit,
                      args.budget_seconds, args.ttl_days)
    except (json.JSONDecodeError, OSError) as exc:
        # Details are a passenger on the crawl, exactly like history. The index
        # is the product and something else will scream about it; this job
        # declines the cycle rather than failing it.
        log(f"index at {index_path} unreadable ({exc!r}); nothing to enrich")
        return 0


if __name__ == "__main__":
    raise SystemExit(main())

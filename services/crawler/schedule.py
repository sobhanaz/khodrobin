"""Collect from every source, rebuild the index, sleep, repeat.

Deliberately a loop and not cron. One process means one log stream, a crash
shows up as a container restart instead of a silent missing run, and the
interval is one environment variable.

Failure policy: a cycle that raises is logged and the loop continues. The API
keeps serving the last good index, so a bad night costs freshness, not uptime.
"""
from __future__ import annotations

import os
import pathlib
import subprocess
import sys
import time
from datetime import datetime, timezone

HERE = pathlib.Path(__file__).parent
INTERVAL = int(os.getenv("CRAWL_INTERVAL_SECONDS", "10800"))   # 3 hours
CITIES = os.getenv("CRAWL_CITIES", "tehran mashhad isfahan shiraz tabriz").split()
PAGES = os.getenv("CRAWL_PAGES", "30")
RAW = os.getenv("CRAWL_RAW", "/data/raw/listings.jsonl")
INDEX = os.getenv("CRAWL_INDEX", "/data/index.json")
EXPLANATIONS = os.getenv("CRAWL_EXPLANATIONS", "/data/explanations.json")
HISTORY = os.getenv("CRAWL_HISTORY", "/data/history.json")
AI_URL = os.getenv("CRAWL_AI_URL", "http://ai:8000")
WARM_LIMIT = os.getenv("CRAWL_WARM_LIMIT", "120")
WARM_BUDGET = os.getenv("CRAWL_WARM_BUDGET_SECONDS", "1800")
SEED = os.getenv("CRAWL_SEED", "/seed/listings.seed.jsonl")


def log(msg: str) -> None:
    print(f"{datetime.now(timezone.utc).isoformat(timespec='seconds')} {msg}", flush=True)


def run(args: list[str]) -> int:
    log("$ " + " ".join(args))
    return subprocess.call(args, cwd=HERE)


def bootstrap() -> None:
    """Seed the raw file on first boot so the very first index is never empty."""
    raw = pathlib.Path(RAW)
    if raw.exists() and raw.stat().st_size > 0:
        return
    seed = pathlib.Path(SEED)
    if not seed.exists():
        log(f"no seed at {seed}; the first cycle will build from a live crawl only")
        return
    raw.parent.mkdir(parents=True, exist_ok=True)
    raw.write_bytes(seed.read_bytes())
    log(f"seeded {RAW} from {SEED} ({raw.stat().st_size / 1e6:.1f} MB)")


def record_history() -> None:
    """Append this cycle's medians to the price history, right after the index
    lands. This runs before warming because warming can spend its whole
    30-minute budget, and the history point describes the index just written,
    not the index half an hour from now.
    """
    rc = run([sys.executable, "record_history.py", "--index", INDEX, "--out", HISTORY])
    if rc != 0:
        log(f"history recording exited {rc}; the chart misses a point, the cycle continues")


def warm_explanations() -> None:
    rc = run([sys.executable, "warm_explanations.py", "--index", INDEX,
              "--out", EXPLANATIONS, "--ai", AI_URL,
              "--limit", WARM_LIMIT, "--budget-seconds", WARM_BUDGET])
    if rc != 0:
        log(f"explanation warming exited {rc}; the API falls back to on-demand")


def rebuild_only() -> None:
    """Rebuild the index from raw data already on disk, without crawling.

    Run once at startup. A collection cycle takes several minutes (Divar alone
    is throttled to 4s per request), so without this a parsing change deployed
    at 22:26 would not reach users until the crawl finished — and a change that
    only affects parsing does not need new data at all.
    """
    rc = run([sys.executable, "build_index.py", "--raw", RAW, "--out", INDEX])
    if rc != 0:
        log(f"startup rebuild failed ({rc}); the API keeps serving the previous index")
        return
    record_history()
    warm_explanations()


def cycle() -> None:
    rc = run([sys.executable, "run.py", "--out", RAW, "--pages", PAGES, "--cities", *CITIES])
    if rc != 0:
        log(f"collection exited {rc}; rebuilding the index from what we already have")
    # Always rebuild: even a failed collection leaves earlier rows worth indexing.
    rc = run([sys.executable, "build_index.py", "--raw", RAW, "--out", INDEX])
    if rc != 0:
        log(f"index build failed ({rc}); the API keeps serving the previous index")
        return
    record_history()
    # Explanations take ~10s each on CPU, which is fine here and unacceptable in
    # a request. Warming the top specs after every rebuild keeps «چرا این؟»
    # instant for everything a visitor sees first.
    warm_explanations()


def main() -> int:
    log(f"crawler starting; interval={INTERVAL}s cities={' '.join(CITIES)} pages={PAGES}")
    bootstrap()
    # Publish what this build makes of the existing data before spending
    # minutes fetching more of it.
    rebuild_only()
    while True:
        started = time.monotonic()
        try:
            cycle()
        except Exception as exc:  # noqa: BLE001 - the loop must outlive any one cycle
            log(f"cycle failed: {exc!r}")
        took = time.monotonic() - started
        log(f"cycle done in {took:.0f}s; sleeping {INTERVAL}s")
        time.sleep(max(INTERVAL - took, 60))


if __name__ == "__main__":
    raise SystemExit(main())

"""Append this cycle's reliable medians to history.json, so a chart can exist.

The index is rebuilt from scratch every cycle and the previous medians are
discarded with it. Whatever price history the product ever shows starts
accumulating the day this file first runs — there is no source to backfill
from, five marketplaces show today's asking prices and nothing else.

History is a passenger on the crawl, never the driver: a corrupt or missing
file is logged and replaced, and no failure here may cost the cycle its index.

File shape (read by the Go API the same way it reads index.json):

    { "updated_at": "<iso8601>",
      "specs": { "<spec_key>": [ {"t": "<iso8601>", "m": <median>, "n": <offers>}, ... ] } }
"""
from __future__ import annotations

import argparse
import json
import pathlib
import sys
from datetime import datetime, timezone

# ~30 days at 8 cycles/day. The ceiling was measured, not guessed: a point
# serializes to ~55 bytes, so even the absurd worst case of all ~4,600 specs
# reliable for 240 straight cycles is ~61 MB — and in practice only specs
# with >= 3 offers earn points (78 of 1,172 on the seed data). Per-cycle size
# is logged below so growth is visible in the logs before it is a problem.
MAX_POINTS = 240

# A spec can vanish for weeks (seller pulls the ad, a source blocks us) and
# come back; its history must survive that. 60 days without a sighting means
# the car has genuinely left the market and the chart would only mislead.
RETIRE_AFTER_DAYS = 60

# schedule.py records after the startup rebuild as well as after every crawl,
# so a crash-looping container would otherwise append a point per restart and
# flush 30 days of unbackfillable history through the 240 cap in an afternoon.
# ponytail: fixed 10-minute floor; make it interval-aware if anyone ever runs
# a crawl cycle shorter than that.
MIN_GAP_SECONDS = 600


def log(msg: str) -> None:
    print(f"{datetime.now(timezone.utc).isoformat(timespec='seconds')} {msg}", flush=True)


def parse_t(value) -> datetime | None:
    try:
        return datetime.fromisoformat(value)
    except (TypeError, ValueError):
        return None


def load_history(path: pathlib.Path) -> dict[str, list]:
    """The stored points, or a fresh start when the file cannot be trusted.

    A corrupt file is moved aside rather than deleted: this data cannot be
    backfilled, so if a truncated write ever gets past the atomic rename the
    bytes should still be on disk for a human to salvage.
    """
    if not path.exists():
        return {}
    try:
        specs = json.loads(path.read_text(encoding="utf-8")).get("specs")
        if not isinstance(specs, dict):
            raise ValueError("no specs map")
        # Every point must be a dict, not merely every spec a list. A file
        # like {"specs": {"<key>": [1, 2]}} is valid JSON and passed this
        # check, then crashed the append path on points[-1].get("t") every
        # cycle. Because the crash lands before save(), the bad file
        # survived to crash the next cycle too.
        #
        # Raising rather than filtering is deliberate. Quietly dropping the
        # bad entries would restart that spec's history with nothing in the
        # log to say why a chart lost its past, which is the same
        # silent-repair shape this project has been bitten by five times.
        # A malformed file is moved aside intact and the operator can see it.
        for key, points in specs.items():
            if not isinstance(points, list) or not all(isinstance(pt, dict) for pt in points):
                raise ValueError(f"malformed points for {key}")
        return dict(specs)
    except (json.JSONDecodeError, OSError, ValueError) as exc:
        corpse = path.with_suffix(".json.corrupt")
        try:
            path.replace(corpse)
            moved = f"; kept the bytes at {corpse}"
        except OSError:
            moved = ""
        log(f"history at {path} unreadable ({exc!r}); starting fresh{moved} "
            "— the chart loses its past, the cycle loses nothing")
        return {}


def record(history: dict[str, list], index: dict, now: datetime) -> dict[str, int]:
    """One pass: append reliable medians, retire specs gone 60 days.

    Append-only by construction — past points are never touched, only the
    oldest fall off the 240 cap. Points carry the index's own built_at, so
    running twice against the same index is a no-op instead of a double point.
    """
    built_at = index.get("built_at") or now.isoformat()
    appended = 0
    live = set()

    for spec in index.get("specs", []):
        key = spec.get("key")
        if not key:
            continue
        live.add(key)
        # median_reliable is the card's own >= 3 offers rule; a median the card
        # refuses to quote must not be quoted by the chart either.
        if not spec.get("median_reliable") or not spec.get("median_price"):
            continue
        points = history.setdefault(key, [])
        if points:
            if points[-1].get("t") == built_at:
                continue  # this index is already recorded
            last = parse_t(points[-1].get("t"))
            if last and (now - last).total_seconds() < MIN_GAP_SECONDS:
                continue  # restart churn, not a new cycle
        points.append({"t": built_at, "m": int(spec["median_price"]),
                       "n": int(spec.get("offer_count") or 0)})
        del points[:-MAX_POINTS]
        appended += 1

    retired = 0
    for key in list(history):
        if key in live:
            continue
        # Absence is dated by the newest point, the only timestamp there is.
        # An unparseable or empty tail cannot be dated, so it retires with the
        # spec rather than surviving forever.
        points = history[key]
        last = parse_t(points[-1].get("t")) if points and isinstance(points[-1], dict) else None
        if last is None or (now - last).days >= RETIRE_AFTER_DAYS:
            del history[key]
            retired += 1

    return {"appended": appended, "retired": retired, "specs": len(history)}


def save(history: dict[str, list], path: pathlib.Path, now: datetime) -> int:
    """Write-then-rename, exactly like index.json and explanations.json: the
    API reads this path on its own schedule and must never observe half a file.
    """
    payload = {"updated_at": now.isoformat(), "specs": history}
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(payload, ensure_ascii=False, separators=(",", ":")),
                   encoding="utf-8")
    tmp.replace(path)
    return path.stat().st_size


def main() -> int:
    ap = argparse.ArgumentParser(description="Append this cycle's medians to the price history.")
    ap.add_argument("--index", default="/data/index.json")
    ap.add_argument("--out", default=None,
                    help="history file; defaults to history.json beside the index")
    args = ap.parse_args()

    index_path = pathlib.Path(args.index)
    if not index_path.exists():
        log(f"no index at {index_path}; nothing to record")
        return 0
    try:
        index = json.loads(index_path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError) as exc:
        # The index is the product and something else will scream about it;
        # history just declines to record a cycle that produced no index.
        log(f"index at {index_path} unreadable ({exc!r}); nothing to record")
        return 0

    out = pathlib.Path(args.out) if args.out else index_path.parent / "history.json"
    now = datetime.now(timezone.utc)
    history = load_history(out)
    stats = record(history, index, now)
    size = save(history, out, now)
    total_points = sum(len(v) for v in history.values())
    log(f"history: appended={stats['appended']} retired={stats['retired']} "
        f"specs={stats['specs']} points={total_points} -> {out} "
        f"({size / 1e6:.1f} MB)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

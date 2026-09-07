"""Generate explanations ahead of time, so «چرا این؟» is instant.

A 7B model on CPU takes about ten seconds per explanation. That is fine for a
background job and unacceptable in a request, so this runs after each crawl
cycle and writes the answers to a file the API serves from memory.

It is the same principle as the rest of the pipeline: the expensive work happens
on a schedule, and the request path only reads. Search is already ~1ms because
clustering and normalization happen offline; explanations now work the same way.

Only the top N specs are warmed — the ones with the most sources and offers,
which are exactly the ones a visitor sees first. The long tail still falls
through to on-demand generation, so nothing is unreachable, it is just slower
for a spec nobody has looked at yet.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import time
from datetime import datetime, timezone

import httpx

DEFAULT_INDEX = "/data/index.json"
DEFAULT_OUT = "/data/explanations.json"
DEFAULT_AI = "http://ai:8000"


def log(msg: str) -> None:
    print(f"{datetime.now(timezone.utc).isoformat(timespec='seconds')} {msg}", flush=True)


def rank_for_warming(specs: list[dict]) -> list[dict]:
    """Warm what a visitor actually sees first.

    The default result order is by source count then offer count, so warming in
    that order means the first screen is always instant even if the budget runs
    out partway through.
    """
    return sorted(
        specs,
        key=lambda s: (-s.get("source_count", 0), -s.get("offer_count", 0), s.get("key", "")),
    )


def load_existing(path: pathlib.Path) -> dict:
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8")).get("explanations", {})
    except (json.JSONDecodeError, OSError):
        return {}


def spec_fingerprint(spec: dict, guard_version: str = "") -> str:
    """What makes an explanation stale.

    Two different things can do it, and the first version of this only modelled
    one of them.

    The data: prices move between crawls, and an explanation that says a car is
    21% below market is wrong the moment the median shifts.

    The rules: when the guard gained a coherence axis it would have refused 10
    of the 93 explanations already sitting in this file — and every one kept
    being served, because their prices had not moved. `guard_version` is a hash
    of the guard's own source, read from the AI service, so editing the guard
    expires everything the old guard approved without anyone remembering to.
    """
    return (f"{spec.get('median_price')}:{spec.get('offer_count')}"
            f":{spec.get('min_price')}:{spec.get('max_price')}:{guard_version}")


def guard_version(client: httpx.Client) -> str:
    """The running guard's identity, or empty if it will not say.

    Empty degrades to the old data-only behaviour rather than discarding the
    whole cache: a health check that fails is a reason to reuse warm answers,
    not to spend the budget regenerating them against a service that is down.
    """
    try:
        return str(client.get("/health", timeout=10).json().get("guard_version", ""))
    except (httpx.HTTPError, ValueError):
        return ""


def warm(index_path: pathlib.Path, out_path: pathlib.Path, ai_url: str,
         limit: int, budget_seconds: float) -> int:
    index = json.loads(index_path.read_text(encoding="utf-8"))
    all_specs = index.get("specs", [])
    by_key = {s["key"]: s for s in all_specs}
    specs = rank_for_warming(all_specs)[:limit]
    existing = load_existing(out_path)

    fresh: dict[str, dict] = {}
    reused = generated = failed = 0
    started = time.monotonic()

    with httpx.Client(timeout=180, base_url=ai_url.rstrip("/")) as client:
        guard = guard_version(client)

        for spec in specs:
            key = spec["key"]
            fingerprint = spec_fingerprint(spec, guard)

            prior = existing.get(key)
            if prior and prior.get("fingerprint") == fingerprint:
                fresh[key] = prior
                reused += 1
                continue

            if time.monotonic() - started > budget_seconds:
                log(f"time budget reached; {len(specs) - len(fresh)} specs left unwarmed")
                break

            try:
                r = client.post("/explain", json={"spec": spec})
                r.raise_for_status()
                body = r.json()
            except httpx.HTTPError as exc:
                failed += 1
                log(f"explain failed for {key}: {exc}")
                continue

            if not body.get("text"):
                failed += 1
                continue

            fresh[key] = {
                "text": body["text"],
                "source": body.get("source", "unknown"),
                "fingerprint": fingerprint,
                "rejected_numbers": body.get("rejected_numbers") or [],
                "rejected_topics": body.get("rejected_topics") or [],
                "usage": body.get("usage"),
            }
            generated += 1

    # Anything already known is kept even if it fell outside this run's limit,
    # because throwing away good work to match an arbitrary cap would make every
    # run slower than the last. But "still valid" has to be checked, not
    # assumed: this loop used to carry every prior entry forward unconditionally
    # while its comment claimed otherwise, so 37 explanations written under an
    # older guard were immortal — the top-N window was the only place the
    # fingerprint was ever consulted, and nothing outside it could expire.
    #
    # Dropping is safe. A spec with no warm entry falls back to on-demand
    # generation, which is slower on first view and correct. On a product whose
    # claim is that its numbers can be trusted, that is the right way round.
    kept = dropped = 0
    for key, prior in existing.items():
        if key in fresh:
            continue
        spec = by_key.get(key)
        # A spec that has left the index takes its explanation with it, or the
        # file grows forever with answers about cars nobody can look up.
        if spec is None or prior.get("fingerprint") != spec_fingerprint(spec, guard):
            dropped += 1
            continue
        fresh[key] = prior
        kept += 1

    payload = {
        "built_at": datetime.now(timezone.utc).isoformat(),
        "count": len(fresh),
        "explanations": fresh,
    }
    tmp = out_path.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(payload, ensure_ascii=False, separators=(",", ":")), encoding="utf-8")
    tmp.replace(out_path)

    by_source: dict[str, int] = {}
    for v in fresh.values():
        by_source[v.get("source", "?")] = by_source.get(v.get("source", "?"), 0) + 1
    log(f"warmed: generated={generated} reused={reused} failed={failed} "
        f"carried={kept} expired={dropped} total={len(fresh)} "
        f"sources={by_source} -> {out_path}")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="Pre-generate explanations for the top specs.")
    ap.add_argument("--index", default=DEFAULT_INDEX)
    ap.add_argument("--out", default=DEFAULT_OUT)
    ap.add_argument("--ai", default=DEFAULT_AI)
    ap.add_argument("--limit", type=int, default=120,
                    help="how many top specs to keep warm")
    ap.add_argument("--budget-seconds", type=float, default=1800,
                    help="stop generating after this long; the rest stay on-demand")
    args = ap.parse_args()

    index_path = pathlib.Path(args.index)
    if not index_path.exists():
        log(f"no index at {index_path}; nothing to warm")
        return 0
    return warm(index_path, pathlib.Path(args.out), args.ai,
                args.limit, args.budget_seconds)


if __name__ == "__main__":
    raise SystemExit(main())

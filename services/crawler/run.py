"""Collect raw listings from every source into an append-only JSONL file.

Raw capture is deliberately dumb: whatever the site returned, plus an envelope
with the source, a stable source id, a content hash and a timestamp. Nothing is
interpreted here. A parser bug should cost a re-parse, never a re-crawl.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import sys
from datetime import datetime, timezone

sys.path.insert(0, str(pathlib.Path(__file__).parent))

import sources  # noqa: E402
from politeness import HostLimiter  # noqa: E402

DEFAULT_CITIES = ["tehran", "mashhad", "isfahan", "shiraz", "tabriz"]


# Fields we never use and should therefore never store.
#
# Neither is secret — a marketplace shows both on the public ad — but the seed
# snapshot is committed to a public repo, and storing per-listing identifiers
# and neighbourhood-level locations we have no use for is exactly what decision
# ۷ says this project does not do. Collect what the product needs; drop the rest
# at the door rather than carrying it forever.
DROP_FIELDS = ("vehicleIdentificationNumber",)
DROP_NESTED = {"web_info": ("district_persian",)}


def scrub(payload: dict) -> dict:
    """Remove data the product does not use before it is ever written."""
    if not isinstance(payload, dict):
        return payload
    cleaned = {k: v for k, v in payload.items() if k not in DROP_FIELDS}
    for parent, fields in DROP_NESTED.items():
        child = cleaned.get(parent)
        if isinstance(child, dict):
            cleaned[parent] = {k: v for k, v in child.items() if k not in fields}
    return cleaned


def envelope(raw: dict, mod) -> dict:
    raw = {**raw, "payload": scrub(raw["payload"])}
    payload = json.dumps(raw["payload"], ensure_ascii=False, sort_keys=True)
    return {
        "source": raw["source"],
        "source_id": mod.source_id(raw),
        "city": raw.get("city"),
        "list_url": raw["list_url"],
        "content_hash": hashlib.sha256(payload.encode()).hexdigest()[:16],
        "fetched_at": datetime.now(timezone.utc).isoformat(),
        "payload": raw["payload"],
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Collect raw car listings.")
    ap.add_argument("--out", default="../data/raw/listings.jsonl")
    ap.add_argument("--cities", nargs="*", default=DEFAULT_CITIES)
    ap.add_argument("--pages", type=int, default=2)
    ap.add_argument("--only", nargs="*", choices=sorted(sources.ALL), default=None)
    args = ap.parse_args()

    limiter = HostLimiter()
    out_path = pathlib.Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    # Re-crawling an unchanged listing is normal and must stay cheap, so skip
    # anything whose (source, id, content) we have already stored.
    seen: set[tuple[str, str, str]] = set()
    if out_path.exists():
        with out_path.open(encoding="utf-8") as fh:
            for line in fh:
                try:
                    r = json.loads(line)
                except json.JSONDecodeError:
                    continue
                seen.add((r["source"], r["source_id"], r["content_hash"]))

    wanted = args.only or list(sources.ALL)
    written = skipped = 0
    with out_path.open("a", encoding="utf-8") as fh:
        for name in wanted:
            mod = sources.ALL[name]
            rows = mod.fetch(limiter, args.cities) if name == "divar" else mod.fetch(limiter, args.pages)
            for raw in rows:
                rec = envelope(raw, mod)
                key = (rec["source"], rec["source_id"], rec["content_hash"])
                if key in seen:
                    skipped += 1
                    continue
                seen.add(key)
                fh.write(json.dumps(rec, ensure_ascii=False) + "\n")
                written += 1
            print(f"  {name:8s} fetched={len(rows):4d}", file=sys.stderr)

    print(f"written={written} skipped_unchanged={skipped} -> {out_path}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

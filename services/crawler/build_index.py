"""Turn raw listings into the index the API serves.

The Persian parsing lives here and only here. Go could re-implement folding,
alias matching and unit reconciliation, but then there would be two of them and
they would drift. Instead this emits a plain JSON artifact and the Go API loads
it — Python does the messy linguistic work, Go does the fast serving.

Output shape:

    {
      "built_at": ...,
      "stats": {...},
      "specs": [
        {
          "key": "peugeot/207/base/mt/1404/0",
          "brand": "peugeot", "brand_fa": "پژو",
          "model": "207",     "model_fa": "۲۰۷",
          "trim": null, "gearbox": "mt", "year": 1404, "km_bucket": 0,
          "median_price": 2114000000,
          "min_price": 1660000000, "max_price": 2470000000,
          "offer_count": 10, "source_count": 3,
          "offers": [ {...}, ... ]
        }, ...
      ]
    }
"""
from __future__ import annotations

import argparse
import json
import pathlib
import statistics
import sys
from collections import defaultdict
from datetime import datetime, timezone

sys.path.insert(0, str(pathlib.Path(__file__).parent))

from extract import GEARBOX, resolve, spec_key  # noqa: E402
from normalize import normalize  # noqa: E402
from plausibility import flags  # noqa: E402
from vocab import BRANDS, MODELS, TRIMS  # noqa: E402

SOURCE_FA = {"divar": "دیوار", "bama": "باما", "hamrah": "همراه‌مکانیک",
             "khodro45": "خودرو۴۵", "sheypoor": "شیپور"}
GEARBOX_FA = {"at": "اتوماتیک", "mt": "دنده‌ای", "na": None}


def build(raw_path: pathlib.Path) -> dict:
    rows = []
    with raw_path.open(encoding="utf-8") as fh:
        for line in fh:
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError:
                continue

    # Collapse re-crawls of the same ad before anything counts them.
    #
    # run.py dedupes on (source, source_id, content_hash), and content_hash
    # covers the whole payload — including fields that change on every fetch
    # for reasons that have nothing to do with the car. Bama alone carries
    # detail.rank and detail.time («لحظاتی پیش»), so an unchanged listing looks
    # new every cycle and lands as another row.
    #
    # Measured on live data: 42% of served offers were the same ad repeated,
    # which inflated offer counts and moved medians by up to 32%. The median is
    # the entire product claim, so this has to happen before grouping.
    newest: dict[tuple[str, str], dict] = {}
    for row in sorted(rows, key=lambda r: r.get("fetched_at") or ""):
        newest[(row["source"], str(row["source_id"]))] = row
    captured = len(rows)
    duplicates_collapsed = captured - len(newest)
    rows = list(newest.values())

    groups: dict[str, list[dict]] = defaultdict(list)
    resolved = unresolved = 0
    flagged = 0

    for row in rows:
        car = normalize(row)
        res = resolve(car)
        key = spec_key(car, res)
        if not key or not car.get("price_toman"):
            unresolved += 1
            continue
        resolved += 1
        found = flags(car)
        if found:
            flagged += 1
        groups[key].append({
            "source": car["source"],
            "source_fa": SOURCE_FA.get(car["source"], car["source"]),
            "title": car["title"],
            "price": car["price_toman"],
            "mileage_km": car["mileage_km"],
            "colour": car["colour"],
            "body_status": car["body_status"],
            "city": car["city"],
            "url": car["url"],
            "image": car.get("image"),
            "seen_at": car["fetched_at"],
            "flags": found,
            # Kept so the UI can show what the source actually said before we
            # touched it. Trust is the product; hiding the input undermines it.
            "raw_brand": car["brand_raw"],
            "raw_model": car["model_raw"],
            "raw_trim": car["trim_raw"],
        })

    specs = []
    for key, offers in groups.items():
        brand, model, trim, gearbox, year, bucket = key.split("/")
        prices = sorted(o["price"] for o in offers)
        median = int(statistics.median(prices))
        for o in offers:
            o["vs_median_pct"] = round(100 * (o["price"] - median) / median, 1)

        # A price far outside its own cohort is a broker's حواله, a placeholder,
        # or a different car wearing the same name — and it was ranking first,
        # because "cheapest" has no opinion about whether a price is real.
        # This check needs the cluster, so it cannot live in plausibility.py
        # where every listing is judged alone.
        if len(prices) >= 3:
            for o in offers:
                ratio = o["price"] / median if median else 1
                if ratio < 0.4 or ratio > 2.5:
                    o.setdefault("flags", []).append({
                        "code": "price_outlier",
                        "message": (
                            f"قیمت این آگهی {o['price']:,} تومان است، "
                            f"در حالی که میانه‌ی همین خودرو {median:,} تومان است؛ "
                            "ممکن است حواله، پیش‌فروش یا قیمت غیرواقعی باشد."
                        ),
                    })
        brand_fa = BRANDS.get(brand, (brand, ()))[0]
        model_fa = MODELS.get((brand, model), (model, ()))[0]

        # Say the car's name once.
        #
        # A base variant repeats the brand as its own name («کوییک کوییک»), and a
        # sub-model carries it as a prefix — «دنا» + «دنا پلاس» + trim «plus»
        # rendered as «دنا دنا پلاس · plus», which is three names for one car.
        if model_fa == brand_fa:
            model_fa = ""
        elif brand_fa and model_fa.startswith(brand_fa + " "):
            model_fa = model_fa[len(brand_fa) + 1:]

        # Drop a trim the model name already states. The trim slug is Latin
        # («plus») and the model name is Persian («پلاس»), so a string compare
        # misses it — the alias table is what knows they are the same word.
        trim_display = None if trim == "base" else trim
        if trim_display:
            for alias in TRIMS.get(trim_display, ()):
                if alias and alias in model_fa:
                    trim_display = None
                    break

        clean_prices = sorted(o["price"] for o in offers if not o.get("flags")) or prices
        image = next((o["image"] for o in sorted(offers, key=lambda o: o["price"]) if o.get("image")), None)

        specs.append({
            "key": key,
            "image": image,
            "brand": brand,
            "brand_fa": brand_fa,
            "model": model,
            "model_fa": model_fa,
            "trim": trim_display,
            "gearbox": gearbox,
            "gearbox_fa": GEARBOX_FA.get(gearbox),
            "year": int(year),
            "km_bucket": None if bucket == "na" else int(bucket),
            "offer_count": len(offers),
            "source_count": len({o["source"] for o in offers}),
            # Below three offers a "market median" is a fiction — with two, it
            # is the mean of the only two asking prices, and no seller is asking
            # it. Consumed by services/ai/app/explain.py, which then says
            # «ارزان‌ترین آگهی از N آگهی» instead of «X٪ پایین‌تر از میانه»; the
            # Vue cards do not branch on it and still print the median. The
            # earlier version of this comment claimed the UI showed a range
            # instead, which was never true anywhere in the front end.
            "median_price": median,
            "median_reliable": len(prices) >= 3,
            # The range shown describes offers a buyer can actually compare.
            # One flagged allocation certificate at 220,000,000 stretched a
            # Dena Plus card from there to 3,570,000,000 — a bar that was mostly
            # empty space describing a car nobody was selling.
            "min_price": clean_prices[0],
            "max_price": clean_prices[-1],
            "flag_count": sum(len(o["flags"]) for o in offers),
            # Cheapest first, but a flagged offer sinks below every clean one.
            #
            # A «حواله» listing at 82% under its cohort is the cheapest number
            # and the worst answer: it is an allocation certificate, not a car.
            # Detecting it and still showing it first would make the flag
            # decorative.
            "offers": sorted(offers, key=lambda o: (bool(o.get("flags")), o["price"])),
        })

    # Biggest and most cross-source first: those are the ones worth looking at.
    specs.sort(key=lambda s: (-s["source_count"], -s["offer_count"], s["key"]))

    return {
        "built_at": datetime.now(timezone.utc).isoformat(),
        "stats": {
            # Three different numbers people conflate: rows we captured, rows
            # left after collapsing re-crawls, and rows that resolved to a
            # priced spec. Reporting only one of them hides the other two.
            "listings_captured": captured,
            "listings": len(rows),
            "indexed": resolved,
            "unresolved": unresolved,
            "duplicates_collapsed": duplicates_collapsed,
            "resolved_pct": round(100 * resolved / max(len(rows), 1), 1),
            "specs": len(specs),
            "multi_source_specs": sum(1 for s in specs if s["source_count"] > 1),
            "flagged_offers": flagged,
            "sources": sorted({r["source"] for r in rows}),
        },
        # The Go API parses Persian queries against these same aliases. Exporting
        # them keeps one source of truth: adding «سراتو» to vocab.py teaches both
        # the crawler and the search box at once.
        "vocab": {
            "brands": [
                {"slug": slug, "fa": display, "aliases": list(aliases)}
                for slug, (display, aliases) in BRANDS.items()
            ],
            "models": [
                {"brand": b, "slug": slug, "fa": display, "aliases": list(aliases)}
                for (b, slug), (display, aliases) in MODELS.items()
            ],
        },
        "specs": specs,
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Build the served index from raw listings.")
    ap.add_argument("--raw", default="../data/raw/listings.jsonl")
    ap.add_argument("--out", default="../api/data/index.json")
    args = ap.parse_args()

    raw = pathlib.Path(args.raw)
    if not raw.exists():
        raw = pathlib.Path("../data/seed/listings.seed.jsonl")
        print(f"no raw file; falling back to seed: {raw}", file=sys.stderr)

    index = build(raw)
    out = pathlib.Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)

    # Write-then-rename. The API watches this path's mtime and reloads on
    # change, so it must never observe a partially written file. rename() is
    # atomic within a filesystem.
    tmp = out.with_suffix(".json.tmp")
    tmp.write_text(json.dumps(index, ensure_ascii=False, separators=(",", ":")), encoding="utf-8")
    tmp.replace(out)

    s = index["stats"]
    print(f"listings={s['listings']} indexed={s['indexed']} ({s['resolved_pct']}%) "
          f"specs={s['specs']} multi_source={s['multi_source_specs']} "
          f"flagged={s['flagged_offers']} -> {out} "
          f"({out.stat().st_size / 1e6:.1f} MB)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

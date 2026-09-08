"""Turn five incompatible source shapes into one canonical car record.

The messiness this exists to absorb, all of it found in real captured data
rather than imagined:

* **Price units differ per source.** Divar and Sheypoor quote rials; Bama,
  Hamrah-Mechanic and Khodro45 quote tomans. Verified by comparing the same
  model and year across sources: a 1385 Pride is 3,100,000,000 on Divar and
  320,000,000 on Bama — the same ~315M tomans. Getting this wrong inflates a
  price tenfold. It is not a 3-2 split you can guess from the field names
  either: Sheypoor labels its price ``priceCurrency: "IRR"`` and Bama labels
  nothing, and both labels are equally uninformative until you price a car.
* **Year systems are mixed inside a single source.** A domestic Pride is listed
  as 1398 (Jalali) while an imported Toyota in the same feed is 2025
  (Gregorian). We store both and key identity on the Jalali year.
* **Mileage is sometimes prose.** «صفر کیلومتر» means 0, not missing.
* **Damage vocabulary differs.** Divar says `half-paint`, Bama says «بدون رنگ».
* **Junk prices exist.** Divar returns placeholder values as low as 10,000
  rials for ads with no real price; below a floor we treat price as absent
  rather than letting it poison a market median.
"""
from __future__ import annotations

import re
from typing import Any

# A price below this (in tomans) is a placeholder, not a car.
MIN_PLAUSIBLE_TOMAN = 10_000_000
JALALI_GREGORIAN_OFFSET = 621

FA_DIGITS = str.maketrans("۰۱۲۳۴۵۶۷۸۹٠١٢٣٤٥٦٧٨٩", "01234567890123456789")

# Divar's actual knownVehicleDamages vocabulary, taken from live payloads.
# The original map guessed at these keys and matched almost none of them, so
# body condition was discarded for 89% of Divar listings — including
# «accidental», which is the one a buyer most needs to see.
BODY_STATUS = {
    "intact": "بدون رنگ",
    "some-scratches": "بدون رنگ",
    "paintless-dent-removal": "بدون رنگ",
    "some-paint": "رنگ‌شدگی",
    "accidental": "تصادفی", "تصادفی": "تصادفی",
    "none": "بدون رنگ", "بدون رنگ": "بدون رنگ", "سالم": "بدون رنگ",
    "half-paint": "رنگ‌شدگی", "رنگ‌شدگی": "رنگ‌شدگی", "دوررنگ": "رنگ‌شدگی",
    "full-paint": "تمام‌رنگ", "تمام رنگ": "تمام‌رنگ",
    "replaced": "تعویض", "تعویض شده": "تعویض", "تعویض": "تعویض",
    "رنگ": "رنگ‌شدگی",  # last resort; longer keys above win
}
TRANSMISSION = {
    "manual": "دنده‌ای", "دنده ای": "دنده‌ای", "دنده‌ای": "دنده‌ای",
    "automatic": "اتوماتیک", "اتومات": "اتوماتیک", "اتوماتیک": "اتوماتیک",
}


def digits(text: str | None) -> str:
    return (text or "").translate(FA_DIGITS)


def to_int(text: Any) -> int | None:
    if isinstance(text, (int, float)):
        return int(text)
    cleaned = re.sub(r"[^0-9]", "", digits(str(text or "")))
    return int(cleaned) if cleaned else None


def split_year(value: Any) -> tuple[int | None, int | None]:
    """Return (jalali, gregorian) from a year that may be in either system."""
    year = to_int(value)
    if year is None:
        return None, None
    if 1300 <= year <= 1500:
        return year, year + JALALI_GREGORIAN_OFFSET
    if 1900 <= year <= 2100:
        return year - JALALI_GREGORIAN_OFFSET, year
    return None, None


def price_toman(value: Any, *, unit: str) -> int | None:
    raw = to_int(value)
    if raw is None or raw <= 0:
        return None
    toman = raw // 10 if unit == "rial" else raw
    return toman if toman >= MIN_PLAUSIBLE_TOMAN else None


def mileage_km(value: Any) -> int | None:
    text = digits(str(value or ""))
    if "صفر" in text:
        return 0
    return to_int(text)


def _lookup(table: dict[str, str], value: Any) -> str | None:
    raw = str(value or "").strip()
    if not raw:
        return None
    hit = table.get(raw.lower()) or table.get(raw)
    if hit:
        return hit
    # Sources write free text around the keyword — Bama says «گلگیر تعویض»,
    # «دو لکه رنگ» and similar. Fall back to substring matching, longest key
    # first so «تمام رنگ» wins over «رنگ».
    for key in sorted(table, key=len, reverse=True):
        if key in raw.lower() or key in raw:
            return table[key]
    return None


def from_divar(p: dict) -> dict:
    jalali, greg = split_year(p.get("productionDate") or p.get("vehicleModelDate"))
    info = p.get("web_info") or {}
    return {
        "title": p.get("name"),
        "brand_raw": (p.get("brand") or {}).get("name"),
        "model_raw": p.get("model"),
        "trim_raw": None,
        "year_jalali": jalali,
        "year_gregorian": greg,
        "mileage_km": mileage_km((p.get("mileageFromOdometer") or {}).get("value")),
        "price_toman": price_toman((p.get("offers") or {}).get("price"), unit="rial"),
        "colour": p.get("color"),
        "body_status": _lookup(BODY_STATUS, p.get("knownVehicleDamages")),
        "transmission": _lookup(TRANSMISSION, p.get("vehicleTransmission")),
        "city": info.get("city_persian"),
        "url": p.get("url"),
        "image": p.get("image"),
    }


def from_bama(p: dict) -> dict:
    d = p.get("detail") or {}
    jalali, greg = split_year(d.get("year"))
    return {
        "title": d.get("title"),
        "brand_raw": d.get("brand_fa") or d.get("brand"),
        "model_raw": d.get("brand"),
        "trim_raw": d.get("trim"),
        "year_jalali": jalali,
        "year_gregorian": greg,
        "mileage_km": mileage_km(d.get("mileage")),
        # Bama quotes three price types. For an instalment listing the `price`
        # field is the financed total, which is not comparable to a cash asking
        # price and was dragging cluster medians upward.
        "price_toman": (
            None if (p.get("price") or {}).get("type") == "installment"
            else price_toman((p.get("price") or {}).get("price"), unit="toman")
        ),
        "colour": d.get("body_color"),
        "body_status": _lookup(BODY_STATUS, d.get("body_status")),
        "transmission": _lookup(TRANSMISSION, d.get("transmission")),
        "city": (d.get("location") or "").split("/")[0].strip() or None,
        "url": "https://bama.ir" + (d.get("url") or ""),
        "image": d.get("image"),
    }


def from_hamrah(p: dict) -> dict:
    jalali, greg = split_year(p.get("carYear"))
    return {
        "title": p.get("carNamePersian"),
        "brand_raw": p.get("brandEnglishName"),
        "model_raw": p.get("modelEnglishName"),
        "trim_raw": p.get("carTypeName"),
        "year_jalali": jalali,
        "year_gregorian": greg,
        "mileage_km": mileage_km(p.get("km")),
        "price_toman": price_toman(p.get("price"), unit="toman"),
        "colour": p.get("carColorName"),
        "body_status": None,
        "transmission": _lookup(TRANSMISSION, p.get("gearBoxPersian")),
        "city": (p.get("carLocation") or "").split("،")[0].strip() or None,
        "url": "https://www.hamrah-mechanic.com" + (p.get("exhibitionDetailUrl") or ""),
        "image": p.get("imageUrl"),
    }


def from_khodro45(p: dict) -> dict:
    props = p.get("car_properties") or {}
    specs = p.get("car_specifications") or {}
    brand = props.get("brand") or {}
    model = props.get("model") or {}
    jalali, greg = split_year(props.get("year"))
    return {
        "title": f"{brand.get('title', '')} {model.get('title', '')}".strip() or None,
        "brand_raw": brand.get("title_en") or brand.get("title"),
        "model_raw": model.get("title_en") or model.get("title"),
        "trim_raw": props.get("trim"),
        "year_jalali": jalali,
        "year_gregorian": greg,
        "mileage_km": to_int(specs.get("klm")),
        "price_toman": price_toman(p.get("price"), unit="toman"),
        "colour": None,
        "body_status": None,
        # Khodro45 files the gearbox inside the trim field — «اتوماتیک»,
        # «دنده‌ای», «اتوماتیک توربو». Left unparsed, every Khodro45 spec keys
        # on gearbox="na" and can never join another source's cluster, which is
        # exactly what happened when this source was first added.
        "transmission": _lookup(TRANSMISSION, props.get("trim")),
        "city": (p.get("city") or {}).get("title"),
        "url": f"https://khodro45.com/used-car/{p.get('slug')}/" if p.get("slug") else "https://khodro45.com/used-car/",
        "image": (p.get("image") or {}).get("url"),
        # Unique to this source: whether the odometer was verified. None means
        # "not checked", which is different from False ("checked, did not match").
        "km_verified": specs.get("is_klm_matched"),
    }


def from_sheypoor(p: dict) -> dict:
    """Sheypoor's schema.org Vehicle, which looks like Divar's and is not.

    Three things differ from the Divar payload it superficially resembles, and
    every one of them fails silently rather than loudly:

    * The price key is ``offers.Price`` with a **capital P** — non-standard, and
      schema.org's own ``offers.price`` is simply absent. Read the lowercase key
      and every Sheypoor listing arrives priceless, which build_index drops. The
      source would appear to integrate and contribute nothing.
    * ``image`` is a **list** of ImageObject, not a string. Copied straight
      through it hands the front end a list where it expects a URL.
    * ``vehicleModelDate`` carries whichever year system the seller used —
      measured at 115 Jalali to 5 Gregorian in one 120-ad sample — while on
      Divar that same field name is always Gregorian and Jalali lives in
      ``productionDate``. Same field, same schema, different meaning per site.

    Units are rials. Not because the payload says ``priceCurrency: "IRR"`` — a
    label costs a site nothing to get wrong — but because the prices only make
    sense at a tenth: a 1384 Pride quotes 4,100,000,000 here against a live
    1385 Pride at 310,000,000 tomans on Bama. Divided by ten those are the same
    car at the same money; read as tomans, Sheypoor is selling a 1384 Pride for
    four billion tomans.

    There is no model and no trim field, so identity is carried entirely by the
    seller's free-text title and extract.resolve has to earn it from there.
    """
    jalali, greg = split_year(p.get("vehicleModelDate"))
    offers = p.get("offers") or {}
    images = p.get("image") or []
    return {
        "title": p.get("name"),
        "brand_raw": (p.get("brand") or {}).get("name"),
        "model_raw": None,
        "trim_raw": None,
        "year_jalali": jalali,
        "year_gregorian": greg,
        "mileage_km": mileage_km((p.get("mileageFromOdometer") or {}).get("value")),
        "price_toman": price_toman(offers.get("Price"), unit="rial"),
        "colour": p.get("color"),
        "body_status": _lookup(BODY_STATUS, p.get("knownVehicleDamages")),
        "transmission": _lookup(TRANSMISSION, p.get("vehicleTransmission")),
        # Attached by the source module: the ad itself has no location, because
        # on Sheypoor the city is a property of the page you found it on.
        "city": p.get("sheypoor_city_fa"),
        "url": p.get("url"),
        "image": (images[0] or {}).get("contentUrl") if isinstance(images, list) and images else None,
    }


ADAPTERS = {
    "divar": from_divar,
    "bama": from_bama,
    "hamrah": from_hamrah,
    "khodro45": from_khodro45,
    "sheypoor": from_sheypoor,
}


def normalize(record: dict) -> dict:
    """Map one raw envelope to the canonical shape."""
    canonical = ADAPTERS[record["source"]](record["payload"])
    canonical |= {
        "source": record["source"],
        "source_id": record["source_id"],
        "fetched_at": record["fetched_at"],
    }
    return canonical

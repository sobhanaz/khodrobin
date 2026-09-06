"""Turn three incompatible source shapes into one canonical car record.

The messiness this exists to absorb, all of it found in real captured data
rather than imagined:

* **Price units differ per source.** Divar quotes rials; Bama and
  Hamrah-Mechanic quote tomans. Verified by comparing the same model and year
  across sources: a 1385 Pride is 3,100,000,000 on Divar and 320,000,000 on
  Bama — the same ~315M tomans. Getting this wrong inflates a price tenfold.
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

BODY_STATUS = {
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
        "price_toman": price_toman((p.get("price") or {}).get("price"), unit="toman"),
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


ADAPTERS = {
    "divar": from_divar,
    "bama": from_bama,
    "hamrah": from_hamrah,
    "khodro45": from_khodro45,
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

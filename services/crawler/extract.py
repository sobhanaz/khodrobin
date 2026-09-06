"""Resolve a listing to a canonical (brand, model, trim) — rules first.

This is the step that makes cross-source comparison possible at all. The three
feeds describe the same car in three vocabularies:

    Divar   brand="Peugeot"   model="Peugeot 206 5"
    Bama    brand_fa="پژو"    title="پژو، 206 SD"      trim="تیپ ۵"
    Hamrah  brand="peugeot"   model="206"              trim="تیپ ۵ اتوماتیک"

All three must land on ``peugeot / 206-sd / type-5`` or the price of a 206 gets
averaged with the price of a 405.

Rules run first because most Iranian listings are formulaic and a dictionary
resolves them for nothing. Whatever the rules cannot resolve confidently is
handed to a model — that split is measured (`Resolution.method`) and reported,
because paying for inference on «پژو ۲۰۶ تیپ ۵» would be waste.
"""
from __future__ import annotations

import re
import unicodedata
from dataclasses import dataclass

from vocab import BRANDS, MODELS, TRIMS

FA_DIGITS = str.maketrans("۰۱۲۳۴۵۶۷۸۹٠١٢٣٤٥٦٧٨٩", "01234567890123456789")
# Arabic forms that Persian keyboards emit interchangeably.
ARABIC_FOLD = str.maketrans({"ي": "ی", "ك": "ک", "ة": "ه", "أ": "ا", "إ": "ا", "آ": "ا"})
ZWNJ = "‌"


def fold(text: str | None) -> str:
    """Normalize Persian text for matching: digits, Arabic forms, ZWNJ, spacing.

    Matching text, not displaying it — the output is deliberately lossy.
    """
    if not text:
        return ""
    s = unicodedata.normalize("NFKC", text)
    s = s.translate(FA_DIGITS).translate(ARABIC_FOLD)
    s = s.replace(ZWNJ, " ")
    s = re.sub(r"[،,/\\|_\-–—]+", " ", s)
    s = re.sub(r"\s+", " ", s)
    return s.strip().lower()


@dataclass
class Resolution:
    brand: str | None = None
    model: str | None = None
    trim: str | None = None
    confidence: float = 0.0
    method: str = "unresolved"      # rules | llm | unresolved
    evidence: str = ""              # the folded text the decision was made on


def _match_brand(text: str) -> tuple[str | None, str]:
    """Longest alias wins, so «بی وای دی» beats a stray «دی»."""
    best: tuple[str | None, str] = (None, "")
    for slug, (_display, aliases) in BRANDS.items():
        for alias in aliases:
            a = fold(alias)
            if a and a in text and len(a) > len(best[1]):
                best = (slug, a)
    return best


def _match_model(brand: str, text: str) -> tuple[str | None, str]:
    best: tuple[str | None, str] = (None, "")
    for (b, slug), (_display, aliases) in MODELS.items():
        if b != brand:
            continue
        for alias in aliases:
            a = fold(alias)
            # Bare numeric aliases ("206") must not match inside "2061" or a
            # price, so require a word boundary for them.
            if a.isdigit():
                hit = re.search(rf"(?<!\d){re.escape(a)}(?!\d)", text) is not None
            else:
                hit = a in text
            if hit and len(a) > len(best[1]):
                best = (slug, a)
    return best


def _match_trim(text: str) -> str | None:
    best: tuple[str | None, str] = (None, "")
    for slug, aliases in TRIMS.items():
        for alias in aliases:
            a = fold(alias)
            if a and a in text and len(a) > len(best[1]):
                best = (slug, a)
    return best[0]


def resolve(car: dict) -> Resolution:
    """Resolve one canonical car record produced by ``normalize.normalize``."""
    # Order matters: the structured fields are more trustworthy than the title,
    # which is free text a seller wrote.
    text = fold(" ".join(str(x) for x in (
        car.get("brand_raw"), car.get("model_raw"), car.get("trim_raw"), car.get("title")
    ) if x))
    if not text:
        return Resolution(evidence="")

    brand, brand_alias = _match_brand(text)
    if not brand:
        return Resolution(method="unresolved", evidence=text[:120])

    model, model_alias = _match_model(brand, text)
    trim = _match_trim(text)

    # Confidence reflects how much of the key we actually pinned down. A brand
    # alone is not enough to compare prices, so it stays below the threshold
    # that would let it skip the model.
    confidence = 0.45
    if model:
        confidence = 0.85
        if trim:
            confidence = 0.95
    return Resolution(
        brand=brand, model=model, trim=trim,
        confidence=confidence,
        method="rules" if model else "unresolved",
        evidence=text[:120],
    )


# Anything at or above this is accepted from the rules; below it, the record is
# a candidate for the model.
RULES_THRESHOLD = 0.85


def needs_model(res: Resolution) -> bool:
    return res.confidence < RULES_THRESHOLD


GEARBOX = {"اتوماتیک": "at", "دنده\u200cای": "mt"}


def spec_key(car: dict, res: Resolution, *, km_bucket: int = 25_000) -> str | None:
    """The clustering key: what makes two listings the same *offer* to a buyer.

    Mileage is bucketed because nobody shops for "exactly 118,400 km" — they
    shop for a band. 25,000 km is roughly two years of Tehran driving.

    Transmission is part of the key because it moves the price materially: a
    1404 Peugeot 207 automatic sits ~30% above the manual in the captured data.
    """
    if not (res.brand and res.model and car.get("year_jalali")):
        return None
    bucket = (car["mileage_km"] // km_bucket) if car.get("mileage_km") is not None else "na"
    gearbox = GEARBOX.get(car.get("transmission") or "", "na")
    return f"{res.brand}/{res.model}/{res.trim or 'base'}/{gearbox}/{car['year_jalali']}/{bucket}"

"""Per-listing detail extraction: the fields a search-results page never carries.

The list feeds give five sources' worth of price, mileage, year and title. What
they do not give is the half of a listing a buyer actually reads before calling
anyone: the seller's own paragraph, the colour, the damage note, the photos, and
on Hamrah-Mechanic a full mechanical inspection. Those live only on the detail
page, one request per listing, which is why this module exists separately from
``sources/`` and is warmed for a shortlist rather than the whole index.

Every extractor here is a pure function of the fetched text. Nothing here opens
a socket; ``enrich_details.py`` does the fetching, the politeness and the
budget. That split is what lets the tests run against saved pages.

THE PRICE RULE, and it is not a style preference.

    An extractor in this module must never return a price, a mileage, a year,
    or anything else the index already holds.

The reason, measured on two real Divar ads during this module's own build:

    ad gadOXBRG   list page  "price": 21300000000  "priceCurrency": "IRR"
                  detail page "price":  2130000000  "priceCurrency": "IRR"
    ad gaVG2nPJ   list page  "price": 19200000000  "priceCurrency": "IRR"
                  detail page "price":  1900000000  "priceCurrency": "IRR"

Same site, same schema.org field, same currency label, one order of magnitude
apart: Divar's list page publishes rials and its detail page publishes tomans,
and both call it IRR. Bama then labels its tomans IRR as well. There is no field
name anywhere in this data that tells you the unit — the list pipeline resolved
it per source by pricing one real car across sources, and that resolution is the
only one anybody should trust. A detail fetch that "improved" a price would move
every median by a factor of ten while every field name still looked right.

``_detail`` is therefore the only constructor: it emits exactly the contract's
keys and nothing else, so adding a price would mean deliberately editing the one
function this docstring is about.
"""
from __future__ import annotations

import json
import re
from typing import Any

from normalize import BODY_STATUS, FA_DIGITS

# The contract's shape. One structure for five sources, so the product renders
# a detail panel once instead of five times.
KEYS = ("description", "color", "damages", "condition", "fuel", "body",
        "engine", "images", "inspection", "extras")

MAX_IMAGES = 8

_LD = re.compile(r'<script[^>]*application/ld\+json[^>]*>(.*?)</script>', re.S)
_NEXT = re.compile(r'<script id="__NEXT_DATA__"[^>]*>(.*?)</script>', re.S)

# Divar publishes each photo three times — /post/ as jpg, /webp_post/ as the
# full-size webp, /webp_thumbnail/ as the card thumbnail — under three different
# path tokens but one shared filename. Matching the filename is what stops a
# gallery of three cars' photos coming back as eight near-duplicates.
_DIVAR_PHOTO = re.compile(
    r'https://[a-z0-9.-]*divarcdn\.com/static/photo/[^\s"\'\\<>]+?\.(?:webp|jpe?g|png)')

# Iranian phone numbers, in a field this project once deleted wholesale.
#
# run.py drops `description` from every raw payload precisely because 34 of the
# 1,891 Sheypoor and Bama descriptions it sampled carried a seller's mobile
# number, and the seed snapshot is a public repo. This module brings the field
# back for the product's sake, so it has to carry that decision with it: the
# file below is served by a public API, and a phone number in it is the same
# leak in a different location.
#
# Anchored on the leading 0 or +98 and a 10-to-12-digit run, because that is
# what separates «۰۹۱۲۵۲۳۷۰۱۱» from «۲,۱۰۰,۰۰۰,۰۰۰» — a price never starts with
# a zero, and the separators a price uses (commas) are not in the class.
_PHONE = re.compile(r"(?<!\d)(?:\+?98|0)[\s.\-]?\d(?:[\s.\-]?\d){8,10}(?!\d)")
_PHONE_REDACTED = "[شماره تماس حذف شد]"

# Bama's ld+json description is an SEO sentence with the seller's own words
# glued on after the city: «خرید خودرو پژو 207 … در شهر تهران، پژو 207tu5p …».
# The product wants the seller's half. Anchored on the template rather than on
# "text after the first comma", so a description that is genuinely one sentence
# with a comma in it survives intact.
_BAMA_SEO = re.compile(r"^خرید خودرو .{0,120}? در شهر [^،]{1,30}،\s*")

# Hamrah-Mechanic has no seller description field at all. What its
# `descriptionPart` carries is a fixed platform sentence about the dealer's own
# damage cover — byte-identical on 5 of the 6 listings sampled while building
# this, with the sixth appending the seller's one real note after it. Left
# whole, `description` would be the same paragraph on every Hamrah card in the
# product, which reads as a bug and buries the one line that is not boilerplate.
_HAMRAH_BOILERPLATE = re.compile(
    r"^این خودرو توسط همراه مکانیک کارشناسی شده است[^.]*\.\s*")

EN_TO_FA = str.maketrans("0123456789", "۰۱۲۳۴۵۶۷۸۹")

# schema.org's four itemCondition values. Deliberately «نو» and not «صفر
# کیلومتر»: the second one asserts an odometer reading, and plausibility.py
# already flags listings that claim zero kilometres and contradict themselves.
# A condition word must not become a mileage claim on the way to the screen.
CONDITION_FA = {
    "newcondition": "نو",
    "usedcondition": "کارکرده",
    "refurbishedcondition": "بازسازی‌شده",
    "damagedcondition": "آسیب‌دیده",
}

# Bama and Khodro45 publish body types in English into an entirely Persian page.
BODY_FA = {
    "hatchback": "هاچبک", "sedan": "سدان", "suv": "شاسی‌بلند",
    "crossover": "کراس‌اوور", "wagon": "استیشن", "pickup": "وانت",
    "van": "ون", "coupe": "کوپه", "convertible": "کروک",
}


def _clean(value: Any) -> str | None:
    """Whitespace-normalise a seller's text without flattening it.

    Runs of spaces collapse and blank lines collapse to one, but line breaks
    survive: sellers write one feature per line and a description folded to a
    single paragraph reads like a run-on sentence in an RTL column.
    """
    if not isinstance(value, str):
        return None
    lines = [re.sub(r"[^\S\n]+", " ", ln).strip() for ln in value.split("\n")]
    text = re.sub(r"\n{3,}", "\n\n", "\n".join(lines)).strip()
    if not text:
        return None
    # A placeholder is not content. Khodro45 sellers type "." to satisfy a
    # required field, and a detail panel that renders «توضیحات: .» is worse
    # than one that renders nothing.
    if len(text) < 2 or not re.search(r"[\w\u0600-\u06FF]", text):
        return None
    return _redact(text)


def _redact(text: str) -> str:
    """Strip phone numbers, matching on ASCII digits and cutting the original.

    ``FA_DIGITS`` maps one character to one character, so translating for the
    match leaves every offset valid against the untranslated string. That is the
    whole trick: detect in ASCII, slice in Persian, and the seller's own digits
    survive everywhere they are not a phone number.
    """
    probe = text.translate(FA_DIGITS)
    out: list[str] = []
    last = 0
    for m in _PHONE.finditer(probe):
        out.append(text[last:m.start()])
        out.append(_PHONE_REDACTED)
        last = m.end()
    out.append(text[last:])
    return "".join(out)


def _fa_num(value: Any) -> str | None:
    """A number as Persian digits, with a trailing .0 dropped."""
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        return None
    text = f"{value:g}"
    return text.translate(EN_TO_FA)


def _condition(value: Any) -> str | None:
    return CONDITION_FA.get(str(value or "").rsplit("/", 1)[-1].lower())


def _body(value: Any) -> str | None:
    raw = str(value or "").strip()
    if not raw:
        return None
    return BODY_FA.get(raw.lower(), raw)


def _images(urls: list[Any]) -> list[str]:
    """Absolute URLs only, deduped by photo, capped.

    Deduped on the filename *stem*, not the whole URL and not the filename:
    Divar serves one photo as ``…/post/<token>/<uuid>.jpg`` and
    ``…/webp_post/<other-token>/<uuid>.webp``, which differ in both the path and
    the extension while being the same picture. A gallery of eight that is
    really three cars photographed twice each is worse than a gallery of three.
    """
    out: list[str] = []
    seen: set[str] = set()
    for url in urls:
        if not isinstance(url, str) or not url.startswith("http"):
            continue
        stem = url.split("?", 1)[0].rsplit("/", 1)[-1].rsplit(".", 1)[0]
        if stem in seen:
            continue
        seen.add(stem)
        out.append(url)
        if len(out) >= MAX_IMAGES:
            break
    return out


# Words that mean the extras map has started republishing the index. extras is
# free-form by design, which makes it the one place a price could get back in
# without touching a named field — so the guard lives here, once, where every
# source's leftovers pass through.
_FORBIDDEN = ("قیمت", "price", "کارکرد", "mileage", "کیلومتر")


def _detail(*, description=None, color=None, damages=None, condition=None,
            fuel=None, body=None, engine=None, images=None, inspection=None,
            extras=None) -> dict | None:
    """Build the one shape, or None when the page carried nothing worth storing.

    Keyword-only and exhaustive: this is the sole constructor, so the set of
    fields a detail can ever contain is the set of parameters here. A price
    cannot arrive by accident, only by someone adding a parameter for it.
    """
    extras = {k: v for k, v in (extras or {}).items()
              if isinstance(v, str) and v.strip()
              and not any(word in k.lower() for word in _FORBIDDEN)}
    detail = {
        "description": _clean(description),
        "color": (str(color).strip() or None) if color else None,
        "damages": (str(damages).strip() or None) if damages else None,
        "condition": condition,
        "fuel": (str(fuel).strip() or None) if fuel else None,
        "body": body,
        "engine": engine,
        "images": images or [],
        "inspection": inspection,
        "extras": extras,
    }
    # Divar answers a soft block with HTTP 200 and no JSON-LD, and a dead
    # listing answers with a shell page. Both parse fine and yield a record of
    # ten nulls, which would then sit in the file for seven days looking like a
    # successful fetch. Nothing extracted is not a detail.
    if not any(v for k, v in detail.items() if k != "extras") and not extras:
        return None
    return detail


def _ld_blocks(html: str) -> list[Any]:
    out = []
    for block in _LD.findall(html):
        try:
            out.append(json.loads(block))
        except json.JSONDecodeError:
            continue
    return out


def _next_data(html: str) -> dict:
    m = _NEXT.search(html)
    if not m:
        return {}
    try:
        data = json.loads(m.group(1))
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def _types(node: Any) -> list[str]:
    """@type, which schema.org allows to be a string or a list of them."""
    if not isinstance(node, dict):
        return []
    t = node.get("@type")
    return [t] if isinstance(t, str) else [x for x in (t or []) if isinstance(x, str)]


def from_divar(html: str) -> dict | None:
    """Divar's detail page: the same [Car] JSON-LD array the list page carries.

    Richest damage vocabulary of the five — `intact`, `some-scratches`,
    `paintless-dent-removal`, `accidental` — mapped through normalize's own
    table so the panel shows Persian rather than an English enum.
    """
    car = None
    for data in _ld_blocks(html):
        if isinstance(data, list) and data and "Car" in _types(data[0]):
            car = data[0]
            break
        if "Car" in _types(data):
            car = data
            break
    if car is None:
        return None

    damages = car.get("knownVehicleDamages")
    # Full-size variants first so they win the dedupe, but thumbnails still come
    # through: the server-rendered HTML carries the whole gallery as thumbnails
    # and only the first few as full-size, so excluding them outright dropped
    # half the photos on every ad measured.
    photos = sorted(dict.fromkeys(_DIVAR_PHOTO.findall(html)),
                    key=lambda u: "/webp_thumbnail/" in u)
    return _detail(
        description=car.get("description"),
        color=car.get("color"),
        damages=BODY_STATUS.get(str(damages or "").lower()) or damages,
        condition=_condition(car.get("itemCondition")),
        images=_images([car.get("image"), *photos]),
    )


def from_bama(html: str) -> dict | None:
    """Bama's ld+json is @type ["Product","Car"] and the only one with a spec sheet."""
    car = None
    graph_images: list[Any] = []
    for data in _ld_blocks(html):
        if car is None and "Car" in _types(data):
            car = data
        if isinstance(data, dict) and isinstance(data.get("@graph"), list):
            graph_images += [n.get("contentUrl") for n in data["@graph"]
                             if isinstance(n, dict) and "ImageObject" in _types(n)]
    if car is None:
        return None

    engine = ((car.get("vehicleEngine") or {}).get("engineDisplacement") or {})
    litres = _fa_num(engine.get("value")) if engine.get("unitCode") == "LTR" else None
    interior = str(car.get("vehicleInteriorColor") or "").strip()
    consumption = _fa_num((car.get("fuelConsumption") or {}).get("value"))
    accel = _fa_num((car.get("accelerationTime") or {}).get("value"))

    return _detail(
        description=_BAMA_SEO.sub("", str(car.get("description") or "")),
        color=car.get("color"),
        condition=_condition(car.get("itemCondition")),
        fuel=car.get("fuelType"),
        body=_body(car.get("bodyType")),
        engine=f"{litres} لیتر" if litres else None,
        images=_images([car.get("image"), *graph_images]),
        extras={
            # Bama fills this with "-" when the seller left it blank.
            "رنگ داخلی": interior if interior not in ("", "-") else "",
            "مصرف سوخت": f"{consumption} لیتر در ۱۰۰ کیلومتر" if consumption else "",
            "شتاب ۰ تا ۱۰۰": f"{accel} ثانیه" if accel else "",
            "شناسه آگهی": str(car.get("identifier") or ""),
        },
    )


def from_sheypoor(html: str) -> dict | None:
    """Sheypoor wraps the car in an ItemPage; the Vehicle is under mainEntity.

    Worth stating plainly because the brief warned it might be description-only:
    it is not. mainEntity also carries color, knownVehicleDamages in the
    seller's own Persian, itemCondition, bodyType, the fuel type nested under
    vehicleEngine, and the full photo array. What it does not carry is engine
    displacement, so ``engine`` stays null for this source.
    """
    entity = None
    for data in _ld_blocks(html):
        main = data.get("mainEntity") if isinstance(data, dict) else None
        if isinstance(main, dict):
            entity = main
            break
    if entity is None:
        return None

    images = entity.get("image")
    urls = [i.get("contentUrl") for i in images if isinstance(i, dict)] \
        if isinstance(images, list) else [images]
    return _detail(
        description=entity.get("description"),
        color=entity.get("color"),
        damages=entity.get("knownVehicleDamages"),
        condition=_condition(entity.get("itemCondition")),
        fuel=(entity.get("vehicleEngine") or {}).get("fuelType"),
        body=_body(entity.get("bodyType")),
        images=_images(urls),
    )


# Hamrah's spec sheet is a list of {key, title, value}. Only the keys that add
# something are read: `km` and `gearBox` are already in the index and a second
# copy on the page is a second chance to disagree with it.
_HAMRAH_SPECS = {"color": "color", "fuelType": "fuel", "bodyType": "body",
                 "bodyCondition": "damages"}
_HAMRAH_EXTRAS = {"technicalDescription": "وضعیت فنی", "document": "سند",
                  "remainingInsurance": "مانده بیمه"}


def from_hamrah(html: str) -> dict | None:
    """Hamrah-Mechanic: __NEXT_DATA__, and the only source with a real inspection.

    ``carInformation`` also carries ``consultantPhone`` unmasked. It is read
    past deliberately — see the redaction note at the top of this file.
    """
    props = _next_data(html).get("props", {}).get("pageProps", {})
    if not isinstance(props, dict) or not props:
        return None
    order = props.get("orderDetails") or {}

    fields: dict[str, str] = {}
    extras: dict[str, str] = {}
    for spec in ((order.get("carSpecifications") or {}).get("specs") or []):
        if not isinstance(spec, dict):
            continue
        value = str(spec.get("value") or "").strip()
        if not value:
            continue
        if spec.get("key") in _HAMRAH_SPECS:
            fields[_HAMRAH_SPECS[spec["key"]]] = value
        elif spec.get("key") in _HAMRAH_EXTRAS:
            extras[_HAMRAH_EXTRAS[spec["key"]]] = value

    features = (order.get("carFeature") or {}).get("features") or []
    if isinstance(features, list) and features:
        extras["نقاط قوت"] = "، ".join(str(f).strip() for f in features if str(f).strip())
    car_type = str((order.get("carInformation") or {}).get("carTypeName") or "").strip()
    if car_type:
        extras["تیپ"] = re.sub(r"\s+", " ", car_type)

    return _detail(
        description=_HAMRAH_BOILERPLATE.sub(
            "", str((order.get("descriptionPart") or {}).get("description") or "")),
        color=fields.get("color"),
        damages=fields.get("damages"),
        fuel=fields.get("fuel"),
        body=_body(fields.get("body")),
        images=_images([g.get("largeImage") for g in (props.get("gallery") or [])
                        if isinstance(g, dict)]),
        inspection=_inspection(props.get("inspectionReport")),
        extras=extras,
    )


def _inspection(report: Any) -> dict | None:
    """Distil a 38-part-per-section inspection into what a buyer reads.

    The raw report is ~90 parts with an icon path and two hex colours each, and
    for a clean car 85 of them say nothing. Kept: the per-section scores, and
    every part that is either not perfect or has something written against it.
    The second half of that condition matters — a part can be marked perfect and
    still say «امکان کارشناسی وجود نداشت», which is the inspector telling you he
    could not check it, and dropping it would turn "unknown" into "fine".
    """
    if not isinstance(report, dict) or not report.get("collections"):
        return None
    sections, parts = [], []
    for coll in report.get("collections") or []:
        if not isinstance(coll, dict):
            continue
        sections.append({
            "key": coll.get("key"),
            "title": coll.get("title"),
            "score": coll.get("score"),
            "damaged": coll.get("numberOfDamagedParts"),
            "healthy": coll.get("numberOfHealthyParts"),
        })
        for part in coll.get("parts") or []:
            if not isinstance(part, dict):
                continue
            notes = [str(d.get("text")).strip()
                     for d in ((part.get("condition") or {}).get("conditionDescription") or [])
                     if isinstance(d, dict) and d.get("text")]
            perfect = bool(part.get("isPerfect"))
            if perfect and not notes:
                continue
            parts.append({"section": coll.get("title"), "part": part.get("title"),
                          "ok": perfect, "note": " ".join(notes) or None})
    if not sections:
        return None
    return {
        "title": report.get("reportTitle"),
        "sections": sections,
        "parts": parts,
        "damaged_parts": [p for p in (report.get("damagedParts") or []) if p],
    }


# Khodro45's document type enum; everything else it publishes is either an
# integer id with no published lookup (body_color: 1, fuel_type: 1) or already
# in the index.
_K45_DOC = {"6_dong": "شش‌دانگ", "power_of_attorney": "وکالتی", "free": "آزاد"}


def from_khodro45(text: str) -> dict | None:
    """Khodro45's detail JSON, which nothing is allowed to fetch.

    Written and tested, wired in disabled. Both routes to this data are closed
    to a crawler by the site's own robots.txt:

        Disallow: /view/      the detail page, which is a client-rendered React
                              shell — 2.3 KB, `<div id="root">`, no car data at
                              any URL depth
        Disallow: /api        including /api/v1/car_listing/{slug}/, the only
                              place the detail payload exists

    and the sitemap it does publish (sitemap-usedcar.xml, 991 URLs) lists
    category pages only, never a listing. The site is saying, in the one file
    that exists for saying it, that per-listing crawling is not welcome.

    The extractor stays because the finding should be reproducible and because
    the day Khodro45 publishes listing pages this is four lines from working.
    See SOURCES in enrich_details.py for the flag.
    """
    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        return None
    if not isinstance(data, dict):
        return None
    specs = data.get("car_specifications") or {}
    return _detail(
        description=specs.get("additional_information"),
        body=_body(specs.get("body_type")),
        images=_images([(img.get("image") or {}).get("url")
                        for img in (data.get("car_images") or []) if isinstance(img, dict)]),
        extras={
            "سند": _K45_DOC.get(str(specs.get("doc_type") or ""), ""),
            "ظرفیت": f"{_fa_num(specs.get('capacity'))} نفر" if specs.get("capacity") else "",
            "گارانتی خودرو۴۵": "دارد" if data.get("guaranteed") else "",
        },
    )


EXTRACTORS = {
    "divar": from_divar,
    "bama": from_bama,
    "sheypoor": from_sheypoor,
    "hamrah": from_hamrah,
    "khodro45": from_khodro45,
}


def extract(source: str, text: str) -> dict | None:
    """One entry point. None means the page carried no detail worth keeping.

    Never raises on junk. A soft block, a 404 shell, a redirect to the homepage
    and a genuine parse failure all arrive here as text, and the caller needs to
    tell "nothing" from "crashed" — a raise would abort a 900-second run over
    one malformed page.
    """
    fn = EXTRACTORS.get(source)
    if fn is None or not text:
        return None
    try:
        return fn(text)
    except (AttributeError, KeyError, TypeError, ValueError):
        return None

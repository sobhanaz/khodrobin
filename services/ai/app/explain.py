"""Write two Persian sentences saying why the top result wins — and prove it.

The interesting part of this module is not the prompt. It is the guard
underneath it.

A model asked to explain a car listing will, sooner or later, state a fact that
was not in its input: a mileage it rounded, a price it recalculated, a year it
inferred. On a price-comparison product that is not a cosmetic flaw — the whole
value proposition is that the numbers are trustworthy. So every number the model
emits is checked against the numbers it was given, and an explanation containing
an unsupported one is thrown away and replaced by a templated sentence built
from the data itself.

The product therefore always has an explanation, and the explanation is always
true. Those two properties matter more than eloquence.
"""
from __future__ import annotations

import re
from dataclasses import dataclass

FA_DIGITS = str.maketrans("۰۱۲۳۴۵۶۷۸۹", "0123456789")

# Numbers this small are ordinary prose — «دو منبع», «۳ آگهی» — not claims about
# the car. Checking them produces false alarms without catching anything real.
IGNORE_BELOW = 100

SYSTEM = """تو یک مشاور خرید خودرو هستی. دقیقاً دو جمله‌ی کوتاه فارسی می‌نویسی که توضیح می‌دهد چرا گزینه‌ی اول بهترین انتخاب است.

قواعد سخت‌گیرانه:
- فقط از عددها و واقعیت‌هایی استفاده کن که در داده‌ی ورودی آمده‌اند. هیچ عددی از خودت نساز و هیچ عددی را گرد نکن.
- جمله‌ی اول: چرا گزینه‌ی اول برنده است.
- جمله‌ی دوم: در ازای آن چه چیزی را از دست می‌دهی (مقایسه با گزینه‌ی دوم).
- درباره‌ی چیزی که در داده نیست حرف نزن: رنگ، بیمه، شرایط فروشنده، وضعیت موتور.
- بدون مقدمه، بدون فهرست، بدون ایموجی. فقط دو جمله.
- جمله‌ی دوم باید یک «در عوض» واقعی باشد: چیزی که گزینه‌ی اول از گزینه‌ی دوم بدتر است. کارکرد کمتر یا قیمت پایین‌تر مزیت است، نه ضرر. اگر گزینه‌ی اول از هر نظر بهتر بود، بنویس که تنها تفاوتش چیست.

مثال درست:
ورودی: گزینه ۱ = ۱٬۶۶۰٬۰۰۰٬۰۰۰ تومان، ۲۱٪ زیر میانه، کارکرد ۳۰٬۰۰۰ کیلومتر. گزینه ۲ = ۱٬۹۲۰٬۰۰۰٬۰۰۰ تومان، کارکرد ۱۲٬۰۰۰ کیلومتر.
خروجی: این خودرو با ۱٬۶۶۰٬۰۰۰٬۰۰۰ تومان، ۲۱٪ زیر میانه‌ی بازار است. در عوض ۱۸٬۰۰۰ کیلومتر بیشتر از گزینه‌ی بعدی کار کرده است.

مثال غلط (این کار را نکن):
«در ازای آن کمترین کارکرد را از دست می‌دهی» — کارکرد کم مزیت است؛ از دست دادنش بی‌معنی است."""


# Grouping marks Persian and English writers both use. The decimal point is
# deliberately absent: treating "." as a separator turned «۱۱.۶٪» into 116 and
# «۲.۱ میلیارد» into 21, so the guard rejected the model for citing figures it
# had never written. Caught in production, rejecting a correct explanation.
GROUPING = ",،٬  ‏‎"

_NUMBER = re.compile(r"\d[\d,،٬\u00a0\u200f\u200e ]*(?:\.\d+)?")


def numbers_in(text: str) -> set[float]:
    """Every quantity mentioned, in either digit system.

    Returns floats because Persian writes percentages with a decimal — «۱۱.۶٪»
    is one number, not 116.
    """
    normalized = text.translate(FA_DIGITS)
    found: set[float] = set()
    for raw in _NUMBER.findall(normalized):
        cleaned = raw
        for ch in GROUPING:
            cleaned = cleaned.replace(ch, "")
        cleaned = cleaned.rstrip(".")
        if not cleaned:
            continue
        try:
            found.add(float(cleaned))
        except ValueError:
            continue
    return found


def digits_in(text: str) -> set[int]:
    """Integer view of numbers_in, for callers that only care about whole values."""
    return {int(n) for n in numbers_in(text) if n == int(n)}


def supported_numbers(facts: dict) -> set[int]:
    """Numbers the model is allowed to say, plus the forms it may say them in.

    A price of 2,114,000,000 tomans is legitimately written as «۲.۱ میلیارد» or
    «۲۱۱۴ میلیون». Accepting only the exact integer would reject correct,
    natural Persian — so the scaled forms are pre-computed and allowed too.
    """
    allowed: set[float] = set()

    def add(n: float | None) -> None:
        if n is None:
            return
        n = float(n)
        allowed.add(n)
        if n == int(n):
            allowed.add(float(int(n)))
        # Unit-scaled restatements of the same figure: 2,114,000,000 tomans is
        # legitimately «۲۱۱۴ میلیون» or «۲.۱ میلیارد».
        for scale in (1_000, 1_000_000, 1_000_000_000):
            scaled = n / scale
            allowed.add(scaled)
            allowed.add(float(round(scaled)))
            allowed.add(round(scaled, 1))

    # Iranian model names are numbers: ۲۰۶، ۴۰۵، ۱۳۱، X33. Saying the car's own
    # name is not a claim about it, and flagging «پژو ۲۰۷» as an invented figure
    # would reject every correct explanation ever written.
    for n in digits_in(facts.get("title") or ""):
        allowed.add(n)
    for offer in facts.get("offers", []):
        add(offer.get("price"))
        add(offer.get("mileage_km"))
        # The percentage is given as a decimal and the model quotes it that way,
        # so both the exact value and its rounding are legitimate.
        pct = offer.get("vs_median_pct")
        if pct is not None:
            add(abs(float(pct)))
            add(float(abs(round(float(pct)))))
    for key in ("median_price", "min_price", "max_price", "year",
                "offer_count", "source_count"):
        add(facts.get(key))
    # Comparing two offers is the point of the product, so the differences
    # between them are derived facts rather than invented ones. Only pairwise
    # gaps between given prices and given mileages are allowed — not arbitrary
    # arithmetic, which would widen the guard until it caught nothing.
    offers = facts.get("offers", [])
    for i, a in enumerate(offers):
        for b in offers[i + 1:]:
            for field in ("price", "mileage_km"):
                x, y = a.get(field), b.get(field)
                if x is not None and y is not None:
                    add(abs(int(x) - int(y)))

    if facts.get("km_bucket") is not None:
        b = facts["km_bucket"]
        add(b * 25)
        add((b + 1) * 25)
        add(b * 25_000)
        add((b + 1) * 25_000)
    return allowed


# Claims the input never contains. The prompt already forbids these; the guard
# exists because a prompt is a request and a check is a guarantee.
#
# Found by running the real model: asked to explain a Peugeot 207 it wrote
# «می‌توانید از ضمانت مکانیکی که ارائه می‌شود بهره‌مند شوید» — inventing a
# mechanical warranty out of nothing. It cited no number, so a numeric guard
# alone let it through. On a price-comparison product an invented warranty is
# as damaging as an invented price.
FORBIDDEN_TOPICS: dict[str, tuple[str, ...]] = {
    "warranty": ("ضمانت", "گارانتی", "وارانتی"),
    "insurance": ("بیمه",),
    "engine_condition": ("موتور سالم", "موتور تعویض", "وضعیت موتور", "گیربکس سالم"),
    "inspection": ("کارشناسی شده", "دیاگ", "پلمپ"),
    "seller": ("فروشنده معتبر", "نمایشگاه معتبر", "شخصی است", "قابل اعتماد"),
    "financing": ("اقساط", "لیزینگ", "وام", "چک"),
    "negotiation": ("قابل مذاکره", "تخفیف می‌دهد"),
}


def forbidden_claims(text: str, facts: dict) -> list[str]:
    """Topics the model mentioned that the input never supplied.

    A phrase is only a violation when nothing in the given facts mentions it —
    a flag that genuinely says «کارشناسی شده» makes that phrase fair game.
    """
    haystack = " ".join([
        facts.get("title") or "",
        *(facts.get("flags") or []),
        *(str(o.get("source_fa") or "") for o in facts.get("offers", [])),
    ])
    hits: list[str] = []
    for topic, phrases in FORBIDDEN_TOPICS.items():
        for phrase in phrases:
            if phrase in text and phrase not in haystack:
                hits.append(topic)
                break
    return hits


@dataclass
class Verdict:
    ok: bool
    text: str
    unsupported: list[int]
    source: str  # model | fallback


# Floats compared for equality need a tolerance. This one is tight enough that
# 11.6 does not match 11.7, and loose enough to absorb the representation error
# of dividing a billion by a billion.
EPSILON = 1e-6


def _supported(value: float, allowed: set[float]) -> bool:
    return any(abs(value - a) <= EPSILON for a in allowed)


def check(text: str, facts: dict) -> tuple[bool, list[float], list[str]]:
    """Verify the explanation against its input.

    Two independent checks, because the model fails in two different ways:
    numbers it invented, and subjects it invented. Either one alone lets real
    fabrications through.
    """
    allowed = supported_numbers(facts)
    bad_numbers = [
        n for n in numbers_in(text)
        if n >= IGNORE_BELOW and not _supported(n, allowed)
    ]
    bad_topics = forbidden_claims(text, facts)
    return (not bad_numbers and not bad_topics), sorted(bad_numbers), bad_topics


def fallback(facts: dict) -> str:
    """A templated explanation assembled only from the data.

    Used when the model is unreachable, too slow, or caught inventing a number.
    It is less fluent and completely true, which is the right trade for a
    product whose claim is that its numbers can be trusted.
    """
    offers = facts.get("offers", [])
    if not offers:
        return "برای این خودرو آگهی کافی نداریم."

    best = offers[0]
    pct = abs(int(round(best.get("vs_median_pct", 0))))
    median = facts.get("median_price", 0)
    name = facts.get("title", "این خودرو")

    first = (
        f"{name} در «{best.get('source_fa', 'این منبع')}» با قیمت "
        f"{best.get('price', 0):,} تومان، {pct}٪ پایین‌تر از میانه‌ی "
        f"{median:,} تومانی این مدل است."
    ) if pct else (
        f"{name} در «{best.get('source_fa', 'این منبع')}» با قیمت "
        f"{best.get('price', 0):,} تومان، نزدیک به میانه‌ی بازار است."
    )

    if len(offers) > 1:
        second = offers[1]
        km_a, km_b = best.get("mileage_km"), second.get("mileage_km")
        if km_a is not None and km_b is not None and km_a != km_b:
            more = "بیشتر" if km_a > km_b else "کمتر"
            second_sentence = (
                f"در عوض {abs(km_a - km_b):,} کیلومتر {more} از گزینه‌ی بعدی "
                f"({second.get('price', 0):,} تومان) کار کرده است."
            )
        else:
            second_sentence = (
                f"گزینه‌ی بعدی {second.get('price', 0):,} تومان است "
                f"از «{second.get('source_fa', 'منبعی دیگر')}»."
            )
    else:
        second_sentence = (
            f"این مدل فقط {facts.get('offer_count', 1)} آگهی دارد، "
            "پس میانه‌ی قیمت هنوز کم‌اعتبار است."
        )

    return f"{first} {second_sentence}"


def build_prompt(facts: dict) -> str:
    """Hand the model a compact, closed set of facts.

    Passing the whole spec object invites the model to comment on fields that
    are not decision-relevant. Only what a buyer would weigh goes in.
    """
    lines = [
        f"خودرو: {facts.get('title', '')}",
        f"سال: {facts.get('year', '')}",
        f"میانه‌ی قیمت: {facts.get('median_price', 0):,} تومان",
        f"تعداد آگهی: {facts.get('offer_count', 0)} از {facts.get('source_count', 0)} منبع",
        "",
        "آگهی‌ها (به ترتیب قیمت):",
    ]
    for i, o in enumerate(facts.get("offers", [])[:3], 1):
        km = o.get("mileage_km")
        lines.append(
            f"{i}. {o.get('source_fa', '')} — {o.get('price', 0):,} تومان"
            f" — {o.get('vs_median_pct', 0)}٪ نسبت به میانه"
            + (f" — کارکرد {km:,} کیلومتر" if km is not None else "")
        )
    if facts.get("flags"):
        lines.append("")
        lines.append("هشدارها: " + "؛ ".join(facts["flags"]))
    return "\n".join(lines)


def facts_from_spec(spec: dict) -> dict:
    """Reduce an index spec to the decision-relevant facts."""
    offers = spec.get("offers", [])[:3]
    name_parts = [spec.get("brand_fa", ""), spec.get("model_fa", "")]
    return {
        "title": " ".join(p for p in name_parts if p).strip(),
        "year": spec.get("year"),
        "median_price": spec.get("median_price", 0),
        "min_price": spec.get("min_price", 0),
        "max_price": spec.get("max_price", 0),
        "offer_count": spec.get("offer_count", 0),
        "source_count": spec.get("source_count", 0),
        "km_bucket": spec.get("km_bucket"),
        "offers": [
            {
                "source_fa": o.get("source_fa"),
                "price": o.get("price", 0),
                "mileage_km": o.get("mileage_km"),
                "vs_median_pct": o.get("vs_median_pct", 0),
            }
            for o in offers
        ],
        "flags": [f.get("message", "") for o in offers for f in (o.get("flags") or [])],
    }

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
#
# It also used to hide the product's headline claim. Every vs_median_pct is a one
# or two digit number, so «۴۵٪ زیر میانه» on a car that is 3٪ under was filtered
# out before the allow-set was ever consulted. Percentages are therefore checked
# separately below, where their size is irrelevant.
IGNORE_BELOW = 100

# A number written immediately before ٪, % or «درصد» is a claim about position
# against the median, and must match one the data supports.
_PERCENT = re.compile(r"(\d+(?:\.\d+)?)\s*(?:٪|%|درصد)")

# Persian and Arabic letters, for word-boundary checks. \b does not work here:
# Go and Python both treat Persian letters as word characters, but «چک» sits
# inside «کوچک», so a bare substring test rejects the word for "smaller".
_PERSIAN_LETTER = r"\u0621-\u064A\u0660-\u0669\u066E-\u06D3\u06F0-\u06F9"

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


def supported_percentages(facts: dict) -> set[float]:
    """Percentages the data actually supports.

    The only real percentage here is a position against the median. Anything
    else the model writes with a ٪ sign is invented, however plausible it looks.
    """
    allowed: set[float] = set()
    for offer in facts.get("offers", []):
        pct = offer.get("vs_median_pct")
        if pct is None:
            continue
        value = abs(float(pct))
        allowed.add(value)
        allowed.add(round(value, 1))
        # Both neighbours: a person writing about 21.5٪ says «۲۱٪» or «۲۲٪», and
        # rejecting either would fail correct Persian. The gap this leaves is one
        # percentage point, which cannot turn a small discount into a large one.
        allowed.add(float(int(value)))
        allowed.add(float(int(value) + 1) if value != int(value) else float(int(value)))
    return allowed


def bad_percentages(text: str, facts: dict) -> list[float]:
    """Percentages stated in the text that no offer supports."""
    allowed = supported_percentages(facts)
    out: list[float] = []
    for raw in _PERCENT.findall(text.translate(FA_DIGITS)):
        try:
            value = float(raw)
        except ValueError:
            continue
        if not any(abs(value - a) <= 0.05 for a in allowed):
            out.append(value)
    return out


# Marketplaces a Persian car listing might name. Used to catch a model that
# attributes an offer to a source this spec has no listing from — the sentence
# reads perfectly and the attribution is fiction.
KNOWN_MARKETPLACES = (
    "دیوار", "باما", "همراه\u200cمکانیک", "همراه مکانیک", "خودرو۴۵", "خودرو ۴۵",
    "ایران\u200cجیب", "ایران جیب", "شیپور", "اتومارکت", "بازار خودرو",
    "خودروبانک", "ماشین\u200cبازار", "کارنامه", "هملو",
)


def bad_sources(text: str, facts: dict) -> list[str]:
    """Marketplaces named in the text that have no offer in this spec."""
    present = {str(o.get("source_fa") or "") for o in facts.get("offers", [])}
    # Fold the ZWNJ so «همراه‌مکانیک» and «همراه مکانیک» compare equal.
    flat = {p.replace("\u200c", " ") for p in present if p}
    haystack = text.replace("\u200c", " ")
    hits: list[str] = []
    for name in KNOWN_MARKETPLACES:
        plain = name.replace("\u200c", " ")
        if plain in flat:
            continue
        # Bounded, and not immediately followed by a digit or a percent sign.
        # «خودرو» means "car", so «این خودرو ۴۵٪ زیر میانه» contains the literal
        # marketplace name «خودرو ۴۵» by accident — a real rejection with an
        # invented reason, which is worse than no reason.
        pattern = (rf"(?<![{_PERSIAN_LETTER}a-zA-Z]){re.escape(plain)}"
                   rf"(?![{_PERSIAN_LETTER}a-zA-Z0-9\u06F0-\u06F9]|\s*[٪%])")
        if re.search(pattern, haystack):
            hits.append(plain)
    return sorted(set(hits))


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
                "offer_count", "source_count"):  # noqa: E501
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

    # km_bucket is deliberately absent. It is a display band, not a fact about
    # any offer: including it let the guard accept «۲۵٬۰۰۰ کیلومتر» on a car
    # whose real readings were 12,000 and 30,000. Every genuine mileage is
    # already in the set from offer.mileage_km.
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
    # The system prompt forbids «رنگ» explicitly and the guard did not enforce
    # it, so «این خودرو بدون رنگ و بدون تصادف است» passed clean.
    "paint": ("بدون رنگ", "رنگ‌شدگی", "دور رنگ", "تمام رنگ", "بدون تصادف", "تصادفی"),
    "options": ("فول آپشن", "آپشن کامل", "فول"),
    "paperwork": ("تعویض پلاک", "سند آزاد", "مدارک کامل"),
    # A superlative is a claim about every listing in the market, and the input
    # only ever describes one spec.
    "superlative": ("بهترین قیمت بازار", "کم‌کارکردترین", "بی‌نظیرترین", "تمیزترین"),
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
            if _mentions(text, phrase) and not _mentions(haystack, phrase):
                hits.append(topic)
                break
    return hits


def _mentions(text: str, phrase: str) -> bool:
    """Whole-word containment.

    A bare substring test rejected «کوچک» because «چک» (cheque) is inside it,
    and «وام» inside other words. The phrase must start and end at a
    non-letter — Persian letters included, which is what \b gets wrong.
    """
    pattern = rf"(?<![{_PERSIAN_LETTER}a-zA-Z]){re.escape(phrase)}(?![{_PERSIAN_LETTER}a-zA-Z])"
    return re.search(pattern, text) is not None


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

    Four independent checks, because the model invents in four different ways
    and any one of them alone lets real fabrications through:

      numbers      — a price or mileage that was never given
      percentages  — a position against the median that no offer supports
      topics       — a subject the input never mentioned
      sources      — an offer attributed to a marketplace with no listing here

    Percentages need their own pass because they are small: the numeric check
    ignores anything under IGNORE_BELOW to avoid flagging «۳ منبع», which meant
    the product's headline claim went unchecked entirely.
    """
    allowed = supported_numbers(facts)
    bad_numbers = [
        n for n in numbers_in(text)
        if n >= IGNORE_BELOW and not _supported(n, allowed)
    ]
    bad_pcts = bad_percentages(text, facts)
    bad_topics = forbidden_claims(text, facts)
    bad_src = bad_sources(text, facts)

    ok = not (bad_numbers or bad_pcts or bad_topics or bad_src)
    # Percentages surface in the numeric list so callers and the UI keep one
    # "numbers the model made up" channel; sources join the topic channel for
    # the same reason.
    return ok, sorted(set(bad_numbers) | set(bad_pcts)), bad_topics + [f"source:{s}" for s in bad_src]


# A mileage gap smaller than this is noise between two used cars, not a reason
# to pick one. Stating «۴۱ کیلومتر کمتر» as though it mattered reads as a
# machine padding a sentence.
MEANINGFUL_KM_GAP = 5_000


def fallback(facts: dict) -> str:
    """A templated explanation assembled only from the data.

    Used when the model is unreachable, too slow, or caught inventing something.
    Less fluent and completely true, which is the right trade for a product
    whose claim is that its numbers can be trusted.

    The second sentence must be an actual trade-off. «در عوض» means "in
    exchange", so it may only introduce something the cheapest option is WORSE
    at. Lower mileage is an advantage; announcing it as a sacrifice — which this
    function used to do — is the same mistake the system prompt tells the model
    not to make.
    """
    offers = facts.get("offers", [])
    if not offers:
        return "برای این خودرو آگهی کافی نداریم."

    best = offers[0]
    pct = abs(int(round(best.get("vs_median_pct", 0))))
    median = facts.get("median_price", 0)
    name = facts.get("title", "این خودرو")
    source = best.get("source_fa", "این منبع")
    price = best.get("price", 0)

    if pct and facts.get("median_reliable", True):
        first = (f"{name} در «{source}» با قیمت {price:,} تومان، "
                 f"{pct}٪ پایین‌تر از میانه‌ی {median:,} تومانی این مدل است.")
    elif pct:
        # Below three offers the median is the mean of a couple of asking
        # prices, so it is quoted as a comparison rather than as market truth.
        first = (f"{name} در «{source}» با قیمت {price:,} تومان، "
                 f"ارزان‌ترین آگهی از {facts.get('offer_count', 1)} آگهی موجود است.")
    else:
        first = (f"{name} در «{source}» با قیمت {price:,} تومان، "
                 "نزدیک به میانه‌ی بازار است.")

    second_sentence = _trade_off(facts, best, offers)
    return f"{first} {second_sentence}"


def _trade_off(facts: dict, best: dict, offers: list[dict]) -> str:
    """The honest second sentence: what the cheapest option costs you.

    Falls back through progressively weaker claims and, when the cheapest offer
    is simply better on every axis we know, says that instead of inventing a
    downside.
    """
    if len(offers) < 2:
        return (f"این مدل فقط {facts.get('offer_count', 1)} آگهی دارد، "
                "پس میانه‌ی قیمت هنوز کم‌اعتبار است.")

    second = offers[1]
    km_a, km_b = best.get("mileage_km"), second.get("mileage_km")

    # A genuine trade-off: cheaper, but more worn.
    if km_a is not None and km_b is not None and km_a - km_b >= MEANINGFUL_KM_GAP:
        return (f"در عوض {km_a - km_b:,} کیلومتر بیشتر از گزینه‌ی بعدی "
                f"({second.get('price', 0):,} تومان) کار کرده است.")

    # Flagged data is a real cost even when the numbers look good.
    if facts.get("flags"):
        return ("در عوض داده‌ی این آگهی یک ناسازگاری دارد که پایین‌تر نشان داده شده است.")

    # Cheaper and less worn: no downside to report, so report the comparison
    # instead of manufacturing one.
    if km_a is not None and km_b is not None and km_b - km_a >= MEANINGFUL_KM_GAP:
        return (f"گزینه‌ی بعدی {second.get('price', 0):,} تومان است و "
                f"{km_b - km_a:,} کیلومتر بیشتر کار کرده، پس این آگهی از هر دو نظر جلوتر است.")

    return (f"تفاوت اصلی‌اش با گزینه‌ی بعدی فقط قیمت است: "
            f"{second.get('price', 0):,} تومان از «{second.get('source_fa', 'منبعی دیگر')}».")


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
        "median_reliable": spec.get("median_reliable", True),
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

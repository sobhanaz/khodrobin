"""Turn a Persian sentence into the same Intent the Go parser produces.

This endpoint exists for the queries rules genuinely cannot do. That boundary
was measured, not guessed: across 92 golden queries the deterministic parser
answers 90, and the two it misses both need Persian numbers written as words —
«ماشین زیر پونصد میلیون», «مدل نود و پنج». Writing a Persian number-word grammar
in regex is possible and miserable; a model does it for free.

So the model's job here is deliberately narrow. It does not rank, it does not
choose, it does not decide what to show. It reads one sentence and fills one
small struct, and everything it returns is validated before it is believed.
"""
from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator

CURRENT_JALALI_YEAR = 1405
MIN_JALALI_YEAR = 1300
# A price below this in tomans is not a car. Guards against a model reading
# «پونصد» as 500 rather than 500,000,000.
MIN_PLAUSIBLE_TOMAN = 10_000_000
MAX_PLAUSIBLE_TOMAN = 500_000_000_000

Priority = Literal["relevant", "cheapest", "lowkm", "value"]


class Intent(BaseModel):
    """Mirrors search.Intent in the Go service, field for field.

    The two must agree: a response the Go side cannot read is worse than no
    response, because it fails after the latency has already been paid.
    """

    brand: str = ""
    model: str = ""
    year_min: int = 0
    year_max: int = 0
    price_min: int = 0
    price_max: int = 0
    gearbox: Literal["", "at", "mt"] = ""
    priority: Priority = "relevant"
    confidence: float = Field(default=0.0, ge=0.0, le=1.0)

    @field_validator("year_min", "year_max")
    @classmethod
    def sane_year(cls, v: int) -> int:
        # A Gregorian year is a plausible thing for a model to emit; convert it
        # rather than rejecting the whole parse over it.
        if 1900 <= v <= 2100:
            v -= 621
        if v and not (MIN_JALALI_YEAR <= v <= CURRENT_JALALI_YEAR + 1):
            return 0
        return v

    @field_validator("price_min", "price_max")
    @classmethod
    def sane_price(cls, v: int) -> int:
        if v and not (MIN_PLAUSIBLE_TOMAN <= v <= MAX_PLAUSIBLE_TOMAN):
            return 0
        return v

    def coherent(self) -> "Intent":
        """Fix orderings that are impossible rather than merely unlikely."""
        if self.year_min and self.year_max and self.year_min > self.year_max:
            self.year_min, self.year_max = self.year_max, self.year_min
        if self.price_min and self.price_max and self.price_min > self.price_max:
            self.price_min, self.price_max = self.price_max, self.price_min
        return self


SYSTEM = """تو یک تحلیل‌گر جست‌وجوی خودرو هستی. جمله‌ی فارسی کاربر را می‌خوانی و فقط یک شیء JSON برمی‌گردانی.

فیلدها:
- brand: اسلاگ انگلیسی برند از فهرست مجاز، یا ""
- model: اسلاگ انگلیسی مدل از فهرست مجاز، یا ""
- year_min, year_max: سال شمسی چهاررقمی (مثل 1396)، یا 0
- price_min, price_max: قیمت به تومان به صورت عدد کامل، یا 0
- gearbox: "at" برای اتوماتیک، "mt" برای دنده‌ای، یا ""
- priority: یکی از relevant | cheapest | lowkm | value
- confidence: عددی بین 0 و 1

قواعد مهم:
- عددهای نوشته‌شده با حروف را به رقم تبدیل کن: «پونصد میلیون» یعنی 500000000، «نود و پنج» یعنی 1395.
- «زیر»، «تا»، «حداکثر» سقف قیمت است. «بالای»، «بیشتر از» کف قیمت است.
- سال دو رقمی شمسی است: «۹۵» یعنی 1395. سال چهاررقمی میلادی را به شمسی تبدیل کن.
- اگر چیزی در جمله نیامده، مقدار پیش‌فرض (خالی یا صفر) بگذار. حدس نزن.
- فقط JSON برگردان، بدون توضیح."""


def build_prompt(query: str, brands: list[dict], models: list[dict]) -> str:
    """Give the model the closed vocabulary it must choose from.

    Without this it invents plausible-looking slugs — "peugeot_206", "pride131" —
    that match nothing in the index. Constraining the output space is cheaper and
    more reliable than correcting it afterwards.
    """
    brand_list = ", ".join(sorted({b["slug"] for b in brands}))
    model_list = ", ".join(sorted({f"{m['brand']}/{m['slug']}" for m in models}))
    return (
        f"برندهای مجاز: {brand_list}\n\n"
        f"مدل‌های مجاز (برند/مدل): {model_list}\n\n"
        f"جمله‌ی کاربر: {query}"
    )


def validate(raw: dict, brands: list[dict], models: list[dict]) -> Intent:
    """Parse the model's answer and discard anything outside the vocabulary.

    A slug the index does not contain filters every result away, so an invented
    brand is worse than an empty one: it turns a vague query into zero results
    with no explanation.
    """
    intent = Intent.model_validate(raw).coherent()

    known_brands = {b["slug"] for b in brands}
    known_models = {(m["brand"], m["slug"]) for m in models}

    if intent.brand and intent.brand not in known_brands:
        intent.brand, intent.model = "", ""
    if intent.model:
        # Models arrive either bare ("206") or qualified ("peugeot/206").
        slug = intent.model.split("/")[-1]
        if (intent.brand, slug) not in known_models:
            intent.model = ""
        else:
            intent.model = slug

    return intent

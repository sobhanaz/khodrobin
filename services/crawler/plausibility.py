"""Flag listings whose own numbers disagree with each other.

Torob lists «کشف تناقض داده» among the problems behind every search, and the
captured data shows why. Real examples from the first crawl:

* Bama lists a **1385 Pride** as «صفر کیلومتر» while also recording
  «گلگیر تعویض» — a forty-year-old car with zero kilometres and body work.
* Divar returns a mileage of exactly 1,000,000 km, which is a human typing
  "unknown", not an odometer reading.
* Divar returns placeholder prices as low as 10,000 rials.

We do not silently repair these. A repaired number is a lie with better
manners; a flagged number lets the user see what the source actually said.
"""
from __future__ import annotations

CURRENT_JALALI_YEAR = 1405

# Above this, a mileage is a sentinel someone typed, not a reading.
IMPLAUSIBLE_KM = 800_000
# A car this old cannot honestly be at zero kilometres.
ZERO_KM_GRACE_YEARS = 2
# Tehran-scale annual mileage; well past this we stop believing the pair.
MAX_KM_PER_YEAR = 60_000

WORKED_ON = {"رنگ‌شدگی", "تمام‌رنگ", "تعویض"}

# Listings that are not a car you can drive away.
#
# A «حواله» is an allocation certificate and a «پیش‌فروش» is a car that does not
# exist yet; both are priced far below a real one because they are not the same
# product. The price-ratio check misses them constantly — one sat at 0.41 of its
# cluster median, just inside a 0.4 threshold — while the seller had written
# «حواله» in the title. A stated fact beats a statistical guess about it.
NOT_A_CAR_YET = ("حواله", "پیش فروش", "پیش‌فروش", "پیش خرید", "پیش‌خرید",
                 "مشارکت در تولید", "قرعه کشی", "قرعه‌کشی", "ثبت نام", "ثبت‌نام")


def flags(car: dict) -> list[dict[str, str]]:
    """Return zero or more {code, message} findings for one canonical car."""
    found: list[dict[str, str]] = []
    km = car.get("mileage_km")
    year = car.get("year_jalali")
    status = car.get("body_status")

    if km is not None and km >= IMPLAUSIBLE_KM:
        found.append({
            "code": "mileage_sentinel",
            "message": f"کارکرد {km:,} کیلومتر واقعی به نظر نمی‌رسد؛ احتمالاً «نامشخص» ثبت شده.",
        })

    age = (CURRENT_JALALI_YEAR - year) if year else None
    if km == 0 and age is not None and age > ZERO_KM_GRACE_YEARS:
        found.append({
            "code": "zero_km_old_car",
            "message": f"خودروی مدل {year} با کارکرد صفر ثبت شده است.",
        })

    if km == 0 and status in WORKED_ON:
        found.append({
            "code": "zero_km_with_bodywork",
            "message": f"کارکرد صفر اعلام شده اما بدنه «{status}» دارد.",
        })

    if km and age and age > 0 and km / age > MAX_KM_PER_YEAR:
        found.append({
            "code": "mileage_vs_age",
            "message": f"میانگین {km // age:,} کیلومتر در سال برای مدل {year} غیرعادی است.",
        })

    title = (car.get("title") or "").replace("\u200c", " ")
    for phrase in NOT_A_CAR_YET:
        if phrase.replace("\u200c", " ") in title:
            found.append({
                "code": "not_a_car_yet",
                "message": ("این آگهی حواله یا پیش‌فروش است، نه خودروی آماده‌ی تحویل؛ "
                            "قیمتش با آگهی‌های دیگر قابل مقایسه نیست."),
            })
            break

    if car.get("price_toman") is None:
        found.append({"code": "no_price", "message": "قیمت معتبری اعلام نشده است."})

    return found

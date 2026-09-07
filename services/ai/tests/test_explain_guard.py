"""The guard that makes an explanation trustworthy.

A price-comparison product whose explanation invents a number is not slightly
wrong, it is wrong about the only thing it claims to do. These tests pin that.
"""
import sys, pathlib
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

from app import explain


FACTS = {
    "title": "پژو ۲۰۷",
    "year": 1404,
    "median_price": 2_114_000_000,
    "min_price": 1_660_000_000,
    "max_price": 2_470_000_000,
    "offer_count": 10,
    "source_count": 3,
    "km_bucket": 0,
    "offers": [
        {"source_fa": "همراه‌مکانیک", "price": 1_660_000_000, "mileage_km": 12_000, "vs_median_pct": -21.5},
        {"source_fa": "دیوار", "price": 1_920_000_000, "mileage_km": 30_000, "vs_median_pct": -9.2},
        {"source_fa": "باما", "price": 2_090_000_000, "mileage_km": 8_000, "vs_median_pct": -1.1},
    ],
    "flags": [],
}


def test_an_explanation_using_only_given_numbers_passes():
    text = ("این پژو ۲۰۷ در همراه‌مکانیک با ۱٬۶۶۰٬۰۰۰٬۰۰۰ تومان، ۲۱٪ زیر میانه‌ی "
            "۲٬۱۱۴٬۰۰۰٬۰۰۰ تومانی است. در عوض ۱۲۰۰۰ کیلومتر کار کرده است.")
    ok, bad, _ = explain.check(text, FACTS)
    assert ok, bad


def test_an_invented_price_is_caught():
    # 1,700,000,000 was never in the input. A model that rounds a price is
    # inventing one.
    text = "این خودرو با ۱٬۷۰۰٬۰۰۰٬۰۰۰ تومان ارزان‌ترین گزینه است."
    ok, bad, _ = explain.check(text, FACTS)
    assert not ok
    assert 1_700_000_000 in bad


def test_an_invented_mileage_is_caught():
    text = "کارکرد این خودرو ۴۵۰۰۰ کیلومتر است و قیمت مناسبی دارد."
    ok, bad, _ = explain.check(text, FACTS)
    assert not ok
    assert 45_000 in bad


def test_persian_digits_are_checked_too():
    # An invented number written in Persian digits must not slip past.
    text = "قیمت آن ۹۹۹٬۹۹۹٬۹۹۹ تومان است."
    ok, bad, _ = explain.check(text, FACTS)
    assert not ok


def test_natural_unit_scaling_is_accepted():
    # «۲.۱ میلیارد» and «۱۶۶۰ میلیون» are correct restatements of given figures.
    # Rejecting them would force stilted Persian for no safety gain.
    ok, _, _ = explain.check("میانه حدود ۲۱۱۴ میلیون تومان است.", FACTS)
    assert ok
    ok, _, _ = explain.check("ارزان‌ترین گزینه ۱۶۶۰ میلیون تومان است.", FACTS)
    assert ok


def test_small_numbers_are_ignored():
    # «۳ منبع», «۱۰ آگهی» are prose, not claims about the car.
    ok, _, _ = explain.check("این خودرو ۱۰ آگهی از ۳ منبع دارد.", FACTS)
    assert ok


def test_the_fallback_is_always_true_and_never_empty():
    text = explain.fallback(FACTS)
    assert text
    ok, bad, _ = explain.check(text, FACTS)
    assert ok, f"the fallback itself cited an unsupported number: {bad}"


def test_the_fallback_survives_a_single_offer():
    facts = FACTS | {"offers": FACTS["offers"][:1], "offer_count": 1, "source_count": 1}
    text = explain.fallback(facts)
    assert text
    ok, bad, _ = explain.check(text, facts)
    assert ok, bad


def test_the_fallback_survives_missing_mileage():
    facts = FACTS | {"offers": [dict(o, mileage_km=None) for o in FACTS["offers"]]}
    text = explain.fallback(facts)
    assert text
    ok, bad, _ = explain.check(text, facts)
    assert ok, bad


def test_facts_are_reduced_to_what_a_buyer_weighs():
    spec = {
        "brand_fa": "پژو", "model_fa": "۲۰۷", "year": 1404,
        "median_price": 2_114_000_000, "min_price": 1_660_000_000,
        "max_price": 2_470_000_000, "offer_count": 10, "source_count": 3,
        "km_bucket": 0,
        "offers": [{"source_fa": "دیوار", "price": 1_920_000_000, "mileage_km": 30_000,
                    "vs_median_pct": -9.2, "url": "https://divar.ir/v/x",
                    "colour": "سفید", "flags": []}] * 5,
    }
    facts = explain.facts_from_spec(spec)
    assert facts["title"] == "پژو ۲۰۷"
    # Only the top three offers are shown to the model, and irrelevant fields
    # (url, colour) are dropped so it cannot comment on them.
    assert len(facts["offers"]) == 3
    assert "url" not in facts["offers"][0]
    assert "colour" not in facts["offers"][0]


def test_a_model_name_that_is_a_number_is_not_a_claim():
    # Iranian model names are numbers. The guard flagged «پژو ۲۰۷» as an
    # invented figure, which would have rejected every correct explanation.
    ok, bad, _ = explain.check("پژو ۲۰۷ گزینه‌ی خوبی است.", FACTS)
    assert ok, bad
    # But a number that is neither the name nor a given fact still fails.
    ok, bad, _ = explain.check("پژو ۲۰۷ با ۳۳۳٬۳۳۳٬۳۳۳ تومان.", FACTS)
    assert not ok


def test_a_difference_between_two_given_numbers_is_a_derived_fact():
    # Comparing offers is the product. «۱۸٬۰۰۰ کیلومتر کمتر» is 30,000 - 12,000,
    # both of which were given — a derivation, not an invention.
    ok, bad, _ = explain.check("۱۸۰۰۰ کیلومتر کمتر از گزینه‌ی بعدی کار کرده است.", FACTS)
    assert ok, bad
    # A price gap between two given offers is allowed for the same reason.
    ok, bad, _ = explain.check("۲۶۰٬۰۰۰٬۰۰۰ تومان ارزان‌تر است.", FACTS)
    assert ok, bad


def test_arbitrary_arithmetic_is_still_rejected():
    # Widening the guard to any derived value would make it catch nothing.
    # A number that is no given fact and no pairwise gap must still fail.
    ok, bad, _ = explain.check("مجموع این دو ۳٬۵۸۰٬۰۰۰٬۰۰۰ تومان می‌شود.", FACTS)
    assert not ok


def test_an_invented_warranty_is_caught_even_with_no_numbers():
    # The real qwen2.5:7b wrote exactly this when asked to explain a Peugeot 207.
    # It cites no figure, so the numeric guard alone let it through.
    text = ("گزینه‌ی اول بهترین انتخاب است. "
            "در این صورت می‌توانید از ضمانت مکانیکی که ارائه می‌شود بهره‌مند شوید.")
    ok, numbers, topics = explain.check(text, FACTS)
    assert not ok
    assert numbers == []
    assert "warranty" in topics


def test_other_invented_subjects_are_caught():
    for text, topic in [
        ("بیمه‌ی آن تا پایان سال اعتبار دارد.", "insurance"),
        ("موتور سالم است و مشکلی ندارد.", "engine_condition"),
        ("قیمت آن قابل مذاکره است.", "negotiation"),
        ("امکان خرید اقساط وجود دارد.", "financing"),
    ]:
        ok, _, topics = explain.check(text, FACTS)
        assert not ok, text
        assert topic in topics, (text, topics)


def test_a_topic_the_data_actually_mentions_is_allowed():
    # A flag that genuinely says «کارشناسی شده» makes that phrase fair game —
    # the guard blocks invention, not vocabulary.
    facts = FACTS | {"flags": ["این خودرو کارشناسی شده است."]}
    ok, _, topics = explain.check("این خودرو کارشناسی شده است.", facts)
    assert ok, topics


def test_an_ordinary_correct_explanation_still_passes():
    text = ("این پژو ۲۰۷ در همراه‌مکانیک با ۱٬۶۶۰٬۰۰۰٬۰۰۰ تومان، ۲۲٪ زیر میانه است. "
            "در عوض ۱۸۰۰۰ کیلومتر بیشتر از گزینه‌ی بعدی کار کرده است.")
    ok, numbers, topics = explain.check(text, FACTS)
    assert ok, (numbers, topics)

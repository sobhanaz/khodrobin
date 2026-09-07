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
    # This assertion used to be vacuous. IGNORE_BELOW=100 filtered «۲۱٪» out
    # before the allow-set was consulted, so the test proved only that the guard
    # ignored the product's headline claim. The companion test below is the one
    # that gives it meaning.
    text = ("این پژو ۲۰۷ در همراه‌مکانیک با ۱٬۶۶۰٬۰۰۰٬۰۰۰ تومان، ۲۱٪ زیر میانه‌ی "
            "۲٬۱۱۴٬۰۰۰٬۰۰۰ تومانی است. در عوض ۱۲۰۰۰ کیلومتر کار کرده است.")
    ok, bad, topics = explain.check(text, FACTS)
    assert ok, (bad, topics)


def test_a_fabricated_percentage_is_caught():
    # The true position is -21.5%. Claiming 45% is the single most damaging
    # thing this product could say, and it went completely unchecked.
    text = "این پژو ۲۰۷ ۴۵٪ زیر میانه‌ی بازار است."
    ok, bad, _ = explain.check(text, FACTS)
    assert not ok
    assert 45.0 in bad


def test_both_neighbours_of_a_decimal_percentage_are_accepted():
    # A person writing about 21.5٪ says «۲۱٪» or «۲۲٪»; rejecting either would
    # fail correct Persian.
    for pct in ("۲۱", "۲۲", "۲۱.۵"):
        ok, bad, _ = explain.check(f"این خودرو {pct}٪ زیر میانه است.", FACTS)
        assert ok, (pct, bad)


def test_an_offer_attributed_to_an_absent_marketplace_is_caught():
    # The sentence reads perfectly; the attribution is fiction.
    ok, _, topics = explain.check("این خودرو در «ایران‌جیب» ارزان‌ترین است.", FACTS)
    assert not ok
    assert any(t.startswith("source:") for t in topics)
    ok, _, _ = explain.check("این خودرو در «دیوار» ارزان‌تر است.", FACTS)
    assert ok


def test_the_word_for_car_is_not_mistaken_for_a_marketplace():
    # «خودرو» means "car", so «این خودرو ۴۵٪...» contains the literal
    # marketplace name «خودرو ۴۵» by accident.
    _, _, topics = explain.check("این خودرو ۲۱٪ زیر میانه است.", FACTS)
    assert not any(t.startswith("source:") for t in topics), topics


def test_paint_and_condition_claims_are_caught():
    # The system prompt forbids «رنگ» explicitly and the guard did not enforce it.
    for text in ("این خودرو بدون رنگ است.", "این خودرو تصادفی نیست.",
                 "این خودرو فول آپشن است."):
        ok, _, topics = explain.check(text, FACTS)
        assert not ok, text
        assert topics, text


def test_a_superlative_about_the_whole_market_is_caught():
    # The input describes one spec; it cannot support a claim about every car.
    ok, _, topics = explain.check("این کم‌کارکردترین خودروی بازار است.", FACTS)
    assert not ok
    assert "superlative" in topics


def test_common_words_containing_a_forbidden_substring_are_not_rejected():
    # «چک» (cheque) sits inside «کوچک» (small), and a bare substring test
    # rejected the ordinary word for "smaller".
    for text in ("در عوض صندوق عقب کوچک‌تری دارد.", "این خودرو کوچک و کم‌مصرف است."):
        ok, nums, topics = explain.check(text, FACTS)
        assert ok, (text, nums, topics)


def test_the_mileage_band_is_not_a_fact_about_any_offer():
    # km_bucket is a display band. Including it let the guard accept
    # «۲۵٬۰۰۰ کیلومتر» on a car whose real readings were 12,000 and 30,000.
    ok, bad, _ = explain.check("در عوض ۲۵٬۰۰۰ کیلومتر کار کرده است.", FACTS)
    assert not ok
    assert 25000.0 in bad


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


def test_the_fallback_never_frames_an_advantage_as_a_sacrifice():
    # Seen live: «در عوض ۴۱ کیلومتر کمتر ... کار کرده است». «در عوض» means "in
    # exchange" and may only introduce a downside. Lower mileage is an
    # advantage; announcing it as a cost is the exact error the system prompt
    # tells the model to avoid.
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 1_050_000_000, "mileage_km": 20_000, "vs_median_pct": -26.0},
        {"source_fa": "باما", "price": 1_320_000_000, "mileage_km": 90_000, "vs_median_pct": 5.0},
    ]}
    text = explain.fallback(facts)
    assert "در عوض" not in text or "بیشتر" in text, text
    ok, nums, topics = explain.check(text, facts)
    assert ok, (nums, topics)


def test_a_real_trade_off_is_still_stated_as_one():
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 1_050_000_000, "mileage_km": 190_000, "vs_median_pct": -26.0},
        {"source_fa": "باما", "price": 1_320_000_000, "mileage_km": 40_000, "vs_median_pct": 5.0},
    ]}
    text = explain.fallback(facts)
    assert "در عوض" in text and "بیشتر" in text, text
    ok, nums, _ = explain.check(text, facts)
    assert ok, nums


def test_a_trivial_mileage_gap_is_not_dressed_up_as_a_reason():
    # A 41 km difference between two used cars is noise, not a decision factor.
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 1_050_000_000, "mileage_km": 100_000, "vs_median_pct": -26.0},
        {"source_fa": "باما", "price": 1_320_000_000, "mileage_km": 100_041, "vs_median_pct": 5.0},
    ]}
    text = explain.fallback(facts)
    assert "41" not in text and "۴۱" not in text, text
    ok, nums, _ = explain.check(text, facts)
    assert ok, nums


def test_an_unreliable_median_is_not_quoted_as_market_truth():
    facts = FACTS | {"median_reliable": False, "offer_count": 2, "offers": FACTS["offers"][:2]}
    text = explain.fallback(facts)
    assert "میانه" not in text.split(".")[0], text
    ok, nums, _ = explain.check(text, facts)
    assert ok, nums


def test_losing_the_best_mileage_is_caught():
    # Real model output: «کارکرد صفر کیلومتر را از دست می‌دهی» — you lose the
    # zero kilometres. Every number is true, so all four factual axes pass it.
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 828_000_000, "mileage_km": 0, "vs_median_pct": -55.7},
        {"source_fa": "باما", "price": 1_900_000_000, "mileage_km": 40_000, "vs_median_pct": 2.1},
    ]}
    text = "این خودرو با قیمت ۸۲۸,۰۰۰,۰۰۰ تومان ارزان‌ترین است. در عوض، کارکرد صفر کیلومتر را از دست می‌دهی."
    ok, nums, topics = explain.check(text, facts)
    assert not ok
    assert nums == []            # nothing was fabricated
    assert "loss_framing_on_mileage" in topics


def test_losing_the_cheapest_price_is_caught():
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 800_000_000, "mileage_km": 90_000, "vs_median_pct": -30.0},
        {"source_fa": "باما", "price": 1_200_000_000, "mileage_km": 40_000, "vs_median_pct": 5.0},
    ]}
    ok, _, topics = explain.check("در عوض ۸۰۰,۰۰۰,۰۰۰ تومان قیمت را از دست می‌دهی.", facts)
    assert not ok
    assert "loss_framing_on_price" in topics


def test_a_genuine_sacrifice_still_reads_as_one():
    # The top offer really is more worn here, so loss framing about mileage is
    # correct and must not be rejected.
    facts = FACTS | {"offers": [
        {"source_fa": "دیوار", "price": 800_000_000, "mileage_km": 180_000, "vs_median_pct": -30.0},
        {"source_fa": "باما", "price": 1_200_000_000, "mileage_km": 20_000, "vs_median_pct": 5.0},
    ]}
    text = "این خودرو ارزان‌ترین است. در عوض ۱۶۰,۰۰۰ کیلومتر بیشتر کار کرده و عمر لاستیک را از دست می‌دهی."
    ok, nums, topics = explain.check(text, facts)
    assert ok, (nums, topics)


def test_an_explanation_without_loss_framing_is_untouched():
    ok, _, topics = explain.check(
        "این خودرو ارزان‌ترین گزینه است و کمترین کارکرد را هم دارد.", FACTS)
    assert ok, topics

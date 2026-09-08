"""Every case here came out of the first real crawl."""
import plausibility as pl


def test_bama_1385_pride_with_zero_km_and_bodywork():
    # The listing that motivated this module.
    car = {"mileage_km": 0, "year_jalali": 1385, "body_status": "تعویض", "price_toman": 320_000_000}
    codes = {f["code"] for f in pl.flags(car)}
    assert "zero_km_old_car" in codes
    assert "zero_km_with_bodywork" in codes


def test_million_km_is_a_sentinel_not_a_reading():
    car = {"mileage_km": 1_000_000, "year_jalali": 1380, "body_status": None, "price_toman": 280_000_000}
    assert "mileage_sentinel" in {f["code"] for f in pl.flags(car)}


def test_a_genuinely_new_car_is_not_flagged():
    car = {"mileage_km": 0, "year_jalali": 1404, "body_status": "بدون رنگ", "price_toman": 1_175_000_000}
    assert pl.flags(car) == []


def test_ordinary_used_car_is_not_flagged():
    car = {"mileage_km": 136_438, "year_jalali": 1398, "body_status": None, "price_toman": 740_000_000}
    assert pl.flags(car) == []


def test_missing_price_is_reported():
    car = {"mileage_km": 50_000, "year_jalali": 1400, "body_status": None, "price_toman": None}
    assert "no_price" in {f["code"] for f in pl.flags(car)}


def test_an_allocation_certificate_is_not_a_car():
    # A «حواله» sat at 0.41 of its cluster median — just inside a 0.4 ratio
    # threshold — while the seller had written what it was in the title. A
    # stated fact beats a statistical guess about it.
    car = {"title": "حواله دنا پلاس دنده تحویل آبان و آذر", "mileage_km": 0,
           "year_jalali": 1405, "body_status": None, "price_toman": 1_025_000_000}
    codes = {f["code"] for f in pl.flags(car)}
    assert "not_a_car_yet" in codes


def test_presale_variants_are_caught():
    for title in ("پیش فروش شاهین", "پیش‌فروش تارا V4", "ثبت نام کوییک",
                  "مشارکت در تولید ساینا", "قرعه‌کشی پژو ۲۰۷"):
        car = {"title": title, "mileage_km": 0, "year_jalali": 1405,
               "body_status": None, "price_toman": 900_000_000}
        assert "not_a_car_yet" in {f["code"] for f in pl.flags(car)}, title


def test_an_ordinary_listing_is_not_flagged_as_an_allocation():
    # «تحویل فوری» is a normal thing to say about a car you have.
    car = {"title": "دنا پلاس ۶ دنده مدل ۱۴۰۵ تحویل فوری", "mileage_km": 1_200,
           "year_jalali": 1405, "body_status": "بدون رنگ", "price_toman": 2_440_000_000}
    assert pl.flags(car) == []


def test_divar_instalment_listing_is_flagged():
    """The asymmetry that let a fake bargain lead a live card.

    Bama declares instalment sales in a field and normalize.py has dropped them
    from the start. Divar declares them only in the title, so nothing caught
    them, and «فروش اقساطی سمند سورن» led a card at 55.8% below the median.
    """
    codes = [f["code"] for f in pl.flags({
        "title": "فروش اقساطی سمند سورن موتور پارس، مدل ۱۴۰۳",
        "price_toman": 828_000_000, "year": 1403, "mileage_km": 0,
    })]
    assert "instalment_price" in codes


def test_an_ordinary_listing_is_not_flagged_as_instalment():
    codes = [f["code"] for f in pl.flags({
        "title": "سمند سورن پلاس، مدل ۱۴۰۳", "price_toman": 1_800_000_000,
        "year": 1403, "mileage_km": 12_000,
    })]
    assert "instalment_price" not in codes


def test_sheypoor_state_sale_listings_are_flagged():
    """76 of 960 rows in one Sheypoor crawl, and none in the other four feeds.

    The number quoted is a registration figure, not a price: «فروش دولتی پراید»
    at 107,000,000 tomans against a real Pride near 300,000,000, «فروش سازمانی
    شاهین» at 846,000,000 against about 1.3 billion. Sheypoor publishes no
    price-type field for these the way Bama does for instalments, so the title
    is the only place the fact is stated — the same asymmetry that let Divar's
    instalment ads lead a card.
    """
    for title in ("فروش دولتی پراید", "پژو 405، فروش دولتی",
                  "فروش سازمانی شاهین", "سمند، فروش سازمانی"):
        car = {"title": title, "mileage_km": 0, "year_jalali": 1404,
               "body_status": None, "price_toman": 107_000_000}
        assert "off_market_sale" in {f["code"] for f in pl.flags(car)}, title


def test_a_car_that_was_once_a_company_car_is_not_flagged():
    # «خودروی سازمانی بوده» is an honest thing to say about a used car's
    # history, which is why the check matches the two-word phrase and not the
    # bare word.
    car = {"title": "پژو پارس مدل ۱۳۹۸، خودروی سازمانی بوده", "mileage_km": 90_000,
           "year_jalali": 1398, "body_status": "بدون رنگ", "price_toman": 780_000_000}
    assert pl.flags(car) == []


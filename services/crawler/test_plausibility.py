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

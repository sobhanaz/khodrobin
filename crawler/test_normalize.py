"""The cases that cost real money to get wrong."""
import normalize as n


def test_divar_rials_become_tomans():
    # A 1385 Pride: 3,100,000,000 rials on Divar is ~310M tomans, which matches
    # the same car at 320,000,000 tomans on Bama. Treating it as tomans would
    # report a 3-billion-toman Pride.
    assert n.price_toman("3100000000", unit="rial") == 310_000_000
    assert n.price_toman("320000000", unit="toman") == 320_000_000


def test_placeholder_prices_are_dropped():
    # Divar returns values this low for ads with no real price. Letting them
    # through would drag every market median down.
    assert n.price_toman("10000", unit="rial") is None
    assert n.price_toman(0, unit="toman") is None


def test_year_system_is_detected_not_assumed():
    assert n.split_year("1398") == (1398, 2019)      # domestic, Jalali
    assert n.split_year(2025) == (1404, 2025)        # imported, Gregorian
    assert n.split_year("۱۳۹۶") == (1396, 2017)      # Persian digits
    assert n.split_year(None) == (None, None)


def test_zero_mileage_is_prose_not_missing():
    assert n.mileage_km("صفر کیلومتر") == 0
    assert n.mileage_km("۱۲۰,۰۰۰") == 120_000
    assert n.mileage_km(None) is None


def test_damage_vocabulary_is_unified():
    assert n._lookup(n.BODY_STATUS, "half-paint") == "رنگ‌شدگی"
    assert n._lookup(n.BODY_STATUS, "بدون رنگ") == "بدون رنگ"
    assert n._lookup(n.TRANSMISSION, "manual") == "دنده‌ای"
    assert n._lookup(n.TRANSMISSION, "اتومات") == "اتوماتیک"

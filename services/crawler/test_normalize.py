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


def test_khodro45_gearbox_is_read_out_of_the_trim_field():
    # Khodro45 has no transmission field; it puts the gearbox in `trim`.
    # Missing this silos every Khodro45 listing into its own cluster.
    car = n.from_khodro45({
        "car_properties": {
            "brand": {"title": "جک", "title_en": "JAC"},
            "model": {"title": "J4", "title_en": "J4"},
            "trim": "اتوماتیک",
            "year": "1402",
        },
        "car_specifications": {"klm": 40000},
        "price": 1_500_000_000,
        "city": {"title": "تهران"},
        "slug": "ABC123",
    })
    assert car["transmission"] == "اتوماتیک"
    assert car["price_toman"] == 1_500_000_000
    assert car["year_jalali"] == 1402
    assert car["mileage_km"] == 40000


def test_khodro45_turbo_variant_still_reads_as_automatic():
    car = n.from_khodro45({
        "car_properties": {"brand": {"title_en": "MVM"}, "model": {"title_en": "X55"},
                           "trim": "اتوماتیک توربو", "year": "1403"},
        "car_specifications": {"klm": 10},
        "price": 3_000_000_000, "city": {"title": "تهران"}, "slug": "X",
    })
    assert car["transmission"] == "اتوماتیک"


def test_every_source_yields_an_image_url():
    # A car listing without a photo is a worse product, and all four sources
    # publish one. Dropping them was an oversight, so this pins it down.
    divar = n.from_divar({"name": "x", "image": "https://s100.divarcdn.com/a.webp",
                          "offers": {"price": "3100000000"}, "productionDate": "1385"})
    bama = n.from_bama({"detail": {"title": "x", "image": "https://cdn-sth1.bama.ir/b.jpg", "year": "1385"},
                        "price": {"price": "320,000,000"}})
    hamrah = n.from_hamrah({"carNamePersian": "x", "imageUrl": "https://cdn.hamrah-mechanic.com/c.jpg",
                            "price": 740_000_000, "carYear": 1398, "km": 100})
    k45 = n.from_khodro45({"car_properties": {"brand": {"title_en": "Kia"}, "model": {"title_en": "Rio"},
                                              "year": "1400"},
                           "car_specifications": {"klm": 10}, "price": 1_000_000_000,
                           "image": {"url": "https://media.khodro45.com/public/image/d"},
                           "city": {"title": "تهران"}, "slug": "S"})
    for car in (divar, bama, hamrah, k45):
        assert car["image"], car


def test_divar_real_damage_vocabulary_is_understood():
    # The original map guessed keys Divar never sends, discarding body condition
    # for 89% of its listings — «accidental» included.
    for raw, expected in [
        ("intact", "بدون رنگ"),
        ("some-scratches", "بدون رنگ"),
        ("some-paint", "رنگ‌شدگی"),
        ("paintless-dent-removal", "بدون رنگ"),
        ("accidental", "تصادفی"),
    ]:
        car = n.from_divar({"name": "x", "knownVehicleDamages": raw,
                            "offers": {"price": "3100000000"}, "productionDate": "1396"})
        assert car["body_status"] == expected, (raw, car["body_status"])


def test_bama_installment_totals_are_not_cash_prices():
    # The financed total is not comparable to an asking price and was pulling
    # cluster medians upward.
    ad = {"detail": {"title": "x", "year": "1400"},
          "price": {"type": "installment", "price": "3,000,000,000"}}
    assert n.from_bama(ad)["price_toman"] is None
    ad["price"]["type"] = "lumpsum"
    assert n.from_bama(ad)["price_toman"] == 3_000_000_000

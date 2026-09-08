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
    # A car listing without a photo is a worse product, and every source
    # publishes one. Dropping them was an oversight, so this pins it down.
    # Sheypoor is covered separately below: its `image` is a list of
    # ImageObject rather than a URL string, which is its own way to fail.
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


def test_unused_identifiers_never_reach_storage():
    # The seed snapshot is committed to a public repo. Neither field is secret,
    # but storing per-listing identifiers and neighbourhood locations the
    # product does not use is what decision ۷ says this project does not do.
    import run
    cleaned = run.scrub({
        "name": "پژو ۲۰۶",
        "vehicleIdentificationNumber": "P5V744CA4XS4L8MHV",
        "web_info": {"city_persian": "تهران", "district_persian": "افسریه شمالی"},
        "offers": {"price": "3100000000"},
    })
    assert "vehicleIdentificationNumber" not in cleaned
    assert "district_persian" not in cleaned["web_info"]
    # City stays: it is shown to the user and is what "where is this car" means.
    assert cleaned["web_info"]["city_persian"] == "تهران"
    assert cleaned["offers"]["price"] == "3100000000"


# --- Sheypoor ---------------------------------------------------------------
#
# Payloads below are verbatim from https://www.sheypoor.com/s/tehran/car, cut
# down to the fields the adapter reads.

def test_sheypoor_rials_reconcile_with_the_other_sources():
    # The whole reason this is a test and not a comment: Sheypoor labels its
    # price «IRR» and Bama labels nothing, and neither label is why we know.
    # A 1384 Pride quoted at 4,100,000,000 here sits beside a live 1385 Pride
    # at 310,000,000 tomans on Bama. Divided by ten they are the same car at
    # the same money; taken as tomans, Sheypoor is asking four billion tomans
    # for a forty-year-old Pride.
    pride84 = n.from_sheypoor({
        "name": "پراید84 خیلی خیلی خیلی تمیز",
        "brand": {"name": "پراید"},
        "vehicleModelDate": "1384",
        "offers": {"priceCurrency": "IRR", "Price": 4_100_000_000, "sku": 1},
        "mileageFromOdometer": {"value": "433000"},
    })
    assert pride84["price_toman"] == 410_000_000
    assert 300_000_000 <= pride84["price_toman"] <= 600_000_000


def test_sheypoor_price_key_is_capital_p():
    # schema.org spells it `price`; Sheypoor ships `Price` and omits the
    # lowercase key entirely. Reading the standard key gives None for every
    # listing, build_index drops every priceless row, and the source ships
    # looking integrated while contributing nothing.
    car = n.from_sheypoor({"name": "x", "brand": {"name": "پژو"},
                           "vehicleModelDate": "1396",
                           "offers": {"Price": 12_000_000_000, "sku": 2}})
    assert car["price_toman"] == 1_200_000_000


def test_sheypoor_mixes_year_systems_in_one_field():
    # Divar splits the two systems across two fields — Jalali in productionDate,
    # Gregorian in vehicleModelDate. Sheypoor puts whichever the seller used
    # into vehicleModelDate alone: a domestic Pride is 1382 and an imported
    # Mohave in the same 24-ad page is 2010.
    pride = n.from_sheypoor({"name": "پرایدمدل1382", "brand": {"name": "پراید"},
                             "vehicleModelDate": "1382",
                             "offers": {"Price": 3_500_000_000, "sku": 3}})
    mohave = n.from_sheypoor({"name": "موهاوی 8سیلندر 2010 فول", "brand": {"name": "کیا"},
                              "vehicleModelDate": "2010",
                              "offers": {"Price": 62_000_000_000, "sku": 4}})
    assert (pride["year_jalali"], pride["year_gregorian"]) == (1382, 2003)
    assert (mohave["year_jalali"], mohave["year_gregorian"]) == (1389, 2010)


def test_sheypoor_damage_vocabulary_is_its_own():
    # Sheypoor writes sentences, not slugs, and «تصادفی» — the one a buyer most
    # needs — was a value in BODY_STATUS but never a key, so it resolved to
    # None until this source arrived.
    for raw, expected in [
        ("سالم بدون خط و خش", "بدون رنگ"),
        ("سالم با خط و خش", "بدون رنگ"),
        ("یک لکه رنگ", "رنگ‌شدگی"),
        ("چند لکه رنگ", "رنگ‌شدگی"),
        ("تصادفی", "تصادفی"),
    ]:
        car = n.from_sheypoor({"name": "x", "brand": {"name": "پراید"},
                               "vehicleModelDate": "1390", "knownVehicleDamages": raw,
                               "offers": {"Price": 4_200_000_000, "sku": 5}})
        assert car["body_status"] == expected, (raw, car["body_status"])


def test_sheypoor_image_is_a_list_not_a_string():
    # Divar's `image` is a URL string; Sheypoor's is a list of ImageObject.
    # Passed through unchanged the front end gets a list where a URL belongs.
    car = n.from_sheypoor({
        "name": "x", "brand": {"name": "پژو"}, "vehicleModelDate": "1396",
        "offers": {"Price": 12_000_000_000, "sku": 6},
        "image": [{"@type": "ImageObject",
                   "contentUrl": "https://cdn.sheypoor.com/imgs/2026/08/08/1/a.webp"}],
    })
    assert car["image"] == "https://cdn.sheypoor.com/imgs/2026/08/08/1/a.webp"


def test_sheypoor_resolves_to_a_spec_without_a_model_field():
    # This source publishes no model and no trim — only a Persian brand and the
    # seller's title. If the title were dropped from the record, every Sheypoor
    # listing would key on brand alone, resolve to nothing, and quietly never
    # join another source's cluster. Same failure shape as the Khodro45 gearbox.
    from extract import resolve, spec_key
    car = n.from_sheypoor({
        "name": "پراید 131 SE مدل 98 بدون رنگ", "brand": {"name": "پراید"},
        "vehicleModelDate": "1398", "vehicleTransmission": "دنده‌ای",
        "mileageFromOdometer": {"value": "120000"},
        "offers": {"Price": 8_200_000_000, "sku": 7},
    })
    assert car["model_raw"] is None
    res = resolve(car)
    assert (res.brand, res.model) == ("pride", "131")
    assert spec_key(car, res) == "pride/131/base/mt/1398/4"


def test_sheypoor_source_id_survives_a_retitled_ad():
    # The ad URL embeds a slug built from the seller's title, and sellers edit
    # titles. Keying on the URL would turn one ad into two offers on the card;
    # dedup collapses on (source, source_id) and 40% of rows are re-crawls.
    from sources import sheypoor
    before = {"payload": {"offers": {"sku": 465632926},
                          "url": "https://www.sheypoor.com/v/موهاوی-2010-465632926.html"}}
    after = {"payload": {"offers": {"sku": 465632926},
                         "url": "https://www.sheypoor.com/v/موهاوی-2010-فول-465632926.html"}}
    assert sheypoor.source_id(before) == sheypoor.source_id(after) == "465632926"


def test_sheypoor_city_comes_from_the_page_not_the_ad():
    # A Sheypoor Vehicle carries no location at all; the city is a property of
    # the listing page, read out of its breadcrumb and attached at fetch time.
    from sources import sheypoor
    blocks = [{"@type": "BreadcrumbList", "itemListElement": [
        {"name": "خراسان رضوی", "item": "https://www.sheypoor.com/s/razavi-khorasan"},
        {"name": "مشهد", "item": "https://www.sheypoor.com/s/mashhad"},
        {"name": "وسایل نقلیه", "item": "https://www.sheypoor.com/s/mashhad/vehicles"},
    ]}]
    assert sheypoor._city_fa(blocks, "mashhad") == "مشهد"
    # «تهران» must not be answered by the province crumb /s/tehran-province.
    tehran = [{"@type": "BreadcrumbList", "itemListElement": [
        {"name": "استان تهران", "item": "https://www.sheypoor.com/s/tehran-province"},
        {"name": "تهران", "item": "https://www.sheypoor.com/s/tehran"},
    ]}]
    assert sheypoor._city_fa(tehran, "tehran") == "تهران"


def test_sheypoor_price_unit_belongs_to_the_source_not_to_the_number():
    """The 10x question, settled against the market rather than the label.

    A Lamari Eco quoted at 44,000,000,000 on Sheypoor is 4.4 billion tomans,
    which is what a new one costs; read as tomans it is a 44-billion-toman
    family sedan. The same figure arriving from Bama means the other thing, and
    nothing in either payload says so — Sheypoor writes ``priceCurrency: IRR``
    and Bama writes no currency at all. The unit is a property of the source.

    Held up against the rest of the shipped seed: 65 spec clusters carry both a
    Sheypoor offer and an offer from another source, and the median
    Sheypoor/other price ratio across them is 1.03. Read as tomans it is 10.3.
    """
    sheypoor = n.from_sheypoor({
        "name": "لاماری اکو خشک", "brand": {"name": "لاماری"},
        "vehicleModelDate": "1404", "vehicleTransmission": "اتوماتیک",
        "mileageFromOdometer": {"value": "600"},
        "offers": {"priceCurrency": "IRR", "Price": 44_000_000_000, "sku": 8},
    })
    bama = n.from_bama({
        "detail": {"title": "لاماری اکو", "year": "1404", "mileage": "600"},
        "price": {"price": 44_000_000_000},
    })
    assert sheypoor["price_toman"] == 4_400_000_000
    assert bama["price_toman"] == 44_000_000_000
    assert bama["price_toman"] == 10 * sheypoor["price_toman"]


def test_sheypoor_year_systems_both_reach_the_same_spec_key():
    """846 Jalali to 108 Gregorian in one 960-ad crawl, in the same field.

    The Gregorian ones are the imports — Tucson, Land Cruiser, Megane — and
    they are exactly the listings that need to cluster with a domestic feed's
    Jalali row for the same car. Convert one path and not the other and an
    imported Tucson gets its own private cluster with a median of one.
    """
    from extract import resolve, spec_key
    payload = {
        "name": "توسان 2013 دودیفرانسیل 2400cc", "brand": {"name": "هیوندای"},
        "vehicleTransmission": "اتوماتیک",
        "mileageFromOdometer": {"value": "226000"},
        "offers": {"Price": 47_000_000_000, "sku": 9},
    }
    imported = n.from_sheypoor(payload | {"vehicleModelDate": "2013"})
    domestic = n.from_sheypoor(payload | {"vehicleModelDate": "1392"})
    assert (imported["year_jalali"], imported["year_gregorian"]) == (1392, 2013)
    assert (domestic["year_jalali"], domestic["year_gregorian"]) == (1392, 2013)
    key = spec_key(imported, resolve(imported))
    assert key == "hyundai/tucson/base/at/1392/9"
    assert key == spec_key(domestic, resolve(domestic))


def test_sheypoor_gearbox_is_its_own_field_not_a_trim():
    """Where Khodro45 hid the gearbox in the trim, Sheypoor states it.

    Worth an assertion anyway, because the failure is silent: a source whose
    gearbox never parses keys every spec on ``na`` and forms a parallel
    universe that no other source's cluster can ever join. Across the 422
    indexed rows of the shipped crawl the split is 324 mt, 97 at, 1 na.
    """
    from extract import resolve, spec_key
    for stated, expected in (("اتوماتیک", "at"), ("دنده‌ای", "mt")):
        car = n.from_sheypoor({
            "name": "پژو 207 مدل 1404", "brand": {"name": "پژو"},
            "vehicleModelDate": "1404", "vehicleTransmission": stated,
            "mileageFromOdometer": {"value": "5000"},
            "offers": {"Price": 23_600_000_000, "sku": 10},
        })
        assert spec_key(car, resolve(car)).split("/")[3] == expected, stated

"""Cases taken from real captured titles across all three sources."""
import sys, pathlib
sys.path.insert(0, str(pathlib.Path(__file__).parent))
from extract import fold, resolve, spec_key


def r(**kw):
    base = {"brand_raw": None, "model_raw": None, "trim_raw": None, "title": None}
    return resolve(base | kw)


def test_three_vocabularies_reach_one_key():
    divar = r(brand_raw="Peugeot", model_raw="Peugeot 206 5", title="۲۰۶ تیپ ۵ مدل ۹۷")
    bama = r(brand_raw="پژو", model_raw="peugeot", trim_raw="تیپ 5", title="پژو، 206")
    hamrah = r(brand_raw="peugeot", model_raw="206", trim_raw="تیپ ۵ اتوماتیک")
    for res in (divar, bama, hamrah):
        assert res.brand == "peugeot", res
        assert res.model == "206", res
        assert res.trim == "type-5", res


def test_206_sd_is_not_a_206():
    assert r(title="پژو، 206 SD").model == "206-sd"
    assert r(title="پژو 206").model == "206"


def test_corolla_cross_is_not_a_corolla():
    # These were clustering together and averaging a 5B hybrid with a 9.3B crossover.
    assert r(title="تویوتا، کرولا کراس هیبرید").model == "corolla-cross"
    assert r(title="تویوتا، کرولا هیبرید").model == "corolla"


def test_persian_digits_and_arabic_forms_fold():
    assert fold("۲۰۶ تيپ ۵") == "206 تیپ 5"
    assert r(title="پژو ۲۰۶ تيپ ۵").trim == "type-5"


def test_bare_numbers_need_word_boundaries():
    # "206" must not match inside a price or a longer number.
    assert r(brand_raw="Peugeot", title="پژو پارس قیمت 2065000000").model != "206"


def test_unresolved_when_model_is_missing():
    res = r(brand_raw="Peugeot", title="پژو در حد صفر")
    assert res.brand == "peugeot" and res.model is None
    assert res.method == "unresolved"


def test_spec_key_buckets_mileage_and_keeps_gearbox():
    car = {"year_jalali": 1396, "mileage_km": 118_400, "transmission": "اتوماتیک"}
    res = r(brand_raw="Peugeot", model_raw="206", trim_raw="تیپ ۵")
    assert spec_key(car, res) == "peugeot/206/type-5/at/1396/4"


def test_transmission_does_not_shadow_the_trim():
    # «تیپ ۵ اتوماتیک» carries two facts. Folding them into one field let the
    # longer word "اتوماتیک" win and lose the trim entirely.
    res = r(brand_raw="peugeot", model_raw="206", trim_raw="تیپ ۵ اتوماتیک")
    assert res.trim == "type-5"

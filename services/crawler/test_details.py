"""What a detail fetch is allowed to add, and what it must never touch.

Fixtures under testdata/details/ are real pages fetched while this module was
written, trimmed to the script tags the extractors actually read (plus Divar's
CDN image URLs, which live in the body) and with every phone-shaped digit run
zeroed. Trimmed because a full Hamrah-Mechanic page is 450 KB of markup that
tests nothing; scrubbed because one of them shipped an unmasked seller mobile in
`carInformation.consultantPhone` and this repo is public.
"""
import json
import pathlib
from datetime import datetime, timedelta, timezone

import httpx
import pytest

import details
import enrich_details
from politeness import HostLimiter

FIXTURES = pathlib.Path(__file__).parent / "testdata" / "details"
NOW = datetime(2026, 9, 9, 12, 0, tzinfo=timezone.utc)


def page(name: str) -> str:
    return (FIXTURES / name).read_text(encoding="utf-8")


class Instant(HostLimiter):
    """The real breaker, without the sleeping. The per-host cadence is
    politeness.py's contract and is tested there; what these tests are about is
    what counts as a strike."""

    def wait(self, host: str) -> None:
        pass


# --------------------------------------------------------------------------
# One extractor per source, against a page that source really served.
# --------------------------------------------------------------------------

def test_divar_reads_the_car_json_ld():
    d = details.extract("divar", page("divar.html"))
    assert d["description"].startswith("فول ۸۵ گانه")
    assert d["color"] == "سفید"
    # "intact" is Divar's own vocabulary; a Persian page must not print it.
    assert d["damages"] == "بدون رنگ"
    assert d["condition"] == "نو"
    # Six distinct photos, not one photo six ways: the same picture is served
    # as /post/*.jpg and /webp_post/*.webp with different path tokens.
    assert len(d["images"]) == 6
    assert len({u.rsplit("/", 1)[-1].split(".")[0] for u in d["images"]}) == 6


def test_bama_reads_the_product_car_json_ld():
    d = details.extract("bama", page("bama.html"))
    # Bama prefixes the seller's words with a generated SEO sentence.
    assert not d["description"].startswith("خرید خودرو")
    assert "پلاس_شخصی" in d["description"]
    assert d["fuel"] == "بنزینی"
    assert d["body"] == "هاچبک"          # published as "hatchback"
    assert d["engine"] == "۱.۶ لیتر"      # published as {value: 1.6, unitCode: LTR}
    assert d["extras"]["شتاب ۰ تا ۱۰۰"] == "۱۱.۶ ثانیه"


def test_sheypoor_digs_into_main_entity():
    """The car is under mainEntity of an ItemPage, not at the top level."""
    d = details.extract("sheypoor", page("sheypoor.html"))
    assert d["description"].startswith("برای معاوضه")
    assert d["damages"] == "سالم بدون خط و خش"
    assert d["fuel"] == "بنزین"           # nested under vehicleEngine
    assert d["body"] == "هاچ بک"
    assert len(d["images"]) == 4
    # No displacement is published here. Saying so beats inventing one.
    assert d["engine"] is None


def test_hamrah_reads_next_data_and_the_inspection():
    d = details.extract("hamrah", page("hamrah.html"))
    assert d["color"] == "سفید"
    assert d["fuel"] == "بنزینی"
    assert d["extras"]["سند"] == "تک برگی"
    report = d["inspection"]
    assert [s["title"] for s in report["sections"]][0] == "بدنه و شاسی"
    assert all(s["score"] is not None for s in report["sections"])
    # Kept parts are the ones with something to say. A part marked perfect that
    # still carries «امکان کارشناسی نداشت» is the inspector saying he could not
    # check it, and dropping it would turn unknown into fine.
    assert any(p["note"] for p in report["parts"])
    assert len(report["parts"]) < 40      # ~90 raw parts, distilled


def test_hamrah_description_is_the_sellers_half_or_nothing():
    """Measured on six live listings: `descriptionPart` is a fixed platform
    sentence about Hamrah's own damage cover, with the seller's note appended
    when there is one. Left whole it would be the same paragraph on every card.
    """
    assert details.extract("hamrah", page("hamrah.html"))["description"] is None
    real = ("این خودرو توسط همراه مکانیک کارشناسی شده است و تحت پوشش ضمانت تامین "
            "خسارت تا سقف 800 میلیون تومان است. *خودرو به صورت دستی دوگانه سوز شده است* ")
    assert details._HAMRAH_BOILERPLATE.sub("", real) == "*خودرو به صورت دستی دوگانه سوز شده است* "


def test_khodro45_extractor_works_but_is_never_fetched():
    """robots.txt disallows /view/ (the page, a data-free React shell) and /api
    (the only copy of the payload). The extractor is kept and tested so the
    finding is reproducible; the fetch plan leaves the source off.
    """
    d = details.extract("khodro45", page("khodro45.json"))
    assert d["body"] == "سدان"
    assert len(d["images"]) == 8
    assert enrich_details.SOURCES["khodro45"]["enabled"] is False
    assert "robots" in enrich_details.SOURCES["khodro45"]["why"]


# --------------------------------------------------------------------------
# THE RULE. Divar's list page publishes rials and its detail page publishes
# tomans, both labelled IRR — ad gadOXBRG quotes 21,300,000,000 on the list and
# 2,130,000,000 on the detail. The index resolved units per source by pricing a
# real car; anything this module wrote back would move medians by 10x while
# every field name still looked correct.
# --------------------------------------------------------------------------

# (fixture, the price it publishes, the mileage it publishes)
PRICED = [("divar.html", "divar", "1900000000", "0"),
          ("bama.html", "bama", "2170000000", None),
          ("sheypoor.html", "sheypoor", "23600000000", "64"),
          ("hamrah.html", "hamrah", "1330000000", "63"),
          ("khodro45.json", "khodro45", "2029500000", "180775")]


@pytest.mark.parametrize("fixture,source,price,mileage", PRICED)
def test_no_extractor_returns_a_price_or_a_mileage(fixture, source, price, mileage):
    d = details.extract(source, page(fixture))
    # The shape is closed: exactly the contract's keys, so a price cannot arrive
    # under a name nobody thought to forbid.
    assert set(d) == set(details.KEYS)
    body = json.dumps(d, ensure_ascii=False)
    assert price not in body
    assert f"{int(price):,}" not in body
    if mileage and len(mileage) > 2:
        # Short values like "0" and "64" appear inside unrelated numbers; only
        # a mileage distinctive enough to be evidence is asserted on.
        assert mileage not in body


def test_extras_cannot_smuggle_a_price_back_in():
    """extras is the free-form slot, which makes it the one way round the closed
    shape. The guard is in the single constructor rather than in five sources.
    """
    d = details._detail(color="سفید", extras={"قیمت نقدی": "۲٬۱۰۰٬۰۰۰٬۰۰۰",
                                              "price": "2100000000",
                                              "کارکرد واقعی": "۱۲٬۰۰۰",
                                              "رنگ داخلی": "مشکی"})
    assert d["extras"] == {"رنگ داخلی": "مشکی"}


# --------------------------------------------------------------------------
# Nothing, junk and soft blocks.
# --------------------------------------------------------------------------

@pytest.mark.parametrize("source", sorted(details.EXTRACTORS))
@pytest.mark.parametrize("body", [
    "",
    "<html><body>سلام</body></html>",
    '<html><script type="application/ld+json">{not json</script></html>',
    '<html><script type="application/ld+json">{"@type":"WebPage"}</script></html>',
    '<script id="__NEXT_DATA__">{"props":{}}</script>',
])
def test_a_page_with_no_structured_data_yields_nothing_rather_than_raising(source, body):
    """A soft block, a sold listing and a redirect to the homepage all arrive as
    text. One malformed page must not abort a 900-second run.
    """
    assert details.extract(source, body) is None


def test_all_null_page_is_not_stored_as_a_successful_fetch():
    """Ten nulls would sit in the file for the whole TTL looking like a hit."""
    assert details._detail() is None
    assert details._detail(color="سفید") is not None


def test_a_soft_block_backs_the_host_off():
    """Divar answers 200 with the JSON-LD absent when we ask too fast, so the
    status code is not a health signal — yield is. Three empties open the
    breaker and the host is left alone.
    """
    limiter = HostLimiter(breaker_threshold=3, breaker_cooldown=60)
    for _ in range(3):
        assert not limiter.is_open("divar.ir")
        limiter.record_empty("divar.ir")
    assert limiter.is_open("divar.ir")


def test_an_empty_200_is_counted_as_empty_and_stops_the_source(monkeypatch, tmp_path):
    """End to end: every request answers 200 with no data, and the run must
    record it as empty, back off, and leave the previous file's entries alone.
    """
    served = []

    def handler(request):
        served.append(str(request.url))
        return httpx.Response(200, text="<html><body>هیچ</body></html>")

    real_client = httpx.Client   # captured before the patch, or the lambda recurses
    monkeypatch.setattr(httpx, "Client",
                        lambda **kw: real_client(transport=httpx.MockTransport(handler), **kw))
    monkeypatch.setattr(enrich_details, "HostLimiter", Instant)

    index = tmp_path / "index.json"
    index.write_text(json.dumps({"specs": [spec_with(12)]}), encoding="utf-8")
    out = tmp_path / "details.json"
    out.write_text(json.dumps({"details": {"divar:old-0": {
        "description": "از دور قبل", "fetched_at": "2026-09-01T00:00:00+00:00"}}}),
        encoding="utf-8")

    enrich_details.enrich(index, out, limit=10, budget_seconds=30, ttl_days=7)

    # Twelve offers, but the breaker opens after three empty 200s — which only
    # works because the success reset is reported after the body is read. An
    # earlier version called record(200) first, and every empty strike was
    # cancelled by the next request's status: nine attempts, breaker never open.
    assert len(served) == 3
    # And the entry from the previous cycle is untouched: a host that yields
    # nothing must not blank what it wrote when it was working.
    stored = json.loads(out.read_text(encoding="utf-8"))["details"]
    assert stored["divar:old-0"]["description"] == "از دور قبل"


def spec_with(n: int, source: str = "divar") -> dict:
    return {"key": "peugeot/207/base/mt/1404/0", "source_count": 1, "offer_count": n,
            "offers": [{"source": source, "source_id": f"old-{i}",
                        "url": f"https://divar.ir/v/x/old-{i}"} for i in range(n)]}


# --------------------------------------------------------------------------
# Selectivity: the TTL, the shortlist, and what survives a cycle.
# --------------------------------------------------------------------------

def test_an_entry_newer_than_the_ttl_is_not_refetched():
    """A seller's paragraph does not change every three hours, and 8,800 offers
    on a three-hour cycle is 70,000 requests a day.
    """
    specs = [spec_with(1)]
    stored = {"divar:old-0": {"fetched_at": (NOW - timedelta(days=2)).isoformat()}}
    assert enrich_details.plan(specs, stored, timedelta(days=7), NOW) == []

    stored = {"divar:old-0": {"fetched_at": (NOW - timedelta(days=8)).isoformat()}}
    assert [j["key"] for j in enrich_details.plan(specs, stored, timedelta(days=7), NOW)] \
        == ["divar:old-0"]


def test_an_unreadable_timestamp_is_refetched_rather_than_trusted_forever():
    stored = {"divar:old-0": {"fetched_at": "some time last week"}}
    assert len(enrich_details.plan([spec_with(1)], stored, timedelta(days=7), NOW)) == 1


def test_the_disabled_source_is_never_planned():
    assert enrich_details.plan([spec_with(3, "khodro45")], {}, timedelta(days=7), NOW) == []


def test_every_spec_gets_its_first_offer_before_any_gets_its_second():
    """Budget exhaustion is the normal case, so it has to degrade sideways: 60
    cards with one detailed offer each, not 8 cards finished and 52 empty.
    """
    specs = []
    for n in range(3):
        spec = spec_with(3)
        for i, offer in enumerate(spec["offers"]):
            offer["source_id"] = f"s{n}-{i}"
        specs.append(spec)
    order = [j["key"] for j in enrich_details.plan(specs, {}, timedelta(days=7), NOW)]
    assert order[:3] == ["divar:s0-0", "divar:s1-0", "divar:s2-0"]


def test_a_round_is_spread_across_hosts_rather_than_hammering_one():
    """Divar makes us wait 4 seconds between its own requests. That wait is free
    if somebody else's request happens inside it, so a round alternates hosts.
    """
    def spec(source_a, source_b):
        return {"key": f"{source_a}{source_b}", "source_count": 2, "offer_count": 2,
                "offers": [{"source": source_a, "source_id": f"{source_a}1", "url": "u"},
                           {"source": source_b, "source_id": f"{source_b}2", "url": "u"}]}

    order = [j["source"] for j in enrich_details.plan(
        [spec("divar", "bama"), spec("bama", "divar"), spec("sheypoor", "hamrah")],
        {}, timedelta(days=7), NOW)]
    # Round 0 holds divar, bama and sheypoor; no host is asked twice in a row.
    assert order[:3] == ["divar", "bama", "sheypoor"]
    assert all(a != b for a, b in zip(order, order[1:]))


def test_an_offer_that_left_the_index_expires(tmp_path, monkeypatch):
    """Otherwise the file accretes detail panels for ads nobody can open."""
    monkeypatch.setattr(enrich_details, "HostLimiter", Instant)
    index = tmp_path / "index.json"
    index.write_text(json.dumps({"specs": []}), encoding="utf-8")
    out = tmp_path / "details.json"
    out.write_text(json.dumps({"details": {"divar:gone": {
        "description": "x", "fetched_at": NOW.isoformat()}}}), encoding="utf-8")

    enrich_details.enrich(index, out, limit=10, budget_seconds=0, ttl_days=7)
    assert json.loads(out.read_text(encoding="utf-8"))["details"] == {}


def test_an_offer_outside_the_shortlist_keeps_the_detail_it_already_had(tmp_path, monkeypatch):
    """The top-N window moves every cycle. A card dropping from rank 59 to 61
    must not lose its panel — it is still in the index and still reachable.
    """
    monkeypatch.setattr(enrich_details, "HostLimiter", Instant)
    index = tmp_path / "index.json"
    index.write_text(json.dumps({"specs": [spec_with(1)]}), encoding="utf-8")
    out = tmp_path / "details.json"
    out.write_text(json.dumps({"details": {"divar:old-0": {
        "description": "هنوز معتبر", "fetched_at": NOW.isoformat()}}}), encoding="utf-8")

    enrich_details.enrich(index, out, limit=0, budget_seconds=0, ttl_days=7)
    stored = json.loads(out.read_text(encoding="utf-8"))["details"]
    assert stored["divar:old-0"]["description"] == "هنوز معتبر"


def test_a_corrupt_details_file_is_started_over_rather_than_crashing(tmp_path):
    out = tmp_path / "details.json"
    out.write_text("{truncated", encoding="utf-8")
    assert enrich_details.load_existing(out) == {}


def test_a_missing_index_is_not_an_error(tmp_path, monkeypatch):
    """First deploy: the file does not exist and the cycle must not fail."""
    monkeypatch.setattr("sys.argv", ["enrich_details.py",
                                     "--index", str(tmp_path / "nope.json")])
    assert enrich_details.main() == 0


# --------------------------------------------------------------------------
# The field this project deleted once already.
# --------------------------------------------------------------------------

def test_a_phone_number_in_a_description_is_redacted():
    """run.py drops `description` from every raw payload because sellers put
    their mobile in it and the seed snapshot is public. This module brings the
    field back, so it carries that decision with it.
    """
    d = details._detail(description="فروش فوری\nتماس ۰۹۱۲۵۲۳۷۰۱۱ فقط پیامک")
    assert "۰۹۱۲۵۲۳۷۰۱۱" not in d["description"]
    assert "فروش فوری" in d["description"]
    assert details._redact("تماس: 0912 523 7011") == "تماس: [شماره تماس حذف شد]"


def test_redaction_leaves_a_price_written_in_the_description_alone():
    """«مبلغ 2.100» and «۲۱۳۰۰۰۰۰۰۰» are the seller's own words, and mangling
    them would be a worse lie than leaving them — a price never starts with 0.
    """
    for text in ["مبلغ 2.100 تومان", "قیمت ۲۱۳۰۰۰۰۰۰۰", "مدل 1404", "کارکرد 63000"]:
        assert details._redact(text) == text


def test_intermittent_soft_block_still_opens_the_breaker(monkeypatch, tmp_path):
    """The shipped test only covered an all-empty source, which is not the shape
    Divar's soft block actually takes: it interleaves real pages with shells.

    Success used to call limiter.record(host, 200), which zeroes the strike
    counter, so one good page erased the evidence of the empty one before it and
    the breaker could never open. Forty requests with twenty soft blocks trips
    nothing under the old code.
    """
    from politeness import HostLimiter

    limiter = HostLimiter()
    host = "divar.ir"
    # Alternate: empty, extracted, empty, extracted... Only the empties vote.
    for i in range(10):
        if i % 2 == 0:
            limiter.record_empty(host)
        # a successful extraction reports nothing, which is the fix
    assert limiter.is_open(host), (
        "an alternating success/empty run never opened the breaker; "
        "success is resetting the strike counter again"
    )

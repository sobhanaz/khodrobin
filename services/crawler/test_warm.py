"""The warm cache is only as honest as what it considers stale."""
import httpx
import pytest

from warm_explanations import guard_version, spec_fingerprint

SPEC = {"median_price": 500_000_000, "offer_count": 7,
        "min_price": 420_000_000, "max_price": 610_000_000}


def test_price_change_expires_an_explanation():
    assert spec_fingerprint(SPEC, "g1") != spec_fingerprint(SPEC | {"median_price": 510_000_000}, "g1")


def test_guard_change_expires_an_explanation():
    """The bug this exists for: identical data, tightened rules.

    Nine of 93 live explanations used loss framing on the very thing the top
    offer was best at. The coherence axis catches them now — but every one was
    still being served, because the fingerprint only ever asked whether the
    prices had moved.
    """
    assert spec_fingerprint(SPEC, "g1") != spec_fingerprint(SPEC, "g2")


def test_same_data_same_guard_is_reused():
    assert spec_fingerprint(SPEC, "g1") == spec_fingerprint(dict(SPEC), "g1")


def test_unreachable_ai_service_reuses_rather_than_regenerates():
    """A health check that fails is not a reason to burn the whole budget."""
    def down(request):
        raise httpx.ConnectError("no route")

    with httpx.Client(transport=httpx.MockTransport(down), base_url="http://ai") as c:
        assert guard_version(c) == ""


@pytest.mark.parametrize("body", [{"status": "ok"}, {"guard_version": "abc123"}])
def test_guard_version_read_from_health(body):
    with httpx.Client(transport=httpx.MockTransport(lambda r: httpx.Response(200, json=body)),
                      base_url="http://ai") as c:
        assert guard_version(c) == body.get("guard_version", "")

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


def test_entries_outside_the_limit_expire_when_the_guard_changes(tmp_path):
    """The carry-forward loop used to keep every prior entry unconditionally.

    Its comment said "already known and still valid" while checking nothing, so
    37 live explanations written under an older guard could never expire: the
    fingerprint was only ever consulted for specs inside the top-N window.
    """
    import json
    import warm_explanations as w

    spec = {"key": "outside", "median_price": 500_000_000, "offer_count": 7,
            "min_price": 420_000_000, "max_price": 610_000_000,
            "source_count": 1}
    index = tmp_path / "index.json"
    index.write_text(json.dumps({"specs": [spec]}), encoding="utf-8")

    out = tmp_path / "explanations.json"
    out.write_text(json.dumps({"explanations": {"outside": {
        "text": "قدیمی", "source": "llm",
        # Written under an older guard: four fields, no guard version.
        "fingerprint": "500000000:7:420000000:610000000",
    }}}), encoding="utf-8")

    # limit=0 puts the spec outside the window entirely, which is exactly the
    # case the old loop could not reach.
    w.warm(index, out, "http://ai.invalid", limit=0, budget_seconds=0)
    assert json.loads(out.read_text(encoding="utf-8"))["explanations"] == {}


def test_entry_for_a_spec_that_left_the_index_is_dropped(tmp_path):
    """Otherwise the file accretes answers about cars nobody can look up."""
    import json
    import warm_explanations as w

    index = tmp_path / "index.json"
    index.write_text(json.dumps({"specs": []}), encoding="utf-8")
    out = tmp_path / "explanations.json"
    out.write_text(json.dumps({"explanations": {"gone": {
        "text": "x", "source": "llm", "fingerprint": "1:1:1:1:abc"}}}), encoding="utf-8")

    w.warm(index, out, "http://ai.invalid", limit=10, budget_seconds=0)
    assert json.loads(out.read_text(encoding="utf-8"))["explanations"] == {}

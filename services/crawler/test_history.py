"""The chart can only ever show what this file agreed to keep."""
import json
from datetime import datetime, timedelta, timezone

from record_history import (MAX_POINTS, MIN_GAP_SECONDS, load_history, record,
                            save)

T0 = datetime(2026, 9, 8, 12, 0, tzinfo=timezone.utc)


def index_at(t: datetime, median=500_000_000, offers=5, reliable=True, key="peugeot/207/base/mt/1404/0"):
    return {"built_at": t.isoformat(),
            "specs": [{"key": key, "median_price": median,
                       "offer_count": offers, "median_reliable": reliable}]}


def cycles(n: int, hist=None):
    hist = hist if hist is not None else {}
    for i in range(n):
        t = T0 + timedelta(hours=3 * i)
        record(hist, index_at(t, median=500_000_000 + i), t)
    return hist


def test_one_point_per_cycle():
    hist = cycles(3)
    points = hist["peugeot/207/base/mt/1404/0"]
    assert len(points) == 3
    assert [p["m"] for p in points] == [500_000_000, 500_000_001, 500_000_002]
    assert points[0]["n"] == 5


def test_same_index_recorded_twice_appends_nothing():
    """schedule.py records after the startup rebuild and after every cycle; a
    restart must not turn one index into two points."""
    hist = {}
    idx = index_at(T0)
    record(hist, idx, T0)
    record(hist, idx, T0 + timedelta(hours=1))
    assert len(hist["peugeot/207/base/mt/1404/0"]) == 1


def test_restart_churn_within_the_gap_appends_nothing():
    hist = {}
    record(hist, index_at(T0), T0)
    soon = T0 + timedelta(seconds=MIN_GAP_SECONDS - 1)
    record(hist, index_at(soon), soon)
    assert len(hist["peugeot/207/base/mt/1404/0"]) == 1


def test_cap_drops_the_oldest():
    hist = cycles(MAX_POINTS + 3)
    points = hist["peugeot/207/base/mt/1404/0"]
    assert len(points) == MAX_POINTS
    # The three oldest medians are gone, the newest survives.
    assert points[0]["m"] == 500_000_003
    assert points[-1]["m"] == 500_000_000 + MAX_POINTS + 2


def test_unreliable_median_appends_nothing():
    hist = {}
    stats = record(hist, index_at(T0, offers=2, reliable=False), T0)
    assert hist == {} and stats["appended"] == 0


def test_missing_median_appends_nothing():
    idx = {"built_at": T0.isoformat(),
           "specs": [{"key": "x", "median_reliable": True, "median_price": None}]}
    hist = {}
    record(hist, idx, T0)
    assert hist == {}


def test_corrupt_file_starts_fresh_and_keeps_the_bytes(tmp_path):
    path = tmp_path / "history.json"
    path.write_text('{"specs": {"k": [truncated', encoding="utf-8")
    assert load_history(path) == {}
    # The unreadable bytes were moved aside, not destroyed.
    assert (tmp_path / "history.json.corrupt").exists()
    assert not path.exists()


def test_valid_json_wrong_shape_starts_fresh(tmp_path):
    path = tmp_path / "history.json"
    path.write_text('{"specs": "not a map"}', encoding="utf-8")
    assert load_history(path) == {}


def test_spec_absent_60_days_is_dropped_5_days_is_kept():
    old = [{"t": (T0 - timedelta(days=61)).isoformat(), "m": 1, "n": 3}]
    recent = [{"t": (T0 - timedelta(days=5)).isoformat(), "m": 2, "n": 3}]
    hist = {"gone/60d": list(old), "gone/5d": list(recent)}
    record(hist, {"built_at": T0.isoformat(), "specs": []}, T0)
    assert "gone/60d" not in hist
    assert hist["gone/5d"] == recent  # untouched, not just present


def test_live_spec_keeps_history_even_when_unreliable_today():
    """Presence in today's index is a sighting; only a reliable median is a
    point. A spec limping along at two offers must not age out of the chart."""
    stale = [{"t": (T0 - timedelta(days=61)).isoformat(), "m": 1, "n": 3}]
    hist = {"peugeot/207/base/mt/1404/0": list(stale)}
    record(hist, index_at(T0, offers=2, reliable=False), T0)
    assert hist["peugeot/207/base/mt/1404/0"] == stale


def test_save_and_reload_round_trip(tmp_path):
    path = tmp_path / "history.json"
    hist = cycles(2)
    save(hist, path, T0)
    data = json.loads(path.read_text(encoding="utf-8"))
    assert data["updated_at"] == T0.isoformat()
    assert load_history(path) == hist
    assert not path.with_suffix(".json.tmp").exists()

"""Evaluate Persian query understanding against a hand-labelled golden set.

Two decisions shape this file.

**It measures the deployed system, not a library.** The harness sends each query
to a running ``/api/v1/search`` and grades the ``intent`` that comes back. A unit
test of a parser can pass while the thing users actually reach is broken — by a
stale index, a bad deploy, a routing change. Grading the endpoint means the
number always describes the product.

**It works before any model exists.** Today the endpoint answers with the
deterministic rules parser, so this produces the baseline the LLM path has to
beat. When the AI service lands, the same file grades it against the same 50
queries, and the comparison is honest because nothing else changed.

Grading is strict per field. A query counts as correct only when every field the
golden set specifies matches exactly; partial credit is reported separately as
per-field accuracy, because "we got the brand but lost the year" is a different
failure from "we got nothing".

Usage
-----
    python run_eval.py --api https://khodrobin.noxioai.com
    python run_eval.py --api http://localhost:8080 --threshold 90 --json report.json
"""
from __future__ import annotations

import argparse
import json
import pathlib
import statistics
import sys
import time
from collections import Counter

import httpx
from dataclasses import dataclass, field

HERE = pathlib.Path(__file__).parent
GOLDEN = HERE / "golden_intents.jsonl"

# Fields a golden case may pin. Anything absent from a case is not graded, so a
# case can assert "brand and year" without also having to spell out the price.
GRADED_FIELDS = (
    "brand", "model", "year_min", "year_max",
    "price_min", "price_max", "gearbox", "priority",
)


@dataclass
class Case:
    q: str
    expect: dict
    note: str = ""


@dataclass
class Outcome:
    case: Case
    got: dict
    latency_ms: float
    mismatches: list[tuple[str, object, object]] = field(default_factory=list)
    error: str | None = None

    @property
    def ok(self) -> bool:
        return self.error is None and not self.mismatches


def load_cases(path: pathlib.Path) -> list[Case]:
    cases: list[Case] = []
    with path.open(encoding="utf-8") as fh:
        for n, line in enumerate(fh, 1):
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as exc:
                raise SystemExit(f"{path}:{n}: {exc}") from exc
            cases.append(Case(q=row["q"], expect=row["expect"], note=row.get("note", "")))
    return cases


def query(client: httpx.Client, api: str, q: str) -> tuple[dict, float]:
    """Ask the running API what it understood.

    httpx rather than urllib because it ships a CA bundle. Verification stays
    on: a harness that reports accuracy while silently trusting any certificate
    is measuring the wrong thing.
    """
    started = time.perf_counter()
    resp = client.get(api.rstrip("/") + "/api/v1/search",
                      params={"q": q, "mode": "relevant", "limit": 1})
    resp.raise_for_status()
    elapsed = (time.perf_counter() - started) * 1000
    return resp.json()["result"]["intent"], elapsed


def grade(case: Case, got: dict) -> list[tuple[str, object, object]]:
    """Return the fields that disagree, as (field, expected, actual)."""
    bad: list[tuple[str, object, object]] = []
    for key, want in case.expect.items():
        if key not in GRADED_FIELDS:
            raise SystemExit(f"golden set pins an ungraded field {key!r} on {case.q!r}")
        # The API omits zero values, so "expected 0" means "must be absent".
        actual = got.get(key, 0 if isinstance(want, int) else "")
        if actual != want:
            bad.append((key, want, actual))
    return bad


def run(api: str, cases: list[Case], timeout: float) -> list[Outcome]:
    out: list[Outcome] = []
    with httpx.Client(timeout=timeout, follow_redirects=True) as client:
        for case in cases:
            try:
                got, ms = query(client, api, case.q)
            except Exception as exc:  # noqa: BLE001 - one bad query must not end the run
                out.append(Outcome(case=case, got={}, latency_ms=0.0, error=repr(exc)))
                continue
            out.append(Outcome(case=case, got=got, latency_ms=ms,
                               mismatches=grade(case, got)))
    return out


def report(outcomes: list[Outcome], threshold: float) -> dict:
    total = len(outcomes)
    passed = sum(1 for o in outcomes if o.ok)
    errored = sum(1 for o in outcomes if o.error)
    accuracy = 100 * passed / total if total else 0.0

    # Per-field accuracy over only the cases that pin that field.
    graded: Counter[str] = Counter()
    correct: Counter[str] = Counter()
    for o in outcomes:
        if o.error:
            continue
        wrong = {m[0] for m in o.mismatches}
        for key in o.case.expect:
            graded[key] += 1
            if key not in wrong:
                correct[key] += 1

    latencies = sorted(o.latency_ms for o in outcomes if not o.error)
    sources = Counter(o.got.get("source", "?") for o in outcomes if not o.error)

    print(f"\nintent parsing      {passed}/{total}   {accuracy:.1f}%"
          + (f"   ({errored} request errors)" if errored else ""))
    for key in sorted(graded, key=lambda k: (correct[k] / graded[k], k)):
        pct = 100 * correct[key] / graded[key]
        bar = "█" * round(pct / 5) + "·" * (20 - round(pct / 5))
        print(f"  {key:12} {correct[key]:>3}/{graded[key]:<3} {pct:5.1f}%  {bar}")

    if latencies:
        p50 = statistics.median(latencies)
        p95 = latencies[min(len(latencies) - 1, int(len(latencies) * 0.95))]
        print(f"\nlatency             p50 {p50:.0f}ms   p95 {p95:.0f}ms")
    print(f"intent source       {dict(sources)}")
    print(f"cost per query      $0.0000  (deterministic rules; no model calls)")

    failures = [o for o in outcomes if not o.ok]
    if failures:
        print(f"\n{len(failures)} failing case(s):")
        for o in failures:
            if o.error:
                print(f"  ✗ {o.case.q}\n      request failed: {o.error}")
                continue
            detail = "  ".join(f"{k}: want {w!r} got {g!r}" for k, w, g in o.mismatches)
            print(f"  ✗ {o.case.q}")
            print(f"      {detail}")
            if o.case.note:
                print(f"      ({o.case.note})")

    verdict = "PASS" if accuracy >= threshold else "FAIL"
    print(f"\n{verdict}  accuracy {accuracy:.1f}%  threshold {threshold:.1f}%\n")

    return {
        "total": total,
        "passed": passed,
        "errored": errored,
        "accuracy_pct": round(accuracy, 1),
        "threshold_pct": threshold,
        "per_field": {k: {"correct": correct[k], "graded": graded[k],
                          "pct": round(100 * correct[k] / graded[k], 1)} for k in graded},
        "latency_ms": {"p50": round(statistics.median(latencies), 1) if latencies else None,
                       "p95": round(latencies[min(len(latencies) - 1, int(len(latencies) * 0.95))], 1)
                       if latencies else None},
        "intent_sources": dict(sources),
        "failures": [{"q": o.case.q, "note": o.case.note, "error": o.error,
                      "mismatches": [{"field": k, "want": w, "got": g} for k, w, g in o.mismatches]}
                     for o in outcomes if not o.ok],
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Grade Persian query understanding.")
    ap.add_argument("--api", default="https://khodrobin.noxioai.com",
                    help="base URL of a running API")
    ap.add_argument("--golden", default=str(GOLDEN))
    ap.add_argument("--threshold", type=float, default=85.0,
                    help="fail below this accuracy")
    ap.add_argument("--timeout", type=float, default=15.0)
    ap.add_argument("--json", dest="json_out", default=str(HERE / "report.json"))
    args = ap.parse_args()

    cases = load_cases(pathlib.Path(args.golden))
    print(f"grading {len(cases)} queries against {args.api}", file=sys.stderr)

    outcomes = run(args.api, cases, args.timeout)
    result = report(outcomes, args.threshold)

    pathlib.Path(args.json_out).write_text(
        json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")

    return 0 if result["accuracy_pct"] >= args.threshold else 1


if __name__ == "__main__":
    raise SystemExit(main())

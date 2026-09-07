"""The AI service: two narrow jobs, both measured, neither trusted blindly.

It does not rank, choose, or decide what to show. It reads a sentence and fills
a struct, or reads facts and writes two sentences — and both results are
validated before they leave the process.

Every response carries its own cost and latency. A model whose price you cannot
state is a model you cannot run in production.
"""
from __future__ import annotations

import hashlib
import json
import logging
import os
import time
from collections import OrderedDict
from typing import Any

from fastapi import FastAPI
from pydantic import BaseModel, Field

from app import explain as explain_mod
from app import intent as intent_mod
from app import providers

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger("khodrobin.ai")

app = FastAPI(title="KhodroBin AI", version="0.1.0")

# A bounded in-process cache. Explanations are pure functions of their facts and
# the prompt version, so an unchanged spec never pays for inference twice.
# Bounded because an unbounded cache is a memory leak with good intentions.
CACHE_MAX = int(os.getenv("AI_CACHE_MAX", "2000"))
_cache: OrderedDict[str, dict] = OrderedDict()

# Bump when a prompt changes: the cache key includes it, so old entries expire
# by construction instead of by hand.
PROMPT_VERSION = "1"

# A hard daily ceiling. Past it the service serves cache and fallbacks rather
# than spending. Torob publishes their own $/day; running without a limit is not
# a thing to show them.
DAILY_CEILING_USD = float(os.getenv("AI_DAILY_COST_CEILING_USD", "5"))

_spend = {"day": time.strftime("%Y-%m-%d"), "usd": 0.0, "calls": 0,
          "cache_hits": 0, "guard_rejections": 0, "fallbacks": 0}


def _budget_left() -> bool:
    today = time.strftime("%Y-%m-%d")
    if _spend["day"] != today:
        _spend.update(day=today, usd=0.0, calls=0, cache_hits=0,
                      guard_rejections=0, fallbacks=0)
    return _spend["usd"] < DAILY_CEILING_USD


def _record(usage: providers.Usage) -> None:
    _spend["usd"] += usage.cost_usd
    _spend["calls"] += 1


def _cache_get(key: str) -> dict | None:
    if key in _cache:
        _cache.move_to_end(key)
        _spend["cache_hits"] += 1
        return _cache[key]
    return None


def _cache_put(key: str, value: dict) -> None:
    _cache[key] = value
    _cache.move_to_end(key)
    while len(_cache) > CACHE_MAX:
        _cache.popitem(last=False)


def _key(*parts: Any) -> str:
    blob = json.dumps(parts, ensure_ascii=False, sort_keys=True, default=str)
    return hashlib.sha256((PROMPT_VERSION + blob).encode()).hexdigest()[:32]


class IntentRequest(BaseModel):
    q: str
    brands: list[dict] = Field(default_factory=list)
    models: list[dict] = Field(default_factory=list)


class ExplainRequest(BaseModel):
    spec: dict


@app.get("/health")
def health() -> dict:
    p = providers.from_env()
    return {"status": "ok", "provider": p.name, "model": p.model,
            "cache": len(_cache), "spend": _spend}


@app.post("/intent")
def parse_intent(req: IntentRequest) -> dict:
    """Persian sentence → validated Intent.

    Called only for queries the deterministic parser could not resolve, so its
    latency lands on a small minority of traffic.
    """
    key = _key("intent", req.q, len(req.brands), len(req.models))
    if (hit := _cache_get(key)) is not None:
        return hit | {"cached": True}

    if not _budget_left():
        return {"intent": intent_mod.Intent().model_dump(), "source": "budget_exceeded",
                "cached": False, "usage": None}

    provider = providers.from_env()
    try:
        completion = provider.complete(
            intent_mod.SYSTEM,
            intent_mod.build_prompt(req.q, req.brands, req.models),
            json_mode=True,
        )
        raw = providers.parse_json(completion.text)
        parsed = intent_mod.validate(raw, req.brands, req.models)
    except (providers.ProviderError, ValueError, KeyError) as exc:
        # The Go service still has its rules answer; failing here degrades the
        # result rather than breaking the request.
        log.warning(json.dumps({"event": "intent.failed", "err": str(exc)[:200]}))
        return {"intent": intent_mod.Intent().model_dump(), "source": "error",
                "cached": False, "usage": None, "error": str(exc)[:200]}

    _record(completion.usage)
    out = {"intent": parsed.model_dump(), "source": "llm", "cached": False,
           "usage": completion.usage.__dict__}
    _cache_put(key, out)
    log.info(json.dumps({"event": "intent.served", "q": req.q,
                         "brand": parsed.brand, "model": parsed.model,
                         "cost_usd": completion.usage.cost_usd,
                         "latency_ms": round(completion.usage.latency_ms)},
                        ensure_ascii=False))
    return out


@app.post("/explain")
def explain(req: ExplainRequest) -> dict:
    """Two Persian sentences about why the top offer wins.

    Every number in the answer is checked against the numbers that went in. An
    explanation that invents a figure is discarded, not shipped with a warning:
    on a price-comparison product a wrong number is the whole product being
    wrong.
    """
    facts = explain_mod.facts_from_spec(req.spec)
    key = _key("explain", facts)
    if (hit := _cache_get(key)) is not None:
        return hit | {"cached": True}

    if not _budget_left():
        _spend["fallbacks"] += 1
        return {"text": explain_mod.fallback(facts), "source": "budget_exceeded",
                "cached": False, "usage": None}

    provider = providers.from_env()
    try:
        completion = provider.complete(explain_mod.SYSTEM,
                                       explain_mod.build_prompt(facts),
                                       json_mode=False)
        text = completion.text.strip()
        _record(completion.usage)
    except providers.ProviderError as exc:
        log.warning(json.dumps({"event": "explain.failed", "err": str(exc)[:200]}))
        _spend["fallbacks"] += 1
        return {"text": explain_mod.fallback(facts), "source": "fallback",
                "cached": False, "usage": None, "error": str(exc)[:200]}

    ok, unsupported = explain_mod.check(text, facts)
    if not ok:
        _spend["guard_rejections"] += 1
        log.warning(json.dumps({"event": "explain.rejected",
                                "unsupported": unsupported[:5],
                                "text": text[:200]}, ensure_ascii=False))
        out = {"text": explain_mod.fallback(facts), "source": "fallback",
               "cached": False, "rejected_numbers": unsupported[:5],
               "usage": completion.usage.__dict__}
        _cache_put(key, out)
        return out

    out = {"text": text, "source": "llm", "cached": False,
           "usage": completion.usage.__dict__}
    _cache_put(key, out)
    log.info(json.dumps({"event": "explain.served",
                         "cost_usd": completion.usage.cost_usd,
                         "latency_ms": round(completion.usage.latency_ms)}))
    return out


@app.get("/metrics")
def metrics() -> str:
    """Prometheus exposition. Cost is a first-class metric, not an afterthought."""
    lines = [
        "# HELP khodrobin_ai_spend_usd_today Model spend today in USD.",
        "# TYPE khodrobin_ai_spend_usd_today gauge",
        f"khodrobin_ai_spend_usd_today {_spend['usd']:.6f}",
        "# HELP khodrobin_ai_calls_total Model calls today.",
        "# TYPE khodrobin_ai_calls_total counter",
        f"khodrobin_ai_calls_total {_spend['calls']}",
        "# HELP khodrobin_ai_cache_hits_total Answers served from cache today.",
        "# TYPE khodrobin_ai_cache_hits_total counter",
        f"khodrobin_ai_cache_hits_total {_spend['cache_hits']}",
        "# HELP khodrobin_ai_guard_rejections_total Explanations discarded for citing an unsupported number.",
        "# TYPE khodrobin_ai_guard_rejections_total counter",
        f"khodrobin_ai_guard_rejections_total {_spend['guard_rejections']}",
        "# HELP khodrobin_ai_fallbacks_total Templated explanations served.",
        "# TYPE khodrobin_ai_fallbacks_total counter",
        f"khodrobin_ai_fallbacks_total {_spend['fallbacks']}",
    ]
    return "\n".join(lines) + "\n"

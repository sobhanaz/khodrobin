"""Model providers behind one small interface.

Four providers, one method. Which one runs is a single environment variable, so
the product does not depend on any of them being reachable — and from Iran that
is not hypothetical: the cloud APIs refuse Iranian IPs outright, while a local
Ollama always answers.

Every call returns a Usage record alongside the text. Cost and latency are not
optional extras here: a model whose price you cannot state is a model you cannot
run in production, and reporting $/query is the difference between "I used AI"
and "I know what my AI costs".
"""
from __future__ import annotations

import json
import os
import time
from dataclasses import dataclass, field

import httpx

# Published prices per million tokens, USD. Used to turn token counts into a
# number that can go on a slide. Update when a provider changes pricing.
PRICING: dict[str, tuple[float, float]] = {
    "gpt-4o-mini": (0.15, 0.60),
    "gpt-4o": (2.50, 10.00),
    "claude-haiku-4-5-20251001": (1.00, 5.00),
    "gemini-2.0-flash": (0.10, 0.40),
}


@dataclass
class Usage:
    provider: str
    model: str
    tokens_in: int = 0
    tokens_out: int = 0
    latency_ms: float = 0.0
    cost_usd: float = 0.0
    cached: bool = False

    def priced(self) -> "Usage":
        rate = PRICING.get(self.model)
        if rate:
            self.cost_usd = (self.tokens_in / 1e6) * rate[0] + (self.tokens_out / 1e6) * rate[1]
        return self


@dataclass
class Completion:
    text: str
    usage: Usage
    raw: dict = field(default_factory=dict)


class ProviderError(RuntimeError):
    """The model could not be reached or refused the request."""


class Provider:
    name = "base"

    def __init__(self, model: str, timeout: float = 60.0):
        self.model = model
        self.timeout = timeout

    def complete(self, system: str, user: str, *, json_mode: bool = True) -> Completion:
        raise NotImplementedError


class Ollama(Provider):
    """Local models. No key, no network egress, works from anywhere.

    Slower than the hosted APIs on CPU, which matters far less than it sounds:
    the model only runs when the deterministic parser could not answer, so its
    latency lands on a small minority of queries.
    """

    name = "ollama"

    def __init__(self, model: str, host: str | None = None, timeout: float = 120.0):
        super().__init__(model, timeout)
        self.host = host or os.getenv("OLLAMA_HOST", "http://ollama:11434")

    def complete(self, system: str, user: str, *, json_mode: bool = True) -> Completion:
        payload = {
            "model": self.model,
            "system": system,
            "prompt": user,
            "stream": False,
            # Deterministic: an intent parser that answers differently on
            # identical input cannot be evaluated.
            "options": {"temperature": 0, "num_predict": 512},
        }
        if json_mode:
            payload["format"] = "json"
        started = time.perf_counter()
        try:
            with httpx.Client(timeout=self.timeout) as c:
                r = c.post(f"{self.host}/api/generate", json=payload)
                r.raise_for_status()
                body = r.json()
        except httpx.HTTPError as exc:
            raise ProviderError(f"ollama: {exc}") from exc
        ms = (time.perf_counter() - started) * 1000
        return Completion(
            text=body.get("response", ""),
            usage=Usage(self.name, self.model,
                        tokens_in=body.get("prompt_eval_count", 0),
                        tokens_out=body.get("eval_count", 0),
                        latency_ms=ms).priced(),
            raw=body,
        )


class OpenAI(Provider):
    name = "openai"

    def complete(self, system: str, user: str, *, json_mode: bool = True) -> Completion:
        key = os.getenv("OPENAI_API_KEY")
        if not key:
            raise ProviderError("OPENAI_API_KEY is not set")
        payload: dict = {
            "model": self.model,
            "temperature": 0,
            "messages": [{"role": "system", "content": system},
                         {"role": "user", "content": user}],
        }
        if json_mode:
            payload["response_format"] = {"type": "json_object"}
        started = time.perf_counter()
        try:
            with httpx.Client(timeout=self.timeout) as c:
                r = c.post("https://api.openai.com/v1/chat/completions",
                           json=payload, headers={"Authorization": f"Bearer {key}"})
                r.raise_for_status()
                body = r.json()
        except httpx.HTTPError as exc:
            raise ProviderError(f"openai: {exc}") from exc
        ms = (time.perf_counter() - started) * 1000
        u = body.get("usage", {})
        return Completion(
            text=body["choices"][0]["message"]["content"],
            usage=Usage(self.name, self.model,
                        tokens_in=u.get("prompt_tokens", 0),
                        tokens_out=u.get("completion_tokens", 0),
                        latency_ms=ms).priced(),
            raw=body,
        )


class Anthropic(Provider):
    name = "anthropic"

    def complete(self, system: str, user: str, *, json_mode: bool = True) -> Completion:
        key = os.getenv("ANTHROPIC_API_KEY")
        if not key:
            raise ProviderError("ANTHROPIC_API_KEY is not set")
        # No JSON mode on this API; the schema is enforced by the prompt and,
        # more importantly, by validating the result after it comes back.
        started = time.perf_counter()
        try:
            with httpx.Client(timeout=self.timeout) as c:
                r = c.post("https://api.anthropic.com/v1/messages",
                           json={"model": self.model, "max_tokens": 512,
                                 "temperature": 0, "system": system,
                                 "messages": [{"role": "user", "content": user}]},
                           headers={"x-api-key": key,
                                    "anthropic-version": "2023-06-01"})
                r.raise_for_status()
                body = r.json()
        except httpx.HTTPError as exc:
            raise ProviderError(f"anthropic: {exc}") from exc
        ms = (time.perf_counter() - started) * 1000
        u = body.get("usage", {})
        return Completion(
            text="".join(b.get("text", "") for b in body.get("content", [])),
            usage=Usage(self.name, self.model,
                        tokens_in=u.get("input_tokens", 0),
                        tokens_out=u.get("output_tokens", 0),
                        latency_ms=ms).priced(),
            raw=body,
        )


class Gemini(Provider):
    name = "gemini"

    def complete(self, system: str, user: str, *, json_mode: bool = True) -> Completion:
        key = os.getenv("GEMINI_API_KEY")
        if not key:
            raise ProviderError("GEMINI_API_KEY is not set")
        cfg: dict = {"temperature": 0, "maxOutputTokens": 512}
        if json_mode:
            cfg["responseMimeType"] = "application/json"
        started = time.perf_counter()
        try:
            with httpx.Client(timeout=self.timeout) as c:
                r = c.post(
                    f"https://generativelanguage.googleapis.com/v1beta/models/{self.model}:generateContent",
                    params={"key": key},
                    json={"systemInstruction": {"parts": [{"text": system}]},
                          "contents": [{"parts": [{"text": user}]}],
                          "generationConfig": cfg})
                r.raise_for_status()
                body = r.json()
        except httpx.HTTPError as exc:
            raise ProviderError(f"gemini: {exc}") from exc
        ms = (time.perf_counter() - started) * 1000
        u = body.get("usageMetadata", {})
        parts = body["candidates"][0]["content"]["parts"]
        return Completion(
            text="".join(p.get("text", "") for p in parts),
            usage=Usage(self.name, self.model,
                        tokens_in=u.get("promptTokenCount", 0),
                        tokens_out=u.get("candidatesTokenCount", 0),
                        latency_ms=ms).priced(),
            raw=body,
        )


REGISTRY = {"ollama": Ollama, "openai": OpenAI, "anthropic": Anthropic, "gemini": Gemini}

DEFAULT_MODEL = {
    "ollama": "qwen2.5:7b",
    "openai": "gpt-4o-mini",
    "anthropic": "claude-haiku-4-5-20251001",
    "gemini": "gemini-2.0-flash",
}


def from_env() -> Provider:
    name = os.getenv("AI_PROVIDER", "ollama").strip().lower()
    if name not in REGISTRY:
        raise ProviderError(f"unknown AI_PROVIDER {name!r}; expected one of {sorted(REGISTRY)}")
    model = os.getenv("AI_MODEL") or DEFAULT_MODEL[name]
    return REGISTRY[name](model)


def parse_json(text: str) -> dict:
    """Pull a JSON object out of a model response.

    Even in JSON mode a model will occasionally wrap its answer in prose or a
    code fence. Salvaging the object is worth doing; trusting it is not — every
    caller validates the result against a schema afterwards.
    """
    text = text.strip()
    if text.startswith("```"):
        text = text.split("```")[1] if "```" in text[3:] else text[3:]
        text = text.removeprefix("json").strip()
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        start, end = text.find("{"), text.rfind("}")
        if start >= 0 and end > start:
            return json.loads(text[start:end + 1])
        raise

"""Per-host politeness limiting.

Every source gets its own token bucket. We are a guest on these sites: one
request per second with jitter, a real User-Agent that says who we are and how
to reach us, and a circuit breaker that backs off hard when a host pushes back.
"""
from __future__ import annotations

import random
import threading
import time

USER_AGENT = (
    "KhodroBinBot/0.1 (+https://github.com/sobhanaz/khodrobin; "
    "research prototype; contact via repo issues)"
)


class HostLimiter:
    """One token bucket per host, plus a breaker that opens on repeated 429/5xx."""

    def __init__(self, min_interval: float = 1.0, jitter: float = 0.4,
                 breaker_threshold: int = 3, breaker_cooldown: float = 1800.0):
        self.min_interval = min_interval
        self.jitter = jitter
        self.breaker_threshold = breaker_threshold
        self.breaker_cooldown = breaker_cooldown
        # A host may ask for a gentler cadence than the default.
        self.min_interval_for: dict[str, float] = {}
        self._last: dict[str, float] = {}
        self._strikes: dict[str, int] = {}
        self._open_until: dict[str, float] = {}
        self._lock = threading.Lock()

    def is_open(self, host: str) -> bool:
        """True when the breaker is open and the host must not be touched."""
        return time.monotonic() < self._open_until.get(host, 0.0)

    def wait(self, host: str) -> None:
        with self._lock:
            interval = self.min_interval_for.get(host, self.min_interval)
            gap = time.monotonic() - self._last.get(host, 0.0)
            delay = max(0.0, interval - gap) + random.uniform(0, self.jitter)
            self._last[host] = time.monotonic() + delay
        if delay:
            time.sleep(delay)

    def record_empty(self, host: str) -> None:
        """A 200 that carried no data.

        Divar answers HTTP 200 with the listing JSON-LD simply absent once it
        decides you are crawling too fast. Status codes therefore cannot be
        trusted as a health signal — an empty success is a soft block and has to
        count against the host exactly like a 429 would.
        """
        self.record(host, 429)

    def record(self, host: str, status: int) -> None:
        if status == 429 or status >= 500:
            self._strikes[host] = self._strikes.get(host, 0) + 1
            if self._strikes[host] >= self.breaker_threshold:
                self._open_until[host] = time.monotonic() + self.breaker_cooldown
                self._strikes[host] = 0
        else:
            self._strikes[host] = 0

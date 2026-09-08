"""One module per marketplace. Each exposes fetch(limiter, ...) -> list[dict].

The second argument is pages for the national feeds and cities for the two
city-scoped sites; run.py knows which is which.

Every source returns *raw* records — whatever the site gave us, plus a small
envelope. Normalization happens later and separately, so a parser bug never
costs us a re-crawl. The one payload any module is allowed to add to is
Sheypoor's, which states its ads' city on the page rather than in the ad; the
key it adds is prefixed with the source name so it cannot be mistaken later for
something the site sent.
"""
from . import bama, divar, hamrah, khodro45, sheypoor

ALL = {"divar": divar, "bama": bama, "hamrah": hamrah, "khodro45": khodro45,
       "sheypoor": sheypoor}

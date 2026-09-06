"""One module per marketplace. Each exposes fetch(limiter, pages) -> list[dict].

Every source returns *raw* records — whatever the site gave us, untouched, plus
a small envelope. Normalization happens later and separately, so a parser bug
never costs us a re-crawl.
"""
from . import bama, divar, hamrah

ALL = {"divar": divar, "bama": bama, "hamrah": hamrah}

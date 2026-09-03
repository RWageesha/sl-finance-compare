"""Parses the small vocabulary of tenure and interest-rate strings that Sri
Lankan bank rate tables tend to use (e.g. "12 Months", "1 Year", "9.55%").
Shared across all bank scraper modules so each one doesn't reimplement the
same parsing. Port of internal/scrapers/ratetext/ratetext.go — keep the two
in sync if either changes.
"""

import re

# How banks sometimes spell out non-numeric tenure periods, mapped to a
# number of months. Extend this if a bank uses a new label.
_WORDS = {
    "one month": 1,
    "three months": 3,
    "six months": 6,
    "twelve months": 12,
}

_TENURE_NUMBER_RE = re.compile(r"(\d+)\s*(day|month|year)s?")
_RATE_NUMBER_RE = re.compile(r"(\d+(?:\.\d+)?)\s*%?")
_FLAT_RATE_RE = re.compile(r"^\d+(?:\.\d+)?\s*%?$")


class ParseError(ValueError):
    """Raised when a tenure or rate string can't be parsed."""


def parse_tenure_months(s: str) -> int:
    """Converts strings like "3 Months", "1 Year", "30 Days", or one of the
    known word forms into a whole number of months. Day-based tenures round
    up to a 1 month floor (0 months is not a valid tenure). Extra
    surrounding text (e.g. "12 Months -Interest at maturity (LKR)") is
    tolerated — only the first number+unit match is used.
    """
    normalized = s.strip().lower()
    if normalized in _WORDS:
        return _WORDS[normalized]

    m = _TENURE_NUMBER_RE.search(normalized)
    if m is None:
        raise ParseError("unrecognized tenure format")

    n = int(m.group(1))
    unit = m.group(2)

    if unit == "year":
        return n * 12
    if unit == "month":
        return n
    if unit == "day":
        if n <= 0:
            raise ParseError("non-positive day tenure")
        months = n // 30
        return max(months, 1)
    raise ParseError(f"unrecognized tenure unit {unit!r}")


def parse_rate(s: str) -> float:
    """Extracts a percentage value like "13.50%", "13.5 %", or a bare
    "13.50" as a float.
    """
    m = _RATE_NUMBER_RE.search(s.strip())
    if m is None:
        raise ParseError("unrecognized rate format")
    rate = float(m.group(1))
    if rate <= 0 or rate > 100:
        raise ParseError(f"rate {rate:.2f} out of plausible range")
    return rate


def parse_flat_rate(s: str) -> float:
    """A stricter sibling of parse_rate: the entire trimmed string must be a
    bare percentage, with no surrounding text. This is used to tell a real,
    comparable rate ("13.50%") apart from a floating-rate formula
    ("AWPLR + 2.50%"), a range ("12.50% - 17.75%"), or a placeholder
    ("-", "Please refer Treasury Division") — none of which reduce to a
    single comparable number.
    """
    trimmed = s.strip()
    if not _FLAT_RATE_RE.match(trimmed):
        raise ParseError("not a flat percentage")
    return parse_rate(trimmed)

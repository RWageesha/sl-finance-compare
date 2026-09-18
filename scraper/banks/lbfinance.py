"""Scrapes Fixed Deposit rates from LB Finance PLC's public rates page.
LB Finance is a licensed finance company (an NBFI), not a commercial bank
— it's just another row in the same banks/products tables. The FD table
is a Word/Office-exported table (inline MSO styles) but is still plain,
parseable HTML — verified with a plain curl fetch, no WAF.

Only Fixed Deposits are scraped. LB Finance's savings-product pages and
loan/leasing pages are marketing prose with no rate table anywhere on
them (checked by hand) — nothing there to scrape honestly.
"""

from __future__ import annotations

import datetime as dt

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "LB Finance"
BANK_CODE = "LBFIN"

FD_URL = "https://www.lbfinance.com/fixed-deposits/fixed-deposits"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

# Every data row is a fixed 9-cell layout: [Period, Normal-Monthly,
# Normal-Monthly-AER, Normal-Maturity, Normal-Maturity-AER,
# Senior-Monthly, Senior-Monthly-AER, Senior-Maturity,
# Senior-Maturity-AER] — the two-row header (Normal/Senior groups, then
# Monthly/Maturity sub-columns) is skipped rather than parsed, since the
# column positions are fixed and simpler to hardcode than reconstruct.
_MATURITY_COLUMNS = (("normal", 3), ("senior", 7))


class ParseError(ValueError):
    pass


def fetch_fd_page() -> str:
    resp = requests.get(FD_URL, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts both the Normal and Senior Citizen Maturity-payout columns
    from LB Finance's rate table. Monthly-payout columns are skipped, the
    usual convention; several short/day-based tenures only publish a
    Maturity rate at all (Monthly shows "N/A"), which parse_flat_rate
    naturally skips without special-casing.
    """
    soup = BeautifulSoup(html, "lxml")
    table = soup.find("table")
    if table is None:
        raise ParseError("lbfinance: rate table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.find_all("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) != 9:
            continue
        tenure_text = cells[0].get_text(strip=True)
        try:
            months = ratetext.parse_tenure_months(tenure_text)
        except ratetext.ParseError:
            continue

        for rate_type, col in _MATURITY_COLUMNS:
            try:
                rate = ratetext.parse_flat_rate(cells[col].get_text(strip=True))
            except ratetext.ParseError:
                continue
            rates.append(
                {
                    "tenure_months": months,
                    "tenure_label": tenure_text,
                    "interest_rate": rate,
                    "rate_type": rate_type,
                    "source_url": FD_URL,
                    "scraped_at": scrape_at,
                }
            )

    if not rates:
        raise ParseError("lbfinance: no fixed deposit rates parsed (selectors likely stale)")
    return rates

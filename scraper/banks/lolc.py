"""Scrapes Fixed Deposit rates from LOLC Finance PLC's public rates page.
LOLC is a licensed finance company (an NBFI), not a commercial bank — the
site's schema doesn't distinguish, and neither does this scraper; it just
adds another row to the same banks/products tables. Server-rendered HTML
via a WordPress TablePress plugin, no WAF — verified with a plain curl
fetch before writing any selectors.

Only Fixed Deposits are scraped. The "savings" rates-and-tariffs page
turned out to be a pure fee schedule (passbook issuance, transfer fees,
...) with no interest rate column at all, so there's nothing there to
scrape honestly. The loans/leasing pages are marketing prose with no rate
table either (checked by hand) — LOLC's real leasing/loan pricing isn't
published as a rate schedule anywhere on the site.
"""

from __future__ import annotations

import datetime as dt

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "LOLC Finance"
BANK_CODE = "LOLC"

FD_URL = "https://www.lolcfinance.com/rates-and-returns/interest-rates/"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

# Each FD tier is its own tab-pane, distinguishable only by this element
# id (both tables otherwise share the identical "InterestRates" heading).
# The "SCFD" id is misleading — its own tab label reads "WAKALA Investment
# Rates" (a Shariah-compliant investment product), not a senior-citizen
# scheme; "GeneralFD" is labelled "General / Senior Citizen Fixed
# Deposits" (one schedule covering both), confirmed by reading the tab
# button text itself rather than trusting the element id.
_TIERS = (("GeneralFD", "normal"), ("SCFD", "special"))


class ParseError(ValueError):
    pass


def fetch_fd_page() -> str:
    resp = requests.get(FD_URL, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts both FD tiers' rate tables: a 2-row header (payout-group
    then Simple/AER sub-columns) over [Period, Monthly Simple/AER,
    Annually Simple/AER, Maturity Simple/AER]. Only the Maturity-Simple
    column is kept as the standard rate, the usual convention; several
    short/odd tenures only publish a Maturity rate at all (Monthly/
    Annually show "-"), which parse_flat_rate naturally skips without
    special-casing.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for tab_id, rate_type in _TIERS:
        container = soup.find(id=tab_id)
        table = container.find("table") if container else None
        if table is None:
            continue

        for row in table.find_all("tr")[2:]:  # skip the 2-row header
            cells = row.find_all(["td", "th"])
            if len(cells) != 7:
                continue
            tenure_text = cells[0].get_text(strip=True)
            try:
                months = int(tenure_text)
            except ValueError:
                continue

            try:
                rate = ratetext.parse_flat_rate(cells[5].get_text(strip=True))
            except ratetext.ParseError:
                continue

            unit = "Month" if months == 1 else "Months"
            label = f"{tenure_text} {unit}" + (" (WAKALA)" if rate_type == "special" else "")
            rates.append(
                {
                    "tenure_months": months,
                    "tenure_label": label,
                    "interest_rate": rate,
                    "rate_type": rate_type,
                    "source_url": FD_URL,
                    "scraped_at": scrape_at,
                }
            )

    if not rates:
        raise ParseError("lolc: no fixed deposit rates parsed (selectors likely stale)")
    return rates

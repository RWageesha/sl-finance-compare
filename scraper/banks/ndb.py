"""Scrapes Fixed Deposit / Savings / Loan interest rates from National
Development Bank's public rates pages. Server-rendered HTML — the tables
are present directly in the response, parsed with BeautifulSoup, same
approach as boc.py/combank.py.

Both pages date-stamp every table ("Last Updated On: YYYY-MM-DD"), which is
how this module was chosen over their older product marketing pages —
those turned out to carry stale, years-old cached rate tables (checked by
hand before writing this).
"""

from __future__ import annotations

import datetime as dt

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "National Development Bank"
BANK_CODE = "NDB"

DEPOSITS_URL = "https://www.ndbbank.com/rates/interest-rates-on-deposits"
ADVANCES_URL = "https://www.ndbbank.com/rates/interest-rates-on-advances"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"


class ParseError(ValueError):
    pass


def _fetch(url: str) -> str:
    resp = requests.get(url, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def fetch_deposits_page() -> str:
    """Retrieves the raw HTML of NDB's deposit rates page (LKR Fixed
    Deposits and Savings Deposits tables, plus two foreign-currency tables
    this module ignores — LKR-only scope like every other scraper)."""
    return _fetch(DEPOSITS_URL)


def fetch_advances_page() -> str:
    """Retrieves the raw HTML of NDB's lending rates page."""
    return _fetch(ADVANCES_URL)


def _find_table(soup: BeautifulSoup, header_keyword: str, skip_keywords: tuple[str, ...] = ()) -> "any":
    """NDB's rates pages hold several tables (LKR + foreign-currency
    deposits, active-loan + refinance-scheme lending); each table's own
    second row is its real header ("Fixed Deposits", "Savings Deposits",
    ...), the first row being just a "Last Updated On" banner. Finds the
    table whose header row's first cell contains header_keyword and none
    of skip_keywords.
    """
    for table in soup.find_all("table"):
        rows = table.select("tr")
        if len(rows) < 2:
            continue
        header_cells = rows[1].find_all(["td", "th"])
        if not header_cells:
            continue
        header_text = header_cells[0].get_text(strip=True).lower()
        if header_keyword.lower() not in header_text:
            continue
        if any(kw.lower() in header_text for kw in skip_keywords):
            continue
        return table
    return None


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts LKR Fixed Deposit rates. The table uses a rowspan on the
    tenure cell: a 4-cell row starts a new tenure ([tenure, description,
    min_rate, aer]); a 3-cell row continues the same tenure with a
    different payout description ([description, min_rate, aer]). Only
    "at Maturity" rows are kept as the standard rate (skipping "Payable
    Monthly" variants, same convention as every other bank's scraper);
    "Neos FD" rows are kept separately as a digital-banking bonus rate.
    tenure_label preserves the bank's own wording (e.g. "100 Days") since
    day-based tenures can round to the same tenure_months as an unrelated
    whole-month product.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_table(soup, "Fixed Deposits", skip_keywords=("foreign currency",))
    if table is None:
        raise ParseError("ndb: Fixed Deposits table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)
    current_tenure_text: str | None = None

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) == 4:
            current_tenure_text = cells[0].get_text(strip=True)
            description, rate_cell = cells[1], cells[2]
        elif len(cells) == 3:
            description, rate_cell = cells[0], cells[1]
        else:
            continue  # banner row or anything else unexpected

        if current_tenure_text is None:
            continue
        try:
            months = ratetext.parse_tenure_months(current_tenure_text)
        except ratetext.ParseError:
            continue

        desc_text = description.get_text(strip=True)
        lower_desc = desc_text.lower()
        if "monthly" in lower_desc:
            continue  # keep only the "at Maturity" payout variant

        try:
            rate = ratetext.parse_flat_rate(rate_cell.get_text(strip=True))
        except ratetext.ParseError:
            continue

        rate_type = "digital" if "neos" in lower_desc else "normal"
        rates.append(
            {
                "tenure_months": months,
                "tenure_label": current_tenure_text,
                "interest_rate": rate,
                "rate_type": rate_type,
                "source_url": DEPOSITS_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("ndb: no fixed deposit rates parsed (selectors likely stale)")
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts LKR Savings Deposit rates. Same rowspan shape as the FD
    table: a 4-cell row starts a new account ([account_name, tier,
    min_rate, aer]); a 3-cell row continues it with another balance tier
    ([tier, min_rate, aer]). 0.00% tier-1 placeholder rows are dropped
    automatically — ratetext.parse_flat_rate rejects non-positive rates.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_table(soup, "Savings Deposits", skip_keywords=("foreign currency",))
    if table is None:
        raise ParseError("ndb: Savings Deposits table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)
    current_account: str | None = None

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) == 4:
            current_account = cells[0].get_text(strip=True)
            tier_cell, rate_cell = cells[1], cells[2]
        elif len(cells) == 3:
            tier_cell, rate_cell = cells[0], cells[1]
        else:
            continue

        if not current_account:
            continue

        try:
            rate = ratetext.parse_flat_rate(rate_cell.get_text(strip=True))
        except ratetext.ParseError:
            continue

        balance_tier = " ".join(tier_cell.get_text(strip=True).split())
        rates.append(
            {
                "account_name": current_account,
                "balance_tier": balance_tier,
                "interest_rate": rate,
                "source_url": DEPOSITS_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("ndb: no savings rates parsed (selectors likely stale)")
    return rates


def parse_loans(html: str) -> list[dict]:
    """Extracts loan/advance rates from the active-facilities table (the
    separate Refinance Schemes table is all "N/A" placeholders and is
    skipped entirely). Each row packs one or more sub-products into
    parallel <p> tags across the category/description/min-rate/max-rate
    columns (e.g. Personal Loans' 3 sub-products each get their own <p> in
    every cell, positionally aligned) — both the min and max rate are kept
    as separate rows so the real range shows up on the site, same
    principle as every other multi-rate table. The "Others" column (always
    0.00%, an unrelated fee field) is intentionally not read.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_table(soup, "Description", skip_keywords=("refinance",))
    if table is None:
        raise ParseError("ndb: lending rates table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr")[2:]:  # skip the "Last Updated On" banner + header row
        cells = row.find_all("td")
        if len(cells) < 4:
            continue

        category = cells[0].get_text(strip=True)
        descriptions = [p.get_text(strip=True) for p in cells[1].find_all("p")] or [cells[1].get_text(strip=True)]
        min_rates = [p.get_text(strip=True) for p in cells[2].find_all("p")] or [cells[2].get_text(strip=True)]
        max_rates = [p.get_text(strip=True) for p in cells[3].find_all("p")] or [cells[3].get_text(strip=True)]

        for i, product_name in enumerate(descriptions):
            if not product_name:
                continue
            for rate_label, rate_list in (("Min Rate", min_rates), ("Max Rate", max_rates)):
                if i >= len(rate_list):
                    continue
                try:
                    rate = ratetext.parse_flat_rate(rate_list[i])
                except ratetext.ParseError:
                    continue
                rates.append(
                    {
                        "loan_category": category,
                        "loan_product": product_name,
                        "rate_label": rate_label,
                        "tenure": "",
                        "interest_rate": rate,
                        "source_url": ADVANCES_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError("ndb: no loan rates parsed (selectors likely stale)")
    return rates

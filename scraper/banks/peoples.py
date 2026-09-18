"""Scrapes Fixed Deposit / Savings / Loan interest rates from People's
Bank's public interest-rates page. Server-rendered HTML, no WAF —
verified with a plain curl fetch before writing any selectors.

The page holds 18 tables covering LKR retail, niche scheme (Co-operative/
Samurdhi/Parinatha), foreign-currency, and advances products, all with
similar-looking header rows — so (same as nsb.py) each table is picked out
by the section heading that precedes it, not by its own header row alone.
Niche/eligibility-restricted schemes (Co-operative sector, Samurdhi,
Parinatha, "People's Wealth") are intentionally skipped in favour of the
plain "Fixed deposits (Minimum deposit Rs. 5,000/-)" table every retail
customer can actually open — same "pick the generic retail product"
principle every other bank's scraper already follows.
"""

from __future__ import annotations

import datetime as dt

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "People's Bank"
BANK_CODE = "PEOPLES"

RATES_URL = "https://www.peoplesbank.lk/interest-rates/"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"


class ParseError(ValueError):
    pass


def fetch_rates_page() -> str:
    """Retrieves the raw HTML of People's Bank's interest-rates page (LKR
    Fixed Deposits, Savings, and Advances tables, plus FCY/niche-scheme
    tables this module ignores — LKR retail-only scope like every other
    scraper)."""
    resp = requests.get(RATES_URL, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def _heading_for(table) -> str:
    node = table.find_previous(["h1", "h2", "h3", "h4", "h5", "strong"])
    return node.get_text(strip=True) if node is not None else ""


def _first_flat_rate(cell_text: str) -> float:
    """Rate cells here pack the nominal rate and its AER together, e.g.
    "6.75% 6.96% (AER)" or "-" when a payout variant isn't offered — only
    the first whitespace-separated token is ever the nominal rate.
    """
    first_token = cell_text.split()[0] if cell_text.split() else ""
    return ratetext.parse_flat_rate(first_token)


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts the standard LKR Fixed Deposit table: [Period, At Maturity
    p.a., Monthly p.a.], flat rows (no rowspan). Only the "At Maturity"
    column is kept as the standard rate, same "keep the Maturity/AER
    payout" convention every other bank's scraper follows.
    """
    soup = BeautifulSoup(html, "lxml")
    table = None
    for t in soup.find_all("table"):
        if _heading_for(t).lower().startswith("fixed deposits (minimum deposit rs. 5,000"):
            table = t
            break
    if table is None:
        raise ParseError("peoples: standard Fixed Deposits table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) != 3:
            continue
        period_text = cells[0].get_text(strip=True)
        try:
            months = ratetext.parse_tenure_months(period_text)
        except ratetext.ParseError:
            continue  # header row

        try:
            rate = _first_flat_rate(cells[1].get_text(" ", strip=True))
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "tenure_months": months,
                "tenure_label": period_text,
                "interest_rate": rate,
                "rate_type": "normal",
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("peoples: no fixed deposit rates parsed (selectors likely stale)")
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts the "Saving Products" table: [Product Name, Applicable
    Rate %, AER] — flat rows, one rate per product (no balance tiers).
    """
    soup = BeautifulSoup(html, "lxml")
    table = None
    for t in soup.find_all("table"):
        if _heading_for(t).strip().lower() == "saving products":
            table = t
            break
    if table is None:
        raise ParseError("peoples: Saving Products table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) != 3:
            continue
        account_name = cells[0].get_text(strip=True)
        try:
            rate = ratetext.parse_flat_rate(cells[1].get_text(strip=True))
        except ratetext.ParseError:
            continue  # header row

        rates.append(
            {
                "account_name": account_name,
                "balance_tier": "",
                "interest_rate": rate,
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("peoples: no savings rates parsed (selectors likely stale)")
    return rates


def parse_loans(html: str) -> list[dict]:
    """Extracts loan rates from every "Interest Rates on Advances" table
    (the page has two: general advances, and a separate overdrafts table)
    — [label, Min. rate, Max. rate], both kept as separate labeled rows,
    same convention as NDB's min/max loan rows. AWPLR-linked and "-"
    placeholder cells aren't a single comparable percentage and are
    skipped, same as every other scraper.
    """
    soup = BeautifulSoup(html, "lxml")
    tables = [t for t in soup.find_all("table") if _heading_for(t).strip() == "Interest Rates on Advances"]
    if not tables:
        raise ParseError("peoples: Interest Rates on Advances table(s) not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for table in tables:
        for row in table.select("tr"):
            cells = row.find_all(["td", "th"])
            if len(cells) != 3:
                continue
            product = cells[0].get_text(strip=True)
            if product.lower() in ("description", "overdrafts"):
                continue  # header row

            for rate_label, cell in (("Min Rate", cells[1]), ("Max Rate", cells[2])):
                try:
                    rate = ratetext.parse_flat_rate(cell.get_text(strip=True))
                except ratetext.ParseError:
                    continue
                rates.append(
                    {
                        "loan_category": "Interest Rates on Advances",
                        "loan_product": product,
                        "rate_label": rate_label,
                        "tenure": "",
                        "interest_rate": rate,
                        "source_url": RATES_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError("peoples: no loan rates parsed (selectors likely stale)")
    return rates

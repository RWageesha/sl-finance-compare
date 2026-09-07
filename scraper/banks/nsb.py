"""Scrapes Fixed Deposit / Savings / Loan interest rates from National
Savings Bank's public rates pages. Server-rendered HTML — tables are
present directly in the response, parsed with BeautifulSoup, same approach
as boc.py/ndb.py. Verified against the real page structure by hand
(curl + BeautifulSoup inspection) before writing any selectors, per
project convention.

Both deposit and loan tables date/effective-stamp their section heading
("... W.E.F. 02/06/2026"), which is how each table is identified — NSB's
deposits page holds three tables (Savings Deposits, Term Deposits,
Certificate of Deposits) that all share an identical 6-column header row,
so a table can only be told apart from its neighbours by the heading that
precedes it, not by its own contents.

National Savings Certificates (the third deposits-page table) are a
distinct government savings-bond product, not an ordinary bank fixed
deposit or savings account, so this module intentionally does not scrape
them — keeping scope to the same Fixed Deposit / Savings / Loan taxonomy
every other bank module uses.
"""

from __future__ import annotations

import datetime as dt
import re

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "National Savings Bank"
BANK_CODE = "NSB"

DEPOSITS_URL = "https://www.nsb.lk/rates-tarriffs/rupee-deposit-rates/"
LENDING_URL = "https://www.nsb.lk/lending-rates/"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

_WEF_SUFFIX_RE = re.compile(r"\s*W\.?E\.?F\.?.*$", re.IGNORECASE)
_FD_PREFIX_RE = re.compile(r"^Fixed Deposit\s+", re.IGNORECASE)


class ParseError(ValueError):
    pass


def _fetch(url: str) -> str:
    resp = requests.get(url, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def fetch_deposits_page() -> str:
    """Retrieves the raw HTML of NSB's rupee deposit rates page (Savings
    Deposits, Term Deposits, and Certificate of Deposits tables — only the
    first two are scraped)."""
    return _fetch(DEPOSITS_URL)


def fetch_lending_page() -> str:
    """Retrieves the raw HTML of NSB's lending rates page."""
    return _fetch(LENDING_URL)


def _heading_for(table) -> str:
    """Returns the nearest preceding heading/strong tag's text, with the
    trailing "W.E.F. <date>" effective-date stamp stripped, or "" if none
    is found.
    """
    node = table.find_previous(["h1", "h2", "h3", "h4", "h5", "strong"])
    if node is None:
        return ""
    return _WEF_SUFFIX_RE.sub("", node.get_text(strip=True)).strip()


def _find_table_by_heading(soup: BeautifulSoup, keyword: str):
    """NSB's deposit tables all share an identical header row, so the only
    way to tell them apart is the section heading that precedes each one.
    """
    keyword = keyword.lower()
    for table in soup.find_all("table"):
        if keyword in _heading_for(table).lower():
            return table
    return None


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts LKR Term Deposit (Fixed Deposit) rates. Unlike HNB/NDB's
    rowspan-grouped tables, every row here is self-contained: [Details,
    Minimum Deposit, Annual Rate, Method, Payment of Interest, Effective
    Rate]. "Details" mixes tenure and product variant in one string (e.g.
    "Fixed Deposit 60 Months (Triple A)"), and several tenures are
    published twice — once paid at Maturity, once paid Monthly — under the
    exact same Details text, differing only in the Payment of Interest
    column. Rows are grouped by their raw Details text, and only when a
    group has both a Maturity and a Monthly row is the Monthly one dropped
    (the usual "keep the Maturity/AER figure" convention every other bank
    scraper follows); a Details text that appears only once (e.g. "Fixed
    Deposit 60 Months (For Retirement benefits)", which NSB only publishes
    with Monthly payout, or "Endowment Scheme", paid Yearly) is kept as-is
    since there is no Maturity sibling to prefer over it.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_table_by_heading(soup, "term deposit")
    if table is None:
        raise ParseError("nsb: Term Deposits table not found (page structure likely changed)")

    scrape_at = dt.datetime.now(dt.timezone.utc)
    parsed: list[dict] = []  # (details_text, payment, row_dict), pre-grouping

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) != 6:
            continue
        details = cells[0].get_text(strip=True)
        try:
            months = ratetext.parse_tenure_months(details)
        except ratetext.ParseError:
            continue  # header row, or a product with no numeric tenure (e.g. "Endowment Scheme")

        try:
            rate = ratetext.parse_flat_rate(cells[2].get_text(strip=True))
        except ratetext.ParseError:
            continue

        payment = cells[4].get_text(strip=True)
        lower_details = details.lower()
        rate_type = "special" if any(kw in lower_details for kw in ("retirement", "gaurawa", "pensioner", "endowment")) else "normal"

        # normalize.fixed_deposit() already appends "<Bank> Fixed Deposit" as
        # the product name, so a label that starts with "Fixed Deposit" would
        # otherwise show up doubled on the site (e.g. "Fixed Deposit 01 month
        # Fixed Deposit") — strip that leading phrase, keeping the rest
        # (tenure + any variant qualifier) as the label.
        tenure_label = _FD_PREFIX_RE.sub("", details).strip()

        parsed.append(
            {
                "_details": details,
                "_payment": payment,
                "tenure_months": months,
                "tenure_label": tenure_label,
                "interest_rate": rate,
                "rate_type": rate_type,
                "source_url": DEPOSITS_URL,
                "scraped_at": scrape_at,
            }
        )

    groups: dict[str, list[dict]] = {}
    for row in parsed:
        groups.setdefault(row["_details"], []).append(row)

    rates: list[dict] = []
    for group in groups.values():
        if len(group) > 1:
            group = [r for r in group if r["_payment"].lower() == "maturity"] or group
        for r in group:
            rates.append({k: v for k, v in r.items() if not k.startswith("_")})

    if not rates:
        raise ParseError("nsb: no fixed deposit rates parsed (selectors likely stale)")
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts LKR Savings Deposit rates. Same flat 6-column row shape as
    the Term Deposits table, no rowspan tiering — each account has exactly
    one published rate. Range rates ("03.00- 03.50", "3.25 – 7.29") aren't
    a single comparable figure and are skipped, same convention as every
    other bank's scraper.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_table_by_heading(soup, "savings deposit")
    if table is None:
        raise ParseError("nsb: Savings Deposits table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr"):
        cells = row.find_all(["td", "th"])
        if len(cells) != 6:
            continue
        account_name = cells[0].get_text(strip=True)
        if not account_name or account_name.lower() == "details":
            continue

        try:
            rate = ratetext.parse_flat_rate(cells[2].get_text(strip=True))
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "account_name": account_name,
                "balance_tier": "",
                "interest_rate": rate,
                "source_url": DEPOSITS_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("nsb: no savings rates parsed (selectors likely stale)")
    return rates


def parse_loans(html: str) -> list[dict]:
    """Extracts loan rates from the lending-rates page's per-category
    tables (Property Mortgages, Personal Guarantors, Movable Property,
    Specific Purpose loans, Cash-Deposit-secured loans, Pension Loans).
    Two worked penal-interest calculation examples further down the page
    share no structure with these and are excluded by only accepting
    tables whose header row starts with "Loan Type" or "Repayment Period".

    Row shape mirrors NDB's rowspan convention positionally even though
    there's no actual rowspan markup: a row of 3+ cells starts a new loan
    product ([loan_type, description, ..., rate]); a 2-cell row continues
    the current product with another description/rate pair. Rate values
    that are floating-rate formulas ("Effective Rate + 3% p. a") or
    multiple figures crammed into one cell ("13.25 13.50") aren't a single
    comparable percentage and are skipped, same as every other scraper.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for table in soup.find_all("table"):
        rows = table.find_all("tr")
        if not rows:
            continue
        header_cells = rows[0].find_all(["td", "th"])
        if not header_cells:
            continue
        header_first = header_cells[0].get_text(strip=True).lower()
        if header_first not in ("loan type", "repayment period"):
            continue

        category = _heading_for(table) or "NSB Loan"
        current_product = category

        for row in rows[1:]:
            cells = row.find_all(["td", "th"])
            texts = [c.get_text(" ", strip=True) for c in cells]
            if len(texts) < 2:
                continue

            try:
                rate = ratetext.parse_flat_rate(texts[-1])
            except ratetext.ParseError:
                continue

            if len(texts) >= 3 and texts[0]:
                current_product = texts[0]
                description = " ".join(t for t in texts[1:-1] if t)
            else:
                description = texts[0]

            rates.append(
                {
                    "loan_category": category,
                    "loan_product": current_product,
                    "rate_label": "",
                    "tenure": description,
                    "interest_rate": rate,
                    "source_url": LENDING_URL,
                    "scraped_at": scrape_at,
                }
            )

    if not rates:
        raise ParseError("nsb: no loan rates parsed (selectors likely stale)")
    return rates

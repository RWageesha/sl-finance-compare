"""Scrapes Fixed Deposit ("Term Investment") and Savings profit rates from
Amãna Bank's public rates page. Amana is a Shariah-compliant bank — it
doesn't pay "interest", it pays a "profit rate" on Mudarabah-based
deposits, but the number itself is directly comparable to every other
bank's interest rate, so it's stored the same way. Server-rendered HTML,
no WAF — verified with a plain curl fetch.

The page actually holds two Term Investment tables: rates for deposits
placed from the current month onward, and a separate (usually different)
schedule for deposits placed in earlier months. Only the current-month
table is scraped — the one a new depositor opening an account today would
actually get — matching every other bank's "what a new customer gets
today" scope.

Financing (loan) rates aren't scraped: Amana only publishes them as a PDF
of pricing *ranges* ("13.75% - 18.00%"), not a single flat rate per
product, which parse_flat_rate can't reduce to one comparable number —
same "skip a range rather than fake a single figure" rule every other
bank's scraper already follows for range-based cells.
"""

from __future__ import annotations

import datetime as dt
import re

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "Amana Bank"
BANK_CODE = "AMANA"

RATES_URL = "https://www.amanabank.lk/profit-sharing-ratios/local-currency-accounts-paid.html"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

# normalize.fixed_deposit() already appends "<Bank> Fixed Deposit" as the
# product name, so a label that itself says "Term Investment" (Amana's
# own name for the same thing) would otherwise show up doubled on the
# site (e.g. "12 Months Term Investment (Maturity) Fixed Deposit").
_TERM_INVESTMENT_RE = re.compile(r"\s*Term Investment\s*\(Maturity\)\s*", re.IGNORECASE)


class ParseError(ValueError):
    pass


def fetch_rates_page() -> str:
    resp = requests.get(RATES_URL, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def _current_term_investment_table(soup: BeautifulSoup):
    """The page has two Term Investment tables told apart only by their
    own header cell text — "...during <Month> <Year>" (current) vs
    "...prior to <Month> <Year>" (legacy schedule for older deposits).
    """
    for table in soup.find_all("table"):
        header = table.find("tr")
        if header is None:
            continue
        label = header.get_text(" ", strip=True).lower()
        if "term investment" in label and "during" in label:
            return table
    return None


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts the current-month Term Investment table: [Details, Profit
    Rate, AER]. "Details" mixes tenure and payout variant in one string
    (e.g. "24 Months Term Investment (Maturity) only on App") — only the
    "(Maturity)" variant is kept per tenure, the usual convention. A
    couple of named special products ("Amana Kid My Future Term
    Investment", "Flexi Term Investment") state no tenure at all and are
    skipped rather than guessed at.
    """
    soup = BeautifulSoup(html, "lxml")
    table = _current_term_investment_table(soup)
    if table is None:
        raise ParseError("amana: current-month Term Investment table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.find_all("tr")[1:]:
        cells = row.find_all(["td", "th"])
        if len(cells) != 3:
            continue
        label = cells[0].get_text(strip=True)
        if "monthly" in label.lower():
            continue  # keep only the "(Maturity)" payout variant

        try:
            months = ratetext.parse_tenure_months(label)
        except ratetext.ParseError:
            continue
        try:
            rate = ratetext.parse_rate(cells[1].get_text(strip=True))
        except ratetext.ParseError:
            continue

        tenure_label = _TERM_INVESTMENT_RE.sub(" ", label).strip()
        rates.append(
            {
                "tenure_months": months,
                "tenure_label": tenure_label,
                "interest_rate": rate,
                "rate_type": "normal",
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("amana: no term investment rates parsed (selectors likely stale)")
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts the Savings Accounts table: [Account Name, Profit Rate,
    AER] — flat rows, one rate per named account.
    """
    soup = BeautifulSoup(html, "lxml")
    table = None
    for t in soup.find_all("table"):
        header = t.find("tr")
        if header and header.get_text(strip=True).lower().startswith("savings accounts"):
            table = t
            break
    if table is None:
        raise ParseError("amana: Savings Accounts table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.find_all("tr")[1:]:
        cells = row.find_all(["td", "th"])
        if len(cells) != 3:
            continue
        account_name = cells[0].get_text(strip=True)
        try:
            rate = ratetext.parse_rate(cells[1].get_text(strip=True))
        except ratetext.ParseError:
            continue
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
        raise ParseError("amana: no savings rates parsed (selectors likely stale)")
    return rates

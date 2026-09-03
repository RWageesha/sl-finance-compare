"""Scrapes Fixed Deposit / Savings / Loan interest rates from Commercial
Bank of Ceylon's public pages. Unlike HNB, these pages are server-rendered
— the rate tables are present directly in the HTML response, so this
module parses them with BeautifulSoup rather than calling a backend API.

Port of internal/scrapers/combank/{fd,savings,loans}.go.
"""

from __future__ import annotations

import datetime as dt

import requests
from bs4 import BeautifulSoup

import ratetext

BANK_NAME = "Commercial Bank of Ceylon"
BANK_CODE = "COMB"

RATES_URL = "https://www.combank.lk/personal-banking/term-deposits/fixed-deposits"
# A separate hub page covering Savings and Loan products (Fixed Deposit
# rates stay on RATES_URL above, which has a cleaner single table for that
# specific product).
RATES_TARIFF_URL = "https://www.combank.lk/rates-tariff"

_REQUEST_TIMEOUT = 15
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

# The FD rate table has three columns beyond tenure: standard nominal
# rate, Annual Effective Rate, and a bonus "eFD" rate for customers who
# place the deposit through digital banking.
_STANDARD_RATE_COL = 1
_EFD_RATE_COL = 3

# The tab-pane on the rates & tariff page that bundles Savings Account,
# Fixed Deposit, and foreign-currency products — only the savings ones are
# wanted here (FD is scraped from its own, cleaner page; FX is out of
# scope, LKR-only like every other scraper).
_INTEREST_RATES_TAB_SELECTOR = "#interest-rates"
_SAVINGS_SKIP_KEYWORDS = ("fixed deposit", "foreign currency", "fc plus", "pfc", "bfc", "forex")

# The tab-pane covering loan products (Lease Facilities, Personal Loans,
# Home Loans, Gold Loans Pawning, Diribala Loans, Agriculture Sector, All
# Other Advances, Penal Interest on Overdue Credit Facilities).
_LENDING_RATES_TAB_SELECTOR = "#lending-rates"


class ParseError(ValueError):
    pass


def _fetch(url: str) -> str:
    # This site's WAF appears to reject requests that don't look like a
    # real browser (a bot-labelled UA got blocked during development).
    resp = requests.get(url, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


def fetch_page() -> str:
    """Retrieves the raw HTML of the ComBank fixed deposit rates page."""
    return _fetch(RATES_URL)


def fetch_rates_tariff_page() -> str:
    """Retrieves the raw HTML of ComBank's rates & tariff hub page, which
    covers Savings Account and Loan products (unlike fetch_page above,
    this one bundles many product tables under a single URL — see
    parse_savings/parse_loans).
    """
    return _fetch(RATES_TARIFF_URL)


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts tenure/rate pairs from the ComBank rates page HTML. The
    page lists 1-6 month tenures as single rows (paid at maturity by
    definition), and 12/24/36/48/60 month tenures as multiple rows — one
    per payout frequency (monthly/annually/at maturity). Only the "at
    maturity" variant is kept per tenure, so results are directly
    comparable with other banks' headline maturity rates. Where the page
    also lists a bonus "eFD" rate for digital banking customers, that's
    captured as a separate "digital" rate_type record for the same tenure.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in soup.select("table.with-radius tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 2:
            continue  # stray/empty row

        label = cells[0].get_text().strip()
        if not label:
            continue

        lower_label = label.lower()
        has_variant = "interest paid" in lower_label or "interest at" in lower_label
        is_maturity = "at maturity" in lower_label
        if has_variant and not is_maturity:
            continue  # skip "paid monthly"/"paid annually" rows; keep only maturity payout

        try:
            months = ratetext.parse_tenure_months(label)
        except ratetext.ParseError:
            continue

        if len(cells) <= _STANDARD_RATE_COL:
            continue
        try:
            rate = ratetext.parse_rate(cells[_STANDARD_RATE_COL].get_text())
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "tenure_months": months,
                "interest_rate": rate,
                "rate_type": "normal",
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )

        if len(cells) > _EFD_RATE_COL:
            try:
                efd_rate = ratetext.parse_rate(cells[_EFD_RATE_COL].get_text())
            except ratetext.ParseError:
                pass
            else:
                rates.append(
                    {
                        "tenure_months": months,
                        "interest_rate": efd_rate,
                        "rate_type": "digital",
                        "source_url": RATES_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError('combank: no fixed deposit rates parsed from page (selectors likely stale): no rows matched "table.with-radius tbody tr"')
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts savings account rates from ComBank's rates & tariff page.
    Every product under the "Interest Rates" tab is an "expand-block": a
    product name (.expand-link) followed by a single rate table (td[0] =
    row label, td[1] = rate, td[2] = optional AER, ignored — same shape
    the Fixed Deposit table on ComBank's own page uses). One product,
    "Udara Senior Citizens Account", bundles a savings row and two Fixed
    Deposit rows under the same block; rows labelled "fixed deposit" are
    skipped so they don't leak into savings data.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    tab = soup.select_one(_INTEREST_RATES_TAB_SELECTOR)
    if tab is None:
        raise ParseError(f"combank: {_INTEREST_RATES_TAB_SELECTOR!r} tab not found (page structure likely changed)")

    for block in tab.select(".expand-block"):
        link = block.select_one(".expand-link")
        account_name = link.get_text().strip() if link else ""
        if not account_name:
            continue
        lower_name = account_name.lower()
        if any(kw in lower_name for kw in _SAVINGS_SKIP_KEYWORDS):
            continue

        for row in block.select("table tbody tr"):
            cells = row.find_all("td")
            if len(cells) < 2:
                continue

            label = cells[0].get_text().strip()
            if not label:
                continue
            if "fixed deposit" in label.lower():
                continue  # e.g. Udara Senior Citizens Account's bundled FD rows

            balance_tier = label
            lower_label = label.lower()
            if any(kw in lower_label for kw in ("interest paid", "interest at", "int paid", "int at")):
                balance_tier = ""  # payout-frequency boilerplate, not a real tier

            try:
                rate = ratetext.parse_flat_rate(cells[1].get_text())
            except ratetext.ParseError:
                continue

            rates.append(
                {
                    "account_name": account_name,
                    "balance_tier": balance_tier,
                    "interest_rate": rate,
                    "source_url": RATES_TARIFF_URL,
                    "scraped_at": scrape_at,
                }
            )

    if not rates:
        raise ParseError("combank: no savings rates parsed from rates-tariff page (selectors likely stale)")
    return rates


def parse_loans(html: str) -> list[dict]:
    """Extracts loan rates from ComBank's rates & tariff page. Each
    product is an "expand-block" (.expand-link name + a table) in one of
    two shapes:
      - 2-column: Description -> Interest Rate (Per Annum). td[0] is the
        row's own label (e.g. "Short Term Gold Loans -03 Months").
      - matrix: a tier/type column (td[0], e.g. "Standard"/"Premium") plus
        one rate column per tenure ("1 Years" ... "6-7 Years").

    td[0] is always treated as the row label (-> rate_label); every other
    column is a rate candidate, paired with the table's header text at the
    same offset counting from the right (handles the matrix header row
    being one cell short of the data row, since its leading rowspan cell
    isn't repeated). Tenure is left blank for 2-column tables, since there
    the "header" is just "Interest Rate (Per Annum)", not a real tenure.
    Cells that aren't a single flat percentage — floating formulas
    ("AWPLR + 2.50%"), "-", "N/A", commission add-ons — fail
    ratetext.parse_flat_rate and are skipped without special-casing.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    tab = soup.select_one(_LENDING_RATES_TAB_SELECTOR)
    if tab is None:
        raise ParseError(f"combank: {_LENDING_RATES_TAB_SELECTOR!r} tab not found (page structure likely changed)")

    for block in tab.select(".expand-block"):
        link = block.select_one(".expand-link")
        product_name = link.get_text().strip() if link else ""
        if not product_name:
            continue

        table = block.find("table")
        if table is None:
            continue

        theads = table.select("thead tr")
        headers = [th.get_text().strip() for th in theads[-1].find_all("th")] if theads else []

        for row in table.select("tbody tr"):
            cells = row.find_all("td")
            n = len(cells)
            if n < 2:
                continue  # banner/footnote row (colspan collapses it to one <td>)

            label = cells[0].get_text().strip()
            rate_col_count = n - 1

            for ci in range(1, n):
                try:
                    rate = ratetext.parse_flat_rate(cells[ci].get_text())
                except ratetext.ParseError:
                    continue  # floating formula, range, dash, or free text

                tenure = ""
                if rate_col_count > 1:
                    hi = len(headers) - (n - ci)
                    if 0 <= hi < len(headers):
                        tenure = headers[hi]

                rates.append(
                    {
                        "loan_category": product_name,
                        "loan_product": product_name,
                        "rate_label": label,
                        "tenure": tenure,
                        "interest_rate": rate,
                        "source_url": RATES_TARIFF_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError("combank: no loan rates parsed from rates-tariff page (selectors likely stale)")
    return rates

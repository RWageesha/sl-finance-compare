"""Scrapes Sampath Bank Fixed Deposit / Savings / Loan rates.

Sampath's site sits behind an F5 WAF that returns 503 "The requested URL
was rejected" for plain HTTP requests — confirmed by hand, including a
python-requests call with a realistic browser User-Agent. FD and Savings
pages are fetched with a real headless Chromium via Playwright instead:
the WAF's rule keys off browser fingerprint rather than IP reputation (a
real browser passes even from the same machine curl fails from). Loan
rates come from a separate, plain-curl-accessible PDF the bank publishes,
so no WAF workaround is needed there.

Two real, hand-verified quirks shape what this module scrapes:
- The site's own "Rates & Charges" hub page (sampath.lk/rates-and-charges)
  has a genuine bug: its "Term Deposits Rates" and "Loan Rates" tabs
  render identical content to the "Savings Rates" tab (verified by
  clicking each tab in a real browser) — this isn't a scraping artifact,
  it's what a real visitor sees today. FD rates are pulled from Sampath's
  dedicated Fixed-Deposits product page instead, which has the real
  table. The hub page's savings tables are a mix of clean flat-rate
  tables and non-rate "bonus %" tier tables with no headings to tell them
  apart programmatically — this module only scrapes "Hit Saver" (the one
  unambiguous, cleanly-tiered savings table); the others are left out
  rather than risked or guessed.
- The loans PDF's table has deeply merged cells (multi-tenure grids for
  Sevana Housing Loans, a two-row Gold Loan tier, secured/unsecured
  splits) that don't reduce to one product+rate pair — parse_loans keeps
  only the rows that do, skipping the rest rather than reconstructing
  them and risking a wrong mapping.
"""

from __future__ import annotations

import datetime as dt
import re

import pdfplumber
import requests
from bs4 import BeautifulSoup
from playwright.sync_api import sync_playwright

import ratetext

BANK_NAME = "Sampath Bank"
BANK_CODE = "SAMPATH"

FD_URL = "https://www.sampath.lk/personal-banking/term-deposit-accounts/regular-deposits/Fixed-Deposits?category=personal_banking"
SAVINGS_URL = "https://www.sampath.lk/rates-and-charges"
LOANS_PDF_URL = "https://www.sampath.lk/common/loan/interest-rates-loan-and-advances.pdf"

_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
_REQUEST_TIMEOUT = 20


class ParseError(ValueError):
    pass


def _fetch_rendered(url: str) -> str:
    """Fetches url with a real headless Chromium — see module docstring
    for why plain requests can't get past Sampath's WAF.
    """
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=["--disable-blink-features=AutomationControlled"])
        try:
            page = browser.new_page(user_agent=_UA)
            page.add_init_script("Object.defineProperty(navigator, 'webdriver', { get: () => undefined })")
            page.goto(url, wait_until="networkidle", timeout=45000)
            page.wait_for_timeout(1000)
            return page.content()
        finally:
            browser.close()


def fetch_fd_page() -> str:
    return _fetch_rendered(FD_URL)


def fetch_savings_page() -> str:
    return _fetch_rendered(SAVINGS_URL)


def fetch_loans_pdf() -> bytes:
    resp = requests.get(LOANS_PDF_URL, headers={"User-Agent": _UA}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.content


def _first_flat_rate(cell_text: str) -> float:
    """FD rate cells pack the nominal rate and AER together, e.g.
    "8% (AER 8.3%)" or a bare "-" when a payout variant isn't offered —
    only the first whitespace-separated token is ever the nominal rate.
    """
    first_token = cell_text.split()[0] if cell_text.split() else ""
    return ratetext.parse_flat_rate(first_token)


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts the Normal Fixed Deposit table: [Period, At Maturity,
    Monthly, Annually]. Only "At Maturity" is kept as the standard rate,
    the usual "keep the Maturity/AER payout" convention. Several day-based
    tenures (e.g. "75 Days") sit alongside month-based ones and round to
    the same tenure_months as an unrelated product — tenure_label keeps
    the bank's own wording so they don't collide, same as ndb.py.
    """
    soup = BeautifulSoup(html, "lxml")
    table = None
    for t in soup.find_all("table"):
        header_cells = t.find("tr")
        if header_cells and header_cells.find_all(["td", "th"])[0].get_text(strip=True) == "Period":
            header_text = " ".join(c.get_text(" ", strip=True) for c in header_cells.find_all(["td", "th"]))
            if "Interest Paid at Maturity" in header_text:
                table = t
                break
    if table is None:
        raise ParseError("sampath: Fixed Deposit table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr")[1:]:
        cells = row.find_all(["td", "th"])
        if len(cells) != 4:
            continue
        period_text = cells[0].get_text(" ", strip=True)
        try:
            months = ratetext.parse_tenure_months(period_text)
        except ratetext.ParseError:
            continue

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
                "source_url": FD_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("sampath: no fixed deposit rates parsed (selectors likely stale)")
    return rates


def parse_savings(html: str) -> list[dict]:
    """Extracts only the "Hit Saver" savings table: [Monthly Average
    Balance, Interest] — the one savings table on the rates-and-charges
    hub page that is both unambiguous (a distinctive header no other
    table shares) and a real flat per-tier rate (not a bonus multiplier
    like the Double S table sitting right next to it).
    """
    soup = BeautifulSoup(html, "lxml")
    table = None
    for t in soup.find_all("table"):
        rows = t.find_all("tr")
        if not rows:
            continue
        header = [c.get_text(" ", strip=True) for c in rows[0].find_all(["td", "th"])]
        if header == ["Monthly Average Balance", "Interest"]:
            table = t
            break
    if table is None:
        raise ParseError("sampath: Hit Saver savings table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tr")[1:]:
        cells = row.find_all(["td", "th"])
        if len(cells) != 2:
            continue
        tier = cells[0].get_text(strip=True)
        try:
            rate = ratetext.parse_flat_rate(cells[1].get_text(strip=True))
        except ratetext.ParseError:
            continue
        rates.append(
            {
                "account_name": "Hit Saver",
                "balance_tier": tier,
                "interest_rate": rate,
                "source_url": SAVINGS_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("sampath: no savings rates parsed (selectors likely stale)")
    return rates


_TENURE_LIKE_RE = re.compile(r"^\d")


def parse_loans(pdf_bytes: bytes) -> list[dict]:
    """Extracts loan rates from the "Interest Rates on New Loans and
    Advances" PDF. Table extraction reduces each row to its non-empty
    cells; a row that reduces to exactly [category, product, rate] starts
    a new category, [product, rate] continues the current one, and
    anything else (multi-tenure grids, a rate split across several
    sub-rows) is skipped rather than force-reconstructed — see the module
    docstring. Rows whose "product" is actually a stray tenure fragment
    from a skipped multi-row block (e.g. "06 Months") are dropped too,
    since attaching a real rate to that alone would misrepresent it.
    """
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)
    current_category = "Interest Rates on New Loans and Advances"

    import io

    with pdfplumber.open(io.BytesIO(pdf_bytes)) as pdf:
        for page in pdf.pages:
            for table in page.extract_tables():
                for row in table:
                    cells = [c.strip() for c in row if c and c.strip()]
                    if len(cells) == 3:
                        current_category, product, rate_text = cells
                    elif len(cells) == 2:
                        product, rate_text = cells
                    else:
                        # A merged multi-tenure/multi-tier block we don't
                        # try to reconstruct (see module docstring). Its
                        # first cell is sometimes a real category label
                        # and sometimes a sub-row label (e.g. "With Salary
                        # Remittance") that would corrupt current_category
                        # for later rows — cell count alone can't tell
                        # these apart, so it's left untouched here; a
                        # couple of rows keep the previous section's
                        # category rather than their own as a result.
                        continue

                    if _TENURE_LIKE_RE.match(product):
                        continue  # a fragment of a multi-tenure row we can't cleanly reconstruct

                    try:
                        rate = ratetext.parse_flat_rate(rate_text)
                    except ratetext.ParseError:
                        continue

                    rates.append(
                        {
                            "loan_category": current_category.replace("\n", " "),
                            "loan_product": product.replace("\n", " "),
                            "rate_label": "",
                            "tenure": "",
                            "interest_rate": rate,
                            "source_url": LOANS_PDF_URL,
                            "scraped_at": scrape_at,
                        }
                    )

    if not rates:
        raise ParseError("sampath: no loan rates parsed (PDF layout likely changed)")
    return rates

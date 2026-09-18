"""Scrapes Loan rates from DFCC Bank's public rates & tariff page.

DFCC's site is a Next.js app that streams its data as inline React
Server Component JSON rather than plain server-rendered HTML — curl gets
a 200 with the full page, but the actual rate tables don't exist in that
raw HTML at all (confirmed by hand: zero <table> elements, real figures
only as escaped strings inside <script> payloads). A real headless
Chromium is used instead so the page hydrates into its final DOM, the
same general "needs a real browser" situation as Sampath/Pan Asia, just
for a different reason (JS hydration, not a WAF).

Only Loan rates are scraped. This page's three visible tables (checked by
hand, including scrolling/waiting for hydration) are all lending-rate
tables — Overdrafts; Cards/Pawning/Personal Loans; Housing Loans. No
Fixed Deposit or Savings rate table renders anywhere on this page or on
DFCC's dedicated Fixed Deposits product page (also checked directly) —
whatever mechanism DFCC uses to publish those isn't a scrapable HTML
table, so nothing is guessed here; only what's genuinely visible is kept.
"""

from __future__ import annotations

import datetime as dt

from bs4 import BeautifulSoup
from playwright.sync_api import sync_playwright

import ratetext

BANK_NAME = "DFCC Bank"
BANK_CODE = "DFCC"

RATES_URL = "https://www.dfcc.lk/rates-and-tariff"

_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"


class ParseError(ValueError):
    pass


def fetch_rates_page() -> str:
    """Fetches RATES_URL with a real headless Chromium — see module
    docstring for why plain requests won't have the rate tables at all.
    """
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=["--disable-blink-features=AutomationControlled"])
        try:
            page = browser.new_page(user_agent=_UA)
            page.add_init_script("Object.defineProperty(navigator, 'webdriver', { get: () => undefined })")
            page.goto(RATES_URL, wait_until="networkidle", timeout=45000)
            page.wait_for_timeout(1000)
            return page.content()
        finally:
            browser.close()


def parse_loans(html: str) -> list[dict]:
    """Extracts every [Product, Min Rate, Max Rate] table on the page —
    each becomes two rows (Min Rate, Max Rate), the same convention NDB
    and People's Bank use for a bank-published range. AWPLR-linked
    formulas and "-" placeholders aren't a single comparable percentage
    and are skipped, same as every other scraper.
    """
    soup = BeautifulSoup(html, "lxml")
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for table in soup.find_all("table"):
        rows = table.find_all("tr")
        if not rows:
            continue
        header = [c.get_text(strip=True) for c in rows[0].find_all(["td", "th"])]
        if header[:1] != ["Product"]:
            continue  # not one of the rate tables

        for row in rows[1:]:
            cells = row.find_all(["td", "th"])
            if len(cells) != 3:
                continue
            product = cells[0].get_text(strip=True)
            for rate_label, cell in (("Min Rate", cells[1]), ("Max Rate", cells[2])):
                try:
                    rate = ratetext.parse_flat_rate(cell.get_text(strip=True))
                except ratetext.ParseError:
                    continue
                rates.append(
                    {
                        "loan_category": product,
                        "loan_product": product,
                        "rate_label": rate_label,
                        "tenure": "",
                        "interest_rate": rate,
                        "source_url": RATES_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError("dfcc: no loan rates parsed (selectors likely stale)")
    return rates

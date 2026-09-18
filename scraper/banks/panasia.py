"""Scrapes Pan Asia Banking Corporation's Fixed Deposit rates.

Pan Asia's site sits behind a Sucuri WAF that returns a 307 redirect with
no Location header for plain HTTP requests — confirmed by hand, including
a python-requests call with a realistic browser User-Agent. The rates
page is fetched with a real headless Chromium via Playwright instead (see
sampath.py for the same pattern and why it works). Sucuri's rule here
also appears to rate-limit or temporarily blacklist a fingerprint after
repeated requests in a short window — hand-testing saw the same URL
return 200 several times, then start 307-ing even on a fresh browser
context, then presumably recover after a cooldown. _run_scrape already
treats a single source's failure as independent of the others, so this
surfaces as an honest "failed" scrape_runs row rather than blocking
anything — worth knowing if this bank's data looks stale.

Only Fixed Deposits are covered so far — no savings/loan rates page for
Pan Asia has been located and verified yet (the WAF's rate-limiting cut
research short); add banks/panasia.py's savings/loan parsing once that's
done, following the same pattern as parse_fixed_deposits.
"""

from __future__ import annotations

import datetime as dt

from bs4 import BeautifulSoup
from playwright.sync_api import sync_playwright

import ratetext

BANK_NAME = "Pan Asia Banking Corporation"
BANK_CODE = "PANASIA"

FD_URL = "https://www.pabcbank.com/fixed-deposits-rates/"

_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"


class ParseError(ValueError):
    pass


def fetch_fd_page() -> str:
    """Fetches FD_URL with a real headless Chromium — see module
    docstring for why plain requests can't get past Pan Asia's WAF.
    """
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, args=["--disable-blink-features=AutomationControlled"])
        try:
            page = browser.new_page(user_agent=_UA)
            page.add_init_script("Object.defineProperty(navigator, 'webdriver', { get: () => undefined })")
            resp = page.goto(FD_URL, wait_until="networkidle", timeout=45000)
            if resp is not None and resp.status >= 400:
                raise ParseError(f"panasia: fixed deposits page returned HTTP {resp.status} (WAF block or rate limit)")
            page.wait_for_timeout(1000)
            return page.content()
        finally:
            browser.close()


def _expand_colspan_row(row) -> list[str]:
    """Expands a header row's cells into one text value per column,
    repeating a colspan cell's text across the columns it spans — needed
    because the tenure header row (e.g. "12 Months" spanning its Monthly/
    Biannual/Maturity payout sub-columns) has fewer <td>s than the rows
    beneath it.
    """
    expanded: list[str] = []
    for cell in row.find_all(["td", "th"]):
        span = int(cell.get("colspan") or 1)
        text = cell.get_text(strip=True)
        expanded.extend([text] * span)
    return expanded


def _table_groups(soup: BeautifulSoup) -> list[tuple[str, list]]:
    """Groups this page's tables by the nearest preceding label text that
    isn't itself a rate value or the "With effective from" banner —
    there's no heading tag between table groups, just plain text, and a
    rate table is split across 2-3 <table> elements with no marker
    between them (each new <table> after the first in a group is directly
    preceded by the previous table's own last rate/AER cell).
    """
    groups: list[tuple[str, list]] = []
    current_label = ""
    current_tables: list = []
    for table in soup.find_all("table"):
        marker = ""
        node = table
        for _ in range(5):  # skip blank/whitespace-only text nodes between tables
            node = node.find_previous(string=True)
            if node is None:
                break
            text = node.strip()
            if text:
                marker = text
                break
        is_continuation = marker == "" or marker.endswith("%") or marker.lower().startswith("with effective")
        if not is_continuation:
            if current_tables:
                groups.append((current_label, current_tables))
            current_label = marker
            current_tables = [table]
        else:
            current_tables.append(table)
    if current_tables:
        groups.append((current_label, current_tables))
    return groups


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts Fixed Deposit rates from the page's pivoted rate tables:
    a tenure header row (colspan per tenure, one sub-column per payout
    variant), a payout-variant sub-header row, and Int.Rate/AER rows.
    Only the "Maturity" payout column is kept per tenure, the usual
    convention. Two of this page's three table groups are skipped: the
    "Rates for Fixed Deposits open through Pan Asia Internet Banking"
    group (a +0.25% channel bonus on the same tenures, not a distinct
    product) and any table lacking the expected row shape; the "Women's
    Fixed Deposit" group is kept as a real, distinct, named product
    (rate_type "special").
    """
    soup = BeautifulSoup(html, "lxml")
    groups = _table_groups(soup)
    if not groups:
        raise ParseError("panasia: no rate table groups found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for label, tables in groups:
        if "internet banking" in label.lower():
            continue
        rate_type = "special" if "women" in label.lower() else "normal"

        for table in tables:
            rows = table.find_all("tr")
            if len(rows) < 3:
                continue
            tenure_labels = _expand_colspan_row(rows[0])[1:]
            payment_labels = [c.get_text(strip=True) for c in rows[1].find_all(["td", "th"])][1:]
            rate_cells = [c.get_text(strip=True) for c in rows[2].find_all(["td", "th"])][1:]
            if not (len(tenure_labels) == len(payment_labels) == len(rate_cells)) or not tenure_labels:
                continue

            for tenure_text, payment, rate_text in zip(tenure_labels, payment_labels, rate_cells):
                if payment.lower() != "maturity":
                    continue
                try:
                    months = ratetext.parse_tenure_months(tenure_text)
                except ratetext.ParseError:
                    continue
                try:
                    rate = ratetext.parse_flat_rate(rate_text)
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
        raise ParseError("panasia: no fixed deposit rates parsed (selectors likely stale)")
    return rates

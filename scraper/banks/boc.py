"""Scrapes Fixed Deposit / Savings / Loan interest rates from Bank of
Ceylon's public rates & tariff page. The page renders server-side with
real <table> markup, but each cell holds trilingual text (Sinhala/Tamil/
English) separated by <br> tags — usually with English last, though the
Housing Loans table orders it first — so cells need a bit more than plain
text extraction to isolate the English label.

Port of internal/scrapers/boc/{fd,savings,loans}.go.
"""

from __future__ import annotations

import datetime as dt
import html as stdhtml
import re

import requests
from bs4 import BeautifulSoup
from bs4.element import Tag

import ratetext

BANK_NAME = "Bank of Ceylon"
BANK_CODE = "BOC"

RATES_URL = "https://www.boc.lk/rates-tariff"
_REQUEST_TIMEOUT = 15
# This site's WAF rejects requests without a browser-like UA (403).
_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

# The rates & tariff page bundles many unrelated sections (exchange rates,
# loan rates, savings, insurance info, a suspended senior citizen scheme, a
# scheme terminated in 2009, etc). This is the only FD section title we
# want — it's the current, active rupee FD table available for new
# deposits.
_FIXED_DEPOSITS_SECTION_TITLE = "Rupee Fixed Deposits"
_STANDARD_RATE_COL = 1

_SAVINGS_SECTION_TITLE = "Rupee – Savings Deposits"
_SAVINGS_ACCOUNT_TABLE = "SAVINGS DEPOSIT ACCOUNT"

# Loan sections on this page vary wildly in shape, and not every one is a
# clean, comparable, currently-open rate:
#   - Personal Loans, Housing Loans, Ran Surekum Naya Seva (pawning),
#     Leasing, and Government Pensioners' Loan Scheme are included below.
#   - Education Loans is skipped: a single product whose rate cell embeds
#     a trilingual label *and* two tenure/rate pairs in a one-off nested
#     shape not worth a bespoke parser for one product.
#   - Development Loans is skipped: an MSME/agriculture grab-bag of ~15
#     differently-shaped sub-tables (same call as skipping HNB's SME
#     division).
#   - Advances Against the Deposit is skipped: its rates are a margin over
#     AER/Repo, not an absolute rate — not comparable, same principle as
#     skipping AWPLR-formula cells everywhere else.
#   - Credit Cards is skipped: fees/APR, not an interest rate to compare.
_PERSONAL_LOANS_TITLE = "Personal Loans"
_HOUSING_LOANS_TITLE = "Housing Loans"
_RAN_SUREKUM_NAYA_SEVA_TITLE = "Ran Surekum Naya Seva"
_LEASING_TITLE = "Leasing"
_GOVERNMENT_PENSIONERS_LOAN_TITLE = "Government Pensioners’ Loan Scheme"

_BR_SPLIT_RE = re.compile(r"(?i)<br\s*/?>")
_TAG_STRIP_RE = re.compile(r"<[^>]+>")
# Strips a trailing "p.a."/"p.a" unit annotation the savings table's rate
# cells carry (e.g. "2.00% p.a."), which ratetext.parse_flat_rate
# otherwise correctly rejects as not a bare percentage.
_PA_SUFFIX_RE = re.compile(r"(?i)\s*p\.a\.?\s*$")
# Matches one "<tenure> : <rate>%" segment, e.g. "Upto 5 Years : 14.00%"
# or "Above 7 years to 10 years: 15.50%".
_TENURE_RATE_COLON_RE = re.compile(r"^(.*?)\s*:\s*(\d+(?:\.\d+)?)\s*%\s*$")


class ParseError(ValueError):
    pass


def fetch_page() -> str:
    """Retrieves the raw HTML of BOC's rates & tariff page."""
    resp = requests.get(RATES_URL, headers={"User-Agent": _UA, "Accept": "text/html"}, timeout=_REQUEST_TIMEOUT)
    resp.raise_for_status()
    return resp.text


# --- Trilingual cell text helpers ------------------------------------------


def _clean_br_segment(segment: str) -> str:
    """Strips tags and normalizes whitespace/entities in one
    <br>-separated segment of a table cell's inner HTML. html.unescape
    decodes every HTML entity (not just &nbsp; — e.g. &#39; in
    "Teenagers&#39; Savings" becomes an apostrophe), including turning
    &nbsp; into a real non-breaking space, which str.split() then treats
    as ordinary whitespace to collapse.
    """
    s = _TAG_STRIP_RE.sub("", segment)
    s = stdhtml.unescape(s)
    return " ".join(s.split())


def _last_br_segment_text(cell_html: str) -> str:
    """Takes a table cell's inner HTML containing <br>-separated
    multilingual text and returns just the last segment (English, by this
    page's usual convention), with tags stripped and entities normalized.
    """
    parts = _BR_SPLIT_RE.split(cell_html)
    return _clean_br_segment(parts[-1])


def _is_ascii_text(s: str) -> bool:
    return all(ord(c) < 128 for c in s)


def _english_br_segment(cell_html: str) -> str:
    """Returns the run of <br>-separated segments in cell_html that are
    plain ASCII text, joined back together. Most tables on this page order
    their trilingual cells Sinhala/Tamil/English (English last — see
    _last_br_segment_text), but the Housing Loans table orders its
    Type-of-loan and Period cells English/Sinhala/Tamil (English first) —
    and a couple of its English labels are themselves wrapped across two
    consecutive <br> segments ("Aggregate Housing Loan amount from" /
    "Rs. 5.0 Mn up to Rs. 7.5 Mn"), so a single segment isn't always
    enough. Rather than hardcode a position, this collects whichever
    contiguous run of segments actually looks like English, wherever it
    falls.
    """
    run: list[str] = []
    for seg in _BR_SPLIT_RE.split(cell_html):
        text = _clean_br_segment(seg)
        if not text:
            continue
        if _is_ascii_text(text):
            run.append(text)
            continue
        if run:
            break  # a non-ASCII segment ends the English run
    if run:
        return " ".join(run)
    return _last_br_segment_text(cell_html)  # fallback: single-segment cells


class _TenureRate:
    __slots__ = ("tenure", "rate")

    def __init__(self, tenure: str = "", rate: str = ""):
        self.tenure = tenure
        self.rate = rate


def _parse_br_separated_rate_pairs(cell_html: str) -> list[_TenureRate]:
    """Splits a cell's inner HTML on <br>, and reads each resulting
    segment either as a "<tenure> : <rate>%" pair (BOC's Personal Loan
    cells pack several tenure brackets into one cell this way) or, when a
    segment has no such separator, as a single bare rate value with no
    tenure (covers simpler cells that are just "15.00%" with no <br> at
    all, e.g. Ran Surekum Naya Seva).
    """
    out = []
    for seg in _BR_SPLIT_RE.split(cell_html):
        text = _clean_br_segment(seg)
        if not text:
            continue
        m = _TENURE_RATE_COLON_RE.match(text)
        if m:
            out.append(_TenureRate(tenure=m.group(1).strip(), rate=m.group(2) + "%"))
        else:
            out.append(_TenureRate(rate=text))
    return out


def _next_tag(el: Tag) -> Tag | None:
    """BeautifulSoup's .next_sibling includes whitespace-only
    NavigableStrings between tags; goquery's .Next() (which the ported Go
    code relies on for the h3 -> h4 -> wrap-div traversal) skips straight
    to the next element. This replicates the element-only behavior.
    """
    sib = el.next_sibling
    while sib is not None and not isinstance(sib, Tag):
        sib = sib.next_sibling
    return sib


def _find_section_wrap(soup: BeautifulSoup, title: str) -> Tag | None:
    """Locates the h3 titled `title` and returns the div holding its
    content — the page's repeating structure is
    <h3 class="sub-title">Title</h3><h4 .../><div class="exchangerate-table-wrap">...
    """
    for h3 in soup.select("h3.sub-title"):
        if h3.get_text().strip() != title:
            continue
        return _next_tag(_next_tag(h3))  # h4 -> div.exchangerate-table-wrap
    return None


# --- Fixed deposits -------------------------------------------------------


def _find_fixed_deposits_table(soup: BeautifulSoup) -> Tag | None:
    for h3 in soup.select("h3.sub-title"):
        if h3.get_text().strip() != _FIXED_DEPOSITS_SECTION_TITLE:
            continue
        wrap = _next_tag(_next_tag(h3))
        if wrap is None:
            return None
        return wrap.select_one("table.ck-table-resized")
    return None


def parse_fixed_deposits(html: str) -> list[dict]:
    """Extracts tenure/rate pairs from the "Rupee Fixed Deposits" section.
    Like ComBank, tenures of 1 year and above get multiple rows per tenure
    — one per payout frequency (monthly/annually/at maturity) — and only
    the "at maturity" variant is kept per tenure so results stay
    comparable across banks. Rows explicitly for "Senior Citizens" are
    captured separately as rate_type "senior".
    """
    soup = BeautifulSoup(html, "lxml")
    table = _find_fixed_deposits_table(soup)
    if table is None:
        raise ParseError(f"boc: {_FIXED_DEPOSITS_SECTION_TITLE!r} section/table not found (page structure likely changed)")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) <= _STANDARD_RATE_COL:
            continue

        label = _last_br_segment_text(cells[0].decode_contents())
        if not label:
            continue  # stray/empty row

        lower_label = label.lower()
        has_variant = "interest paid" in lower_label or "interest at" in lower_label
        is_maturity = "at maturity" in lower_label
        if has_variant and not is_maturity:
            continue  # skip "paid monthly"/"paid annually" rows; keep only maturity payout

        rate_type = "senior" if "senior" in lower_label else "normal"

        try:
            months = ratetext.parse_tenure_months(label)
            rate = ratetext.parse_rate(cells[_STANDARD_RATE_COL].get_text())
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "tenure_months": months,
                "interest_rate": rate,
                "rate_type": rate_type,
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )

    if not rates:
        raise ParseError("boc: no fixed deposit rates parsed from page (selectors likely stale)")
    return rates


# --- Savings ---------------------------------------------------------------


def parse_savings(html: str) -> list[dict]:
    """Extracts savings account rates from the "Rupee – Savings Deposits"
    section. That section actually bundles three tables: a Savings
    Certificate discount-price table (existing customers only, and a
    completely different shape — skipped by only targeting the specific
    "SAVINGS DEPOSIT ACCOUNT" table by its banner text), the flat-rate
    savings account table this function reads, and an "Other Deposit
    Schemes" table whose rows are both explicitly "now suspended"
    (ratetext.parse_flat_rate rejects that text automatically, no
    special-casing needed — though it's moot since that table isn't the
    one this function looks at anyway).
    """
    soup = BeautifulSoup(html, "lxml")
    wrap = _find_section_wrap(soup, _SAVINGS_SECTION_TITLE)
    if wrap is None:
        raise ParseError(f"boc: {_SAVINGS_SECTION_TITLE!r} section not found (page structure likely changed)")

    table = None
    for t in wrap.select("table"):
        if _SAVINGS_ACCOUNT_TABLE in t.get_text():
            table = t
            break
    if table is None:
        raise ParseError(f"boc: {_SAVINGS_ACCOUNT_TABLE!r} table not found within {_SAVINGS_SECTION_TITLE!r} section")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 2:
            continue

        account_name = _last_br_segment_text(cells[0].decode_contents())
        if not account_name:
            continue  # the table's own banner row, or a stray/empty row

        rate_text = _PA_SUFFIX_RE.sub("", cells[1].get_text())
        try:
            rate = ratetext.parse_flat_rate(rate_text)
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
        raise ParseError(f"boc: no savings rates parsed from {_SAVINGS_ACCOUNT_TABLE!r} table")
    return rates


# --- Loans -------------------------------------------------------------


def _parse_personal_loans(soup: BeautifulSoup, scrape_at: dt.datetime) -> list[dict]:
    """Two products (BOC Personal Loan Scheme, BOC Special Personal Loan
    Scheme), each a single row whose rate cell packs several tenure:rate
    segments together.
    """
    wrap = _find_section_wrap(soup, _PERSONAL_LOANS_TITLE)
    if wrap is None:
        return []

    table = wrap.find("table")
    if table is None:
        return []

    rates: list[dict] = []
    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 2:
            continue

        product_name = _last_br_segment_text(cells[0].decode_contents())
        if not product_name:
            continue

        for pair in _parse_br_separated_rate_pairs(cells[1].decode_contents()):
            try:
                rate = ratetext.parse_flat_rate(pair.rate)
            except ratetext.ParseError:
                continue
            rates.append(
                {
                    "loan_category": _PERSONAL_LOANS_TITLE,
                    "loan_product": product_name,
                    "rate_label": "Fixed Rate",
                    "tenure": pair.tenure,
                    "interest_rate": rate,
                    "source_url": RATES_URL,
                    "scraped_at": scrape_at,
                }
            )
    return rates


def _parse_housing_loans(soup: BeautifulSoup, scrape_at: dt.datetime) -> list[dict]:
    """The rowspan-heavy table: S/N and Type-of-loan cells span 2-3 rows,
    so only the first row of each group has 4 <td>s — continuation rows
    have just 2 (repayment period, rate) and reuse the last-seen type.
    Single-<td> rows are section banners (colspan) and are skipped.
    """
    wrap = _find_section_wrap(soup, _HOUSING_LOANS_TITLE)
    if wrap is None:
        return []

    table = wrap.find("table")
    if table is None:
        return []

    rates: list[dict] = []
    current_type = ""

    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        n = len(cells)

        if n == 4:
            current_type = _english_br_segment(cells[1].decode_contents())
            period_cell, rate_cell = cells[2], cells[3]
        elif n == 2:
            if not current_type:
                continue
            period_cell, rate_cell = cells[0], cells[1]
        else:
            continue  # banner row (colspan) or unexpected shape

        period = _english_br_segment(period_cell.decode_contents())

        try:
            rate = ratetext.parse_flat_rate(rate_cell.get_text())
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "loan_category": _HOUSING_LOANS_TITLE,
                "loan_product": current_type,
                "rate_label": "Fixed Rate",
                "tenure": period,
                "interest_rate": rate,
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )
    return rates


def _parse_ran_surekum_naya_seva(soup: BeautifulSoup, scrape_at: dt.datetime) -> list[dict]:
    """The pawning/gold loan section: one product, one flat rate."""
    wrap = _find_section_wrap(soup, _RAN_SUREKUM_NAYA_SEVA_TITLE)
    if wrap is None:
        return []

    table = wrap.find("table")
    if table is None:
        return []

    rates: list[dict] = []
    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 2:
            continue

        product_name = _last_br_segment_text(cells[0].decode_contents())
        if not product_name:
            continue  # the table's own banner row

        try:
            rate = ratetext.parse_flat_rate(cells[1].get_text())
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "loan_category": _RAN_SUREKUM_NAYA_SEVA_TITLE,
                "loan_product": product_name,
                "rate_label": "Interest Rate",
                "tenure": "",
                "interest_rate": rate,
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )
    return rates


def _parse_leasing(soup: BeautifulSoup, scrape_at: dt.datetime) -> list[dict]:
    """Three vehicle/machinery types, each with a min and max rate
    column.
    """
    wrap = _find_section_wrap(soup, _LEASING_TITLE)
    if wrap is None:
        return []

    table = wrap.find("table")
    if table is None:
        return []

    rates: list[dict] = []
    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 3:
            continue

        type_name = _last_br_segment_text(cells[0].decode_contents())
        if not type_name:
            continue

        for idx, label in enumerate(("Min Rate", "Max Rate")):
            try:
                rate = ratetext.parse_flat_rate(cells[idx + 1].get_text())
            except ratetext.ParseError:
                continue
            rates.append(
                {
                    "loan_category": _LEASING_TITLE,
                    "loan_product": type_name,
                    "rate_label": label,
                    "tenure": "",
                    "interest_rate": rate,
                    "source_url": RATES_URL,
                    "scraped_at": scrape_at,
                }
            )
    return rates


def _parse_government_pensioners_loan_scheme(soup: BeautifulSoup, scrape_at: dt.datetime) -> list[dict]:
    """One product, three tenure brackets each with its own flat rate."""
    wrap = _find_section_wrap(soup, _GOVERNMENT_PENSIONERS_LOAN_TITLE)
    if wrap is None:
        return []

    table = wrap.find("table")
    if table is None:
        return []

    rates: list[dict] = []
    for row in table.select("tbody tr"):
        cells = row.find_all("td")
        if len(cells) < 2:
            continue

        period = _last_br_segment_text(cells[0].decode_contents())
        if not period:
            continue  # the table's own banner row

        try:
            rate = ratetext.parse_flat_rate(cells[1].get_text())
        except ratetext.ParseError:
            continue

        rates.append(
            {
                "loan_category": _GOVERNMENT_PENSIONERS_LOAN_TITLE,
                "loan_product": _GOVERNMENT_PENSIONERS_LOAN_TITLE,
                "rate_label": "Fixed Rate",
                "tenure": period,
                "interest_rate": rate,
                "source_url": RATES_URL,
                "scraped_at": scrape_at,
            }
        )
    return rates


def parse_loans(html: str) -> list[dict]:
    """Extracts loan rates from the sections listed in the module-level
    comment above.
    """
    soup = BeautifulSoup(html, "lxml")
    scrape_at = dt.datetime.now(dt.timezone.utc)

    rates: list[dict] = []
    rates += _parse_personal_loans(soup, scrape_at)
    rates += _parse_housing_loans(soup, scrape_at)
    rates += _parse_ran_surekum_naya_seva(soup, scrape_at)
    rates += _parse_leasing(soup, scrape_at)
    rates += _parse_government_pensioners_loan_scheme(soup, scrape_at)

    if not rates:
        raise ParseError("boc: no loan rates parsed from rates-tariff page (selectors likely stale)")
    return rates

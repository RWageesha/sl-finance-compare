"""Scrapes Fixed Deposit / Savings / Loan interest rates from Hatton
National Bank's public rates JSON API.

HNB's rates page (https://www.hnb.lk/fixed-deposits-interest-rates) is a
client-side rendered React app — the server returns an empty HTML shell,
and the actual rate tables are fetched by the browser from a backend API
and rendered with JavaScript. That means an HTML/CSS-selector scraper can
never see the data: there is no table markup in the server response at
all. Instead, this module talks to the same JSON API the page itself
calls. That one response contains all of HNB's published rates (Fixed
Deposits, Savings, Loans, Leasing, Pawning, Treasury), so it's fetched once
by the caller and parsed three ways here.

Port of internal/scrapers/hnb/{fd,savings,loans}.go.
"""

from __future__ import annotations

import datetime as dt
import json

import requests

import ratetext

BANK_NAME = "Hatton National Bank"
BANK_CODE = "HNB"

RATES_API_URL = "https://venus.hnb.lk/api/get_interest_rates_contents"
_REQUEST_TIMEOUT = 15

_FIXED_DEPOSITS_SUB_CATEGORY = "Fixed Deposits"
_SAVINGS_ACCOUNTS_SUB_CATEGORY = "Savings Accounts Interest Rates"
_MONEY_MARKET_SAVINGS_SUB_CATEGORY = "Money Market Savings"
_LOANS_CATEGORY = "Loans"

# Per-division column map for savings tables: which column holds the
# account name (None means "use the division title instead"), which holds
# the balance tier (None means "no balance tier for this division"), and
# which holds the comparable headline rate. Every division also publishes
# an "Effective Annual Rate" column that is intentionally not captured,
# same as ComBank's AER column — one comparable headline rate per row is
# enough.
_SAVINGS_COLUMN_SPECS = {
    "HNB Savings +": {"account_name_col": None, "balance_tier_col": 0, "rate_col": 1},
    "HNB FIT Account": {"account_name_col": None, "balance_tier_col": 1, "rate_col": 2},
    "Savings Accounts Interest Rates": {"account_name_col": 0, "balance_tier_col": None, "rate_col": 1},
    "Money Market Savings Rates": {"account_name_col": 0, "balance_tier_col": None, "rate_col": 1},
}

# Table columns that describe the loan (type/tenure) rather than carry a
# rate value. Every other column is treated as a rate candidate.
_LOAN_LABEL_COLUMN_NAMES = {"type", "tenure", "tenor", "period", "period (years)", "period (year)"}


class ParseError(ValueError):
    pass


def fetch_rates_json() -> dict:
    """Retrieves and JSON-decodes HNB's interest rates API response."""
    resp = requests.get(
        RATES_API_URL,
        headers={
            "User-Agent": "Mozilla/5.0 (compatible; SLFinanceCompareBot/1.0; +https://github.com/)",
            "Accept": "application/json",
        },
        timeout=_REQUEST_TIMEOUT,
    )
    resp.raise_for_status()
    return resp.json()


def _find_sub_category(resp: dict, name: str) -> dict | None:
    """Searches every category in the response for a sub-category with the
    given name (case-insensitive), regardless of which top-level category
    it lives under — HNB's site nests "Fixed Deposits" and the savings
    sub-categories under the "Savings" category, for instance.
    """
    for cat in resp.get("data", []):
        for sc in cat.get("interest_rate_sub_category", []):
            if sc.get("name", "").strip().lower() == name.lower():
                return sc
    return None


def _find_column_index(columns: list[str], name: str) -> int:
    for i, c in enumerate(columns):
        if c.strip().lower() == name.lower():
            return i
    return -1


def _classify_rate_type(title: str) -> str:
    """Maps a table's title to a rate_type based on keywords. Order
    matters: check specific keywords before falling back to "normal",
    since e.g. "Senior Citizen Fixed Deposits" also contains
    "Fixed Deposits".
    """
    t = title.lower()
    if "senior" in t:
        return "senior"
    if "sathkara" in t:
        return "sathkara"
    if "special" in t:
        return "special"
    return "normal"


def parse_fixed_deposits(resp: dict) -> list[dict]:
    """Extracts tenure/rate pairs for the "Fixed Deposits" sub-category,
    picking the "Maturity" column as the canonical comparable rate per
    tenure (falling back to "Rate" for tables that only publish a single
    rate, which is paid at maturity by definition for those products).
    """
    fd_sub = _find_sub_category(resp, _FIXED_DEPOSITS_SUB_CATEGORY)
    if fd_sub is None:
        raise ParseError(f"hnb: {_FIXED_DEPOSITS_SUB_CATEGORY!r} sub-category not found in api response")

    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for div in fd_sub.get("sub_category_division_approved", []):
        rate_type = _classify_rate_type(div.get("title", ""))

        for td in div.get("table_data_approved", []):
            payload = json.loads(td["data"])
            columns = payload.get("columns", [])

            rate_col_idx = _find_column_index(columns, "maturity")
            if rate_col_idx == -1:
                rate_col_idx = _find_column_index(columns, "rate")
            if rate_col_idx == -1:
                continue

            for row in payload.get("rows", []):
                if len(row) <= rate_col_idx:
                    continue

                tenure_text = row[0].strip()
                rate_text = row[rate_col_idx].strip()
                if rate_text in ("", "-"):
                    continue  # this payout frequency isn't offered for this tenure

                try:
                    months = ratetext.parse_tenure_months(tenure_text)
                    rate = ratetext.parse_rate(rate_text)
                except ratetext.ParseError:
                    continue

                rates.append(
                    {
                        "tenure_months": months,
                        "interest_rate": rate,
                        "rate_type": rate_type,
                        "source_url": RATES_API_URL,
                        "scraped_at": scrape_at,
                    }
                )

    if not rates:
        raise ParseError("hnb: no fixed deposit rates parsed from api response (schema likely changed)")
    return rates


def parse_savings_accounts(resp: dict) -> list[dict]:
    """Extracts savings account rate tiers from the "Savings Accounts
    Interest Rates" and "Money Market Savings" sub-categories.
    """
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for sub_name in (_SAVINGS_ACCOUNTS_SUB_CATEGORY, _MONEY_MARKET_SAVINGS_SUB_CATEGORY):
        sub = _find_sub_category(resp, sub_name)
        if sub is None:
            continue

        for div in sub.get("sub_category_division_approved", []):
            title = div.get("title", "").strip()
            spec = _SAVINGS_COLUMN_SPECS.get(title)
            if spec is None:
                continue

            for td in div.get("table_data_approved", []):
                payload = json.loads(td["data"])

                for row in payload.get("rows", []):
                    max_col = max(
                        spec["rate_col"],
                        spec["account_name_col"] if spec["account_name_col"] is not None else -1,
                        spec["balance_tier_col"] if spec["balance_tier_col"] is not None else -1,
                    )
                    if len(row) <= max_col:
                        continue

                    account_name = title if spec["account_name_col"] is None else row[spec["account_name_col"]].strip()
                    if not account_name:
                        continue  # stray/empty row

                    balance_tier = "" if spec["balance_tier_col"] is None else row[spec["balance_tier_col"]].strip()

                    try:
                        rate = ratetext.parse_flat_rate(row[spec["rate_col"]])
                    except ratetext.ParseError:
                        continue  # e.g. the 0.0% / "-" zero-balance tier

                    rates.append(
                        {
                            "account_name": account_name,
                            "balance_tier": balance_tier,
                            "interest_rate": rate,
                            "source_url": RATES_API_URL,
                            "scraped_at": scrape_at,
                        }
                    )

    if not rates:
        raise ParseError("hnb: no savings rates parsed from api response (schema likely changed)")
    return rates


def _classify_loan_columns(columns: list[str]) -> tuple[list[int], list[int]]:
    """Splits a table's columns into label columns (type, tenure/period)
    and rate-candidate columns (everything else), preserving column order.
    """
    label_cols, rate_cols = [], []
    for i, c in enumerate(columns):
        if c.strip().lower() in _LOAN_LABEL_COLUMN_NAMES:
            label_cols.append(i)
        else:
            rate_cols.append(i)
    return label_cols, rate_cols


def parse_loans(resp: dict) -> list[dict]:
    """Extracts loan rates by walking every division under the "Loans"
    category. Real loan tables mix flat percentages with floating-rate
    formulas ("AWPLR + 2.50%"), ranges, and free text ("Please refer
    Treasury Division") — only cells that are a single flat percentage
    (ratetext.parse_flat_rate) are captured; everything else is silently
    skipped since it isn't a single comparable rate. Divisions covering
    foreign-currency loans are skipped entirely so results stay LKR-only,
    matching how the BOC scraper only captures "Rupee Fixed Deposits".
    """
    rates: list[dict] = []
    scrape_at = dt.datetime.now(dt.timezone.utc)

    for cat in resp.get("data", []):
        if cat.get("name", "").strip().lower() != _LOANS_CATEGORY.lower():
            continue

        for sub in cat.get("interest_rate_sub_category", []):
            for div in sub.get("sub_category_division_approved", []):
                title = div.get("title", "").strip()
                if "foreign currency" in title.lower():
                    continue

                for td in div.get("table_data_approved", []):
                    payload = json.loads(td["data"])
                    columns = payload.get("columns", [])
                    label_cols, rate_cols = _classify_loan_columns(columns)

                    for row in payload.get("rows", []):
                        if len(row) < len(columns):
                            continue

                        tenure = " ".join(row[ci].strip() for ci in label_cols if row[ci].strip())

                        for ci in rate_cols:
                            try:
                                rate = ratetext.parse_flat_rate(row[ci])
                            except ratetext.ParseError:
                                continue  # floating formula, range, dash, or free text

                            rates.append(
                                {
                                    "loan_category": sub.get("name", "").strip(),
                                    "loan_product": title,
                                    "rate_label": columns[ci].strip(),
                                    "tenure": tenure,
                                    "interest_rate": rate,
                                    "source_url": RATES_API_URL,
                                    "scraped_at": scrape_at,
                                }
                            )

    if not rates:
        raise ParseError("hnb: no loan rates parsed from api response (schema likely changed)")
    return rates

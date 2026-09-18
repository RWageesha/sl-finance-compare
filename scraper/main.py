"""One-shot pipeline: fetch each configured bank's rates page, parse it,
normalize the result onto the shared products/product_rates schema, and
persist it to Postgres. Port of cmd/scraper/main.go.
"""

from __future__ import annotations

import datetime as dt
import logging
import os
import sys
from dataclasses import dataclass
from typing import Callable

from dotenv import load_dotenv

import normalize
import validate
from banks import amana, boc, combank, dfcc, hnb, lbfinance, lolc, ndb, nsb, panasia, peoples, sampath
from db import DB, ProductRate, ScrapeRun

logging.basicConfig(level=logging.INFO, format="%(message)s")
log = logging.getLogger("scraper")


def _normalize_all(rows: list[dict], normalize_fn: Callable[[DB, int, dict], ProductRate], db: DB, bank_id: int, label: str) -> list[ProductRate]:
    """Runs normalize_fn over every raw scraped row, validates each
    result, and returns only the rows that passed both steps — logging a
    warning for anything dropped, same "skip and log" posture the
    scrapers themselves already use for malformed rows.
    """
    out = []
    for row in rows:
        try:
            pr = normalize_fn(db, bank_id, row)
        except Exception as exc:  # noqa: BLE001 - mirrors Go's "log and skip"
            log.warning("%s: normalize: %s", label, exc)
            continue
        try:
            validate.rate(pr)
        except validate.ValidationError as exc:
            log.warning("%s: %s", label, exc)
            continue
        out.append(pr)
    return out


def _run_scrape(db: DB, bank_id: int, label: str, source_url: str, fn: Callable[[], int]) -> Exception | None:
    """Wraps one scrape attempt with data_sources/scrape_runs bookkeeping:
    resolves the source row (creating it on first run), then records how
    the attempt went (row count, status, error) once fn returns. fn does
    the actual fetch/parse/normalize/insert work and returns how many rows
    it inserted (raising on total failure).
    """
    source_id = db.get_or_create_data_source(bank_id, label, source_url)

    started = dt.datetime.now(dt.timezone.utc)
    count = 0
    err: Exception | None = None
    try:
        count = fn()
    except Exception as exc:  # noqa: BLE001 - recorded below, then re-raised to the caller
        err = exc
    completed = dt.datetime.now(dt.timezone.utc)

    status = "success"
    error_message = ""
    if err is not None:
        status = "partial" if count > 0 else "failed"
        error_message = str(err)

    try:
        db.record_scrape_run(
            ScrapeRun(
                source_id=source_id,
                started_at=started,
                completed_at=completed,
                status=status,
                records_found=count,
                error_message=error_message,
            )
        )
    except Exception as exc:  # noqa: BLE001
        log.warning("%s: warning: failed to record scrape run: %s", label, exc)

    return err


def _scrape_hnb(db: DB) -> tuple[int, list[Exception]]:
    """Fetches HNB's rates API once and parses the same raw response three
    ways (fixed deposits, savings accounts, loans) — HNB's API returns all
    of a bank's published rates in a single response, so fetching it
    separately per product type would just mean redundant HTTP calls
    against the same data. Each product type's parse/insert failure is
    independent: one being fatal doesn't block the others.

    Returns (sources_attempted, errors) — see run()'s docstring for why
    the count matters as much as the errors themselves.
    """
    log.info("hnb: fetching rates")
    resp = hnb.fetch_rates_json()

    bank_id = db.get_or_create_bank(hnb.BANK_NAME, hnb.BANK_CODE)
    errors: list[Exception] = []
    attempted = 0

    def fd() -> int:
        rows = hnb.parse_fixed_deposits(resp)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(hnb.BANK_NAME), db, bank_id, "hnb")
        db.insert_product_rates(product_rates)
        log.info("hnb: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = hnb.parse_savings_accounts(resp)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "hnb")
        db.insert_product_rates(product_rates)
        log.info("hnb: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    def loans() -> int:
        rows = hnb.parse_loans(resp)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "hnb")
        db.insert_product_rates(product_rates)
        log.info("hnb: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    for label, source_url, fn in (
        ("hnb-fixed-deposits", hnb.RATES_API_URL, fd),
        ("hnb-savings", hnb.RATES_API_URL, savings),
        ("hnb-loans", hnb.RATES_API_URL, loans),
    ):
        attempted += 1
        err = _run_scrape(db, bank_id, label, source_url, fn)
        if err is not None:
            log.warning("hnb: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_combank(db: DB) -> tuple[int, list[Exception]]:
    """ComBank's Fixed Deposit rates live on their own dedicated page;
    Savings and Loans share a separate rates-tariff hub page — two fetches
    total, three parses. Returns (sources_attempted, errors).
    """
    errors: list[Exception] = []
    attempted = 0
    bank_id = db.get_or_create_bank(combank.BANK_NAME, combank.BANK_CODE)

    log.info("combank: fetching fixed deposit rates")

    def fd() -> int:
        html = combank.fetch_page()
        rows = combank.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(combank.BANK_NAME), db, bank_id, "combank")
        db.insert_product_rates(product_rates)
        log.info("combank: inserted %d fixed deposit rate(s) for %s", len(product_rates), combank.BANK_NAME)
        return len(product_rates)

    attempted += 1
    err = _run_scrape(db, bank_id, "combank-fixed-deposits", combank.RATES_URL, fd)
    if err is not None:
        log.warning("combank: fixed-deposits: %s", err)
        errors.append(err)

    log.info("combank: fetching rates-tariff page")
    tariff_html = combank.fetch_rates_tariff_page()

    def savings() -> int:
        rows = combank.parse_savings(tariff_html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "combank")
        db.insert_product_rates(product_rates)
        log.info("combank: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    def loans() -> int:
        rows = combank.parse_loans(tariff_html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "combank")
        db.insert_product_rates(product_rates)
        log.info("combank: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    def credit_cards() -> int:
        rows = combank.parse_credit_cards(tariff_html)
        product_rates = _normalize_all(rows, normalize.card, db, bank_id, "combank")
        db.insert_product_rates(product_rates)
        log.info("combank: inserted %d credit card rate(s)", len(product_rates))
        return len(product_rates)

    def debit_cards() -> int:
        rows = combank.parse_debit_cards(tariff_html)
        product_rates = _normalize_all(rows, normalize.card, db, bank_id, "combank")
        db.insert_product_rates(product_rates)
        log.info("combank: inserted %d debit card rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (
        ("combank-savings", savings),
        ("combank-loans", loans),
        ("combank-credit-cards", credit_cards),
        ("combank-debit-cards", debit_cards),
    ):
        attempted += 1
        err = _run_scrape(db, bank_id, label, combank.RATES_TARIFF_URL, fn)
        if err is not None:
            log.warning("combank: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_boc(db: DB) -> tuple[int, list[Exception]]:
    """Fetches BOC's rates & tariff page once and parses it three ways
    (fixed deposits, savings, loans) — same fetch-once-parse-many
    structure as HNB, since BOC publishes all three product types on the
    one page. Returns (sources_attempted, errors).
    """
    log.info("boc: fetching rates-tariff page")
    html = boc.fetch_page()

    bank_id = db.get_or_create_bank(boc.BANK_NAME, boc.BANK_CODE)
    errors: list[Exception] = []
    attempted = 0

    def fd() -> int:
        rows = boc.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(boc.BANK_NAME), db, bank_id, "boc")
        db.insert_product_rates(product_rates)
        log.info("boc: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = boc.parse_savings(html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "boc")
        db.insert_product_rates(product_rates)
        log.info("boc: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    def loans() -> int:
        rows = boc.parse_loans(html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "boc")
        db.insert_product_rates(product_rates)
        log.info("boc: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (("boc-fixed-deposits", fd), ("boc-savings", savings), ("boc-loans", loans)):
        attempted += 1
        err = _run_scrape(db, bank_id, label, boc.RATES_URL, fn)
        if err is not None:
            log.warning("boc: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_ndb(db: DB) -> tuple[int, list[Exception]]:
    """NDB publishes Fixed Deposit and Savings rates together on one
    deposits page, and Loan rates on a separate advances page — two
    fetches, three parses, same shape as combank. Returns
    (sources_attempted, errors).
    """
    errors: list[Exception] = []
    attempted = 0
    bank_id = db.get_or_create_bank(ndb.BANK_NAME, ndb.BANK_CODE)

    log.info("ndb: fetching deposit rates")
    deposits_html = ndb.fetch_deposits_page()

    def fd() -> int:
        rows = ndb.parse_fixed_deposits(deposits_html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(ndb.BANK_NAME), db, bank_id, "ndb")
        db.insert_product_rates(product_rates)
        log.info("ndb: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = ndb.parse_savings(deposits_html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "ndb")
        db.insert_product_rates(product_rates)
        log.info("ndb: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (("ndb-fixed-deposits", fd), ("ndb-savings", savings)):
        attempted += 1
        err = _run_scrape(db, bank_id, label, ndb.DEPOSITS_URL, fn)
        if err is not None:
            log.warning("ndb: %s: %s", label, err)
            errors.append(err)

    log.info("ndb: fetching lending rates")

    def loans() -> int:
        html = ndb.fetch_advances_page()
        rows = ndb.parse_loans(html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "ndb")
        db.insert_product_rates(product_rates)
        log.info("ndb: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    attempted += 1
    err = _run_scrape(db, bank_id, "ndb-loans", ndb.ADVANCES_URL, loans)
    if err is not None:
        log.warning("ndb: loans: %s", err)
        errors.append(err)

    return attempted, errors


def _scrape_nsb(db: DB) -> tuple[int, list[Exception]]:
    """NSB publishes Fixed Deposit and Savings rates together on one
    deposits page, and Loan rates on a separate lending page — two
    fetches, three parses, same shape as combank/ndb. Returns
    (sources_attempted, errors).
    """
    errors: list[Exception] = []
    attempted = 0
    bank_id = db.get_or_create_bank(nsb.BANK_NAME, nsb.BANK_CODE)

    log.info("nsb: fetching deposit rates")
    deposits_html = nsb.fetch_deposits_page()

    def fd() -> int:
        rows = nsb.parse_fixed_deposits(deposits_html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(nsb.BANK_NAME), db, bank_id, "nsb")
        db.insert_product_rates(product_rates)
        log.info("nsb: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = nsb.parse_savings(deposits_html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "nsb")
        db.insert_product_rates(product_rates)
        log.info("nsb: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (("nsb-fixed-deposits", fd), ("nsb-savings", savings)):
        attempted += 1
        err = _run_scrape(db, bank_id, label, nsb.DEPOSITS_URL, fn)
        if err is not None:
            log.warning("nsb: %s: %s", label, err)
            errors.append(err)

    log.info("nsb: fetching lending rates")

    def loans() -> int:
        html = nsb.fetch_lending_page()
        rows = nsb.parse_loans(html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "nsb")
        db.insert_product_rates(product_rates)
        log.info("nsb: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    attempted += 1
    err = _run_scrape(db, bank_id, "nsb-loans", nsb.LENDING_URL, loans)
    if err is not None:
        log.warning("nsb: loans: %s", err)
        errors.append(err)

    return attempted, errors


def _scrape_peoples(db: DB) -> tuple[int, list[Exception]]:
    """People's Bank publishes Fixed Deposit, Savings, and Loan rates all
    on one interest-rates page — one fetch, three parses, same shape as
    HNB/BOC. Returns (sources_attempted, errors).
    """
    log.info("peoples: fetching rates page")
    html = peoples.fetch_rates_page()

    bank_id = db.get_or_create_bank(peoples.BANK_NAME, peoples.BANK_CODE)
    errors: list[Exception] = []
    attempted = 0

    def fd() -> int:
        rows = peoples.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(peoples.BANK_NAME), db, bank_id, "peoples")
        db.insert_product_rates(product_rates)
        log.info("peoples: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = peoples.parse_savings(html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "peoples")
        db.insert_product_rates(product_rates)
        log.info("peoples: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    def loans() -> int:
        rows = peoples.parse_loans(html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "peoples")
        db.insert_product_rates(product_rates)
        log.info("peoples: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (("peoples-fixed-deposits", fd), ("peoples-savings", savings), ("peoples-loans", loans)):
        attempted += 1
        err = _run_scrape(db, bank_id, label, peoples.RATES_URL, fn)
        if err is not None:
            log.warning("peoples: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_sampath(db: DB) -> tuple[int, list[Exception]]:
    """Sampath publishes FD and Savings on separate pages (both fetched
    with a real headless browser — see sampath.py's module docstring for
    why) and Loans in a standalone PDF. Three fetches, three parses.
    Returns (sources_attempted, errors).
    """
    errors: list[Exception] = []
    attempted = 0
    bank_id = db.get_or_create_bank(sampath.BANK_NAME, sampath.BANK_CODE)

    def fd() -> int:
        html = sampath.fetch_fd_page()
        rows = sampath.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(sampath.BANK_NAME), db, bank_id, "sampath")
        db.insert_product_rates(product_rates)
        log.info("sampath: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        html = sampath.fetch_savings_page()
        rows = sampath.parse_savings(html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "sampath")
        db.insert_product_rates(product_rates)
        log.info("sampath: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    def loans() -> int:
        pdf_bytes = sampath.fetch_loans_pdf()
        rows = sampath.parse_loans(pdf_bytes)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "sampath")
        db.insert_product_rates(product_rates)
        log.info("sampath: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    for label, source_url, fn in (
        ("sampath-fixed-deposits", sampath.FD_URL, fd),
        ("sampath-savings", sampath.SAVINGS_URL, savings),
        ("sampath-loans", sampath.LOANS_PDF_URL, loans),
    ):
        attempted += 1
        log.info("sampath: fetching %s", label)
        err = _run_scrape(db, bank_id, label, source_url, fn)
        if err is not None:
            log.warning("sampath: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_panasia(db: DB) -> tuple[int, list[Exception]]:
    """Pan Asia only has Fixed Deposits scraped so far (see panasia.py's
    module docstring — its WAF rate-limits/blocks aggressively enough
    that savings/loan page research hasn't been completed yet). Returns
    (sources_attempted, errors).
    """
    bank_id = db.get_or_create_bank(panasia.BANK_NAME, panasia.BANK_CODE)

    def fd() -> int:
        html = panasia.fetch_fd_page()
        rows = panasia.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(panasia.BANK_NAME), db, bank_id, "panasia")
        db.insert_product_rates(product_rates)
        log.info("panasia: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    log.info("panasia: fetching fixed deposit rates")
    err = _run_scrape(db, bank_id, "panasia-fixed-deposits", panasia.FD_URL, fd)
    if err is not None:
        log.warning("panasia: fixed-deposits: %s", err)
        return 1, [err]
    return 1, []


def _scrape_amana(db: DB) -> tuple[int, list[Exception]]:
    """Amana publishes Fixed Deposit and Savings rates together on one
    profit-rates page — one fetch, two parses. No loan rates (see
    amana.py's module docstring). Returns (sources_attempted, errors).
    """
    log.info("amana: fetching profit rates page")
    html = amana.fetch_rates_page()

    bank_id = db.get_or_create_bank(amana.BANK_NAME, amana.BANK_CODE)
    errors: list[Exception] = []
    attempted = 0

    def fd() -> int:
        rows = amana.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(amana.BANK_NAME), db, bank_id, "amana")
        db.insert_product_rates(product_rates)
        log.info("amana: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    def savings() -> int:
        rows = amana.parse_savings(html)
        product_rates = _normalize_all(rows, normalize.savings, db, bank_id, "amana")
        db.insert_product_rates(product_rates)
        log.info("amana: inserted %d savings rate(s)", len(product_rates))
        return len(product_rates)

    for label, fn in (("amana-fixed-deposits", fd), ("amana-savings", savings)):
        attempted += 1
        err = _run_scrape(db, bank_id, label, amana.RATES_URL, fn)
        if err is not None:
            log.warning("amana: %s: %s", label, err)
            errors.append(err)

    return attempted, errors


def _scrape_lolc(db: DB) -> tuple[int, list[Exception]]:
    """LOLC Finance: Fixed Deposits only (see lolc.py's module docstring
    for why savings/loans aren't scraped). Returns (sources_attempted,
    errors).
    """
    bank_id = db.get_or_create_bank(lolc.BANK_NAME, lolc.BANK_CODE)

    def fd() -> int:
        html = lolc.fetch_fd_page()
        rows = lolc.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(lolc.BANK_NAME), db, bank_id, "lolc")
        db.insert_product_rates(product_rates)
        log.info("lolc: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    log.info("lolc: fetching fixed deposit rates")
    err = _run_scrape(db, bank_id, "lolc-fixed-deposits", lolc.FD_URL, fd)
    if err is not None:
        log.warning("lolc: fixed-deposits: %s", err)
        return 1, [err]
    return 1, []


def _scrape_lbfinance(db: DB) -> tuple[int, list[Exception]]:
    """LB Finance: Fixed Deposits only (see lbfinance.py's module
    docstring for why savings/loans aren't scraped). Returns
    (sources_attempted, errors).
    """
    bank_id = db.get_or_create_bank(lbfinance.BANK_NAME, lbfinance.BANK_CODE)

    def fd() -> int:
        html = lbfinance.fetch_fd_page()
        rows = lbfinance.parse_fixed_deposits(html)
        product_rates = _normalize_all(rows, normalize.fixed_deposit_for(lbfinance.BANK_NAME), db, bank_id, "lbfinance")
        db.insert_product_rates(product_rates)
        log.info("lbfinance: inserted %d fixed deposit rate(s)", len(product_rates))
        return len(product_rates)

    log.info("lbfinance: fetching fixed deposit rates")
    err = _run_scrape(db, bank_id, "lbfinance-fixed-deposits", lbfinance.FD_URL, fd)
    if err is not None:
        log.warning("lbfinance: fixed-deposits: %s", err)
        return 1, [err]
    return 1, []


def _scrape_dfcc(db: DB) -> tuple[int, list[Exception]]:
    """DFCC: Loan rates only (see dfcc.py's module docstring for why FD/
    savings aren't scraped). Returns (sources_attempted, errors).
    """
    bank_id = db.get_or_create_bank(dfcc.BANK_NAME, dfcc.BANK_CODE)

    def loans() -> int:
        html = dfcc.fetch_rates_page()
        rows = dfcc.parse_loans(html)
        product_rates = _normalize_all(rows, normalize.loan, db, bank_id, "dfcc")
        db.insert_product_rates(product_rates)
        log.info("dfcc: inserted %d loan rate(s)", len(product_rates))
        return len(product_rates)

    log.info("dfcc: fetching rates & tariff page")
    err = _run_scrape(db, bank_id, "dfcc-loans", dfcc.RATES_URL, loans)
    if err is not None:
        log.warning("dfcc: loans: %s", err)
        return 1, [err]
    return 1, []


def run() -> tuple[int, list[Exception]]:
    """Returns (sources_attempted, errors) rather than just errors — a
    source here is one scrape target within a bank (e.g. "combank-loans"),
    tallied by each _scrape_* function above. Tracking the attempted count
    alongside errors is what lets main() tell "one bank's site is
    temporarily blocking us" (a handful of the ~12 sources failed, most
    succeeded — real data still landed) apart from "nothing worked at
    all" (e.g. the database itself is unreachable) — those deserve very
    different exit codes, even though both currently just produce a
    non-empty errors list.
    """
    load_dotenv()

    # Render's environment-variable editor can retain a trailing newline when
    # a connection URL is pasted. Whitespace is never meaningful in a
    # PostgreSQL URL, and without trimming it Postgres treats e.g.
    # "postgres\\n" as a different database name.
    conn_string = os.environ.get("DATABASE_URL", "").strip()
    if not conn_string:
        raise RuntimeError("DATABASE_URL is not set (check your .env file)")

    total_attempted = 0
    errors: list[Exception] = []
    with DB(conn_string) as db:
        # One bank's site being down or having changed its markup
        # shouldn't block scraping the others, so run every scraper and
        # only report failure at the end (still surfacing every individual
        # error via log).
        for scrape_bank in (
            _scrape_hnb,
            _scrape_combank,
            _scrape_boc,
            _scrape_ndb,
            _scrape_nsb,
            _scrape_peoples,
            _scrape_sampath,
            _scrape_panasia,
            _scrape_amana,
            _scrape_lolc,
            _scrape_lbfinance,
            _scrape_dfcc,
        ):
            try:
                attempted, bank_errors = scrape_bank(db)
                total_attempted += attempted
                errors.extend(bank_errors)
            except Exception as exc:  # noqa: BLE001 - a bank-level fetch failure (e.g. network down)
                log.warning("%s: %s", scrape_bank.__name__, exc)
                errors.append(exc)
                # The bank's own function never got far enough to report
                # how many of its sources it would have attempted (e.g.
                # its very first fetch failed) — count it as one failed
                # attempt so it still counts toward "did anything fail"
                # without guessing at a number it never produced.
                total_attempted += 1

    return total_attempted, errors


def main() -> None:
    total_attempted, errors = run()
    if not errors:
        return

    # GitHub Actions (and most other CI UIs) render "::warning::" lines as
    # annotations shown directly on the run summary page, so a partial
    # failure is visible at a glance without opening the log.
    for err in errors:
        print(f"::warning::{err}")

    succeeded = total_attempted - len(errors)
    if succeeded > 0:
        # Real, useful work happened — a bank's site being temporarily
        # down or blocking this specific runner shouldn't paint the whole
        # run red when most of it worked. The warnings above still make
        # exactly what failed visible, so this isn't hiding anything.
        log.warning("partial success: %d/%d source(s) succeeded, %d failed", succeeded, total_attempted, len(errors))
        return

    log.error("total failure: 0/%d source(s) succeeded", total_attempted)
    sys.exit(1)


if __name__ == "__main__":
    main()

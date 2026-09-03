"""Postgres access for the scraper side of the pipeline. Port of the
scraper-facing half of internal/db/db.go — GetLatestRates and the other
API-facing reads stay Go-only (cmd/api), since this module is only used by
scraper/main.py.
"""

from __future__ import annotations

import datetime as dt
from dataclasses import dataclass, field

import psycopg


@dataclass
class ProductRate:
    """Mirrors models.ProductRate (internal/models/models.go) — the shape
    normalize.py produces and insert_product_rates consumes.
    """

    product_id: int
    interest_rate: float
    tenure_value: int | None = None
    tenure_unit: str | None = None
    tenure_label: str = ""
    rate_label: str = ""
    min_amount: float | None = None
    source_url: str = ""
    confidence: str = "high"
    scraped_at: dt.datetime = field(default_factory=lambda: dt.datetime.now(dt.timezone.utc))


@dataclass
class ScrapeRun:
    source_id: int
    started_at: dt.datetime
    completed_at: dt.datetime
    status: str
    records_found: int
    error_message: str = ""


class DB:
    def __init__(self, conn_string: str):
        if not conn_string:
            raise ValueError("db: connection string is empty")
        self._conn = psycopg.connect(conn_string, autocommit=True)

    def close(self) -> None:
        self._conn.close()

    def __enter__(self) -> "DB":
        return self

    def __exit__(self, *exc) -> None:
        self.close()

    def get_or_create_bank(self, name: str, code: str) -> int:
        """Looks up a bank by its code, inserting it if it doesn't already
        exist. Lets the scraper run against a fresh database without
        depending on the seed migration having been applied for this bank.
        """
        with self._conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO banks (name, code, slug)
                VALUES (%s, %s, lower(%s))
                ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
                RETURNING id
                """,
                (name, code, code),
            )
            return cur.fetchone()[0]

    def get_or_create_category(self, code: str, name: str, parent_code: str) -> int:
        """Looks up a product category by code, inserting it (under
        parent_code, which must already exist) if it doesn't already exist.
        Every code the normalizers use is pre-seeded by migration 004 with
        a curated display name — the ON CONFLICT branch intentionally
        leaves that name alone (a no-op self-update, just to keep
        RETURNING id working) rather than overwriting it with whatever name
        a caller happens to pass for an already-existing code.
        """
        with self._conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO product_categories (code, name, parent_id)
                VALUES (%s, %s, (SELECT id FROM product_categories WHERE code = %s))
                ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
                RETURNING id
                """,
                (code, name, parent_code),
            )
            return cur.fetchone()[0]

    def get_or_create_product(self, bank_id: int, category_id: int, name: str) -> int:
        """Looks up a bank's product by (bank, category, name), inserting
        it if it doesn't already exist.
        """
        with self._conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO products (bank_id, category_id, name)
                VALUES (%s, %s, %s)
                ON CONFLICT (bank_id, category_id, name) DO UPDATE SET name = EXCLUDED.name
                RETURNING id
                """,
                (bank_id, category_id, name),
            )
            return cur.fetchone()[0]

    def insert_product_rates(self, rates: list[ProductRate]) -> None:
        """Bulk-inserts a batch of normalized rates. If rates is empty,
        this is a no-op. Every scrape adds new rows rather than
        overwriting, so product_rates is itself the rate history.
        """
        if not rates:
            return
        with self._conn.cursor() as cur:
            cur.executemany(
                """
                INSERT INTO product_rates
                    (product_id, tenure_value, tenure_unit, tenure_label, rate_label,
                     min_amount, interest_rate, source_url, confidence, scraped_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """,
                [
                    (
                        r.product_id,
                        r.tenure_value,
                        r.tenure_unit or None,
                        r.tenure_label,
                        r.rate_label,
                        r.min_amount,
                        r.interest_rate,
                        r.source_url,
                        r.confidence or "high",
                        r.scraped_at,
                    )
                    for r in rates
                ],
            )

    def get_or_create_data_source(self, bank_id: int, label: str, source_url: str) -> int:
        """Looks up a (bank, label) data source, inserting it if it doesn't
        already exist. label is a short scraper-chosen identifier, e.g.
        "hnb-fixed-deposits".
        """
        with self._conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO data_sources (bank_id, label, source_url)
                VALUES (%s, %s, %s)
                ON CONFLICT (bank_id, label) DO UPDATE SET source_url = EXCLUDED.source_url
                RETURNING id
                """,
                (bank_id, label, source_url),
            )
            return cur.fetchone()[0]

    def record_scrape_run(self, run: ScrapeRun) -> None:
        """Inserts one scrape_runs row for a completed attempt."""
        with self._conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO scrape_runs (source_id, started_at, completed_at, status, records_found, error_message)
                VALUES (%s, %s, %s, %s, %s, %s)
                """,
                (
                    run.source_id,
                    run.started_at,
                    run.completed_at,
                    run.status,
                    run.records_found,
                    run.error_message or None,
                ),
            )

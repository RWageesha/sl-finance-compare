package db

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wageesha/sl-finance-compare/internal/models"
)

type DB struct {
	pool *pgxpool.Pool
}

// New creates a connection pool and verifies connectivity with a ping.
func New(ctx context.Context, connString string) (*DB, error) {
	if connString == "" {
		return nil, fmt.Errorf("db: connection string is empty")
	}

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

// GetOrCreateBank looks up a bank by its code, inserting it if it doesn't
// already exist. This lets scrapers run against a fresh database without
// depending on the seed migration having been applied for their bank.
func (d *DB) GetOrCreateBank(ctx context.Context, name, code string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO banks (name, code, slug)
		VALUES ($1, $2, lower($2))
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name, code).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: get or create bank %q: %w", code, err)
	}
	return id, nil
}

// GetOrCreateCategory looks up a product category by code, inserting it
// (under parentCode, which must already exist) if it doesn't already
// exist. Every code the current normalizers use is pre-seeded by
// migration 004 with a curated display name — the ON CONFLICT branch
// intentionally leaves that name alone (a no-op self-update, just to keep
// RETURNING id working) rather than overwriting it with whatever name a
// caller happens to pass for an already-existing code.
func (d *DB) GetOrCreateCategory(ctx context.Context, code, name, parentCode string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO product_categories (code, name, parent_id)
		VALUES ($1, $2, (SELECT id FROM product_categories WHERE code = $3))
		ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
		RETURNING id
	`, code, name, parentCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: get or create category %q: %w", code, err)
	}
	return id, nil
}

// GetOrCreateProduct looks up a bank's product by (bank, category, name),
// inserting it if it doesn't already exist.
func (d *DB) GetOrCreateProduct(ctx context.Context, bankID, categoryID int64, name string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO products (bank_id, category_id, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (bank_id, category_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, bankID, categoryID, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: get or create product %q: %w", name, err)
	}
	return id, nil
}

// InsertProductRates bulk-inserts a batch of normalized rates in a single
// transaction. If rates is empty, it is a no-op.
func (d *DB) InsertProductRates(ctx context.Context, rates []models.ProductRate) error {
	if len(rates) == 0 {
		return nil
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	batch := &pgx.Batch{}
	for _, r := range rates {
		confidence := r.Confidence
		if confidence == "" {
			confidence = "high"
		}
		batch.Queue(`
			INSERT INTO product_rates
				(product_id, tenure_value, tenure_unit, tenure_label, rate_label,
				 min_amount, interest_rate, source_url, confidence, scraped_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, r.ProductID, r.TenureValue, nullIfEmpty(r.TenureUnit), r.TenureLabel, r.RateLabel,
			r.MinAmount, r.InterestRate, r.SourceURL, confidence, r.ScrapedAt)
	}

	br := tx.SendBatch(ctx, batch)
	for range rates {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("db: insert product rate: %w", err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("db: close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit tx: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetLatestRates returns the most recent rate per
// (product, tenure_value, tenure_label, rate_label), joined with bank,
// product, and category details, for every category under categoryGroup
// (a top-level code like "FIXED_DEPOSIT", "SAVINGS", or "LOAN" — matches
// both that category itself and any of its children). Ordered by bank
// name then product name.
func (d *DB) GetLatestRates(ctx context.Context, categoryGroup string) ([]models.ProductRate, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT DISTINCT ON (r.product_id, r.tenure_value, r.tenure_label, r.rate_label)
			r.id, r.product_id, b.name, b.code, c.code, p.name,
			r.tenure_value, r.tenure_unit, r.tenure_label, r.rate_label,
			r.min_amount, r.interest_rate, r.source_url, r.confidence, r.scraped_at
		FROM product_rates r
		JOIN products p ON p.id = r.product_id
		JOIN banks b ON b.id = p.bank_id
		JOIN product_categories c ON c.id = p.category_id
		WHERE c.code = $1 OR c.parent_id = (SELECT id FROM product_categories WHERE code = $1)
		ORDER BY r.product_id, r.tenure_value, r.tenure_label, r.rate_label, r.scraped_at DESC
	`, categoryGroup)
	if err != nil {
		return nil, fmt.Errorf("db: query latest rates for %q: %w", categoryGroup, err)
	}
	defer rows.Close()

	var out []models.ProductRate
	for rows.Next() {
		var r models.ProductRate
		var tenureUnit *string
		if err := rows.Scan(
			&r.ID, &r.ProductID, &r.BankName, &r.BankCode, &r.CategoryCode, &r.ProductName,
			&r.TenureValue, &tenureUnit, &r.TenureLabel, &r.RateLabel,
			&r.MinAmount, &r.InterestRate, &r.SourceURL, &r.Confidence, &r.ScrapedAt,
		); err != nil {
			return nil, fmt.Errorf("db: scan product rate row: %w", err)
		}
		if tenureUnit != nil {
			r.TenureUnit = *tenureUnit
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate product rate rows: %w", err)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].BankName != out[j].BankName {
			return out[i].BankName < out[j].BankName
		}
		return out[i].ProductName < out[j].ProductName
	})

	return out, nil
}

// GetOrCreateDataSource looks up a (bank, label) data source, inserting it
// if it doesn't already exist. label is a short scraper-chosen identifier,
// e.g. "hnb-fixed-deposits".
func (d *DB) GetOrCreateDataSource(ctx context.Context, bankID int64, label, sourceURL string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO data_sources (bank_id, label, source_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (bank_id, label) DO UPDATE SET source_url = EXCLUDED.source_url
		RETURNING id
	`, bankID, label, sourceURL).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: get or create data source %q: %w", label, err)
	}
	return id, nil
}

// RecordScrapeRun inserts one scrape_runs row for a completed attempt.
func (d *DB) RecordScrapeRun(ctx context.Context, run models.ScrapeRun) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO scrape_runs (source_id, started_at, completed_at, status, records_found, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, run.SourceID, run.StartedAt, run.CompletedAt, run.Status, run.RecordsFound, nullIfEmpty(run.ErrorMessage))
	if err != nil {
		return fmt.Errorf("db: record scrape run: %w", err)
	}
	return nil
}

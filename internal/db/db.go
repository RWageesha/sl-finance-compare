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
		INSERT INTO banks (name, code)
		VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name, code).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: get or create bank %q: %w", code, err)
	}
	return id, nil
}

// InsertFixedDepositRates bulk-inserts a batch of scraped rates in a single
// transaction. If rates is empty, it is a no-op (no transaction is opened).
func (d *DB) InsertFixedDepositRates(ctx context.Context, rates []models.FixedDepositRate) error {
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
		batch.Queue(`
			INSERT INTO fixed_deposit_rates
				(bank_id, tenure_months, min_amount, interest_rate, rate_type, source_url, scraped_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, r.BankID, r.TenureMonths, r.MinAmount, r.InterestRate, r.RateType, r.SourceURL, r.ScrapedAt)
	}

	br := tx.SendBatch(ctx, batch)
	for range rates {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("db: insert rate: %w", err)
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

// GetLatestFixedDeposits returns the most recent rate per
// (bank, tenure_months, rate_type), joined with bank details, ordered by
// bank name then tenure.
func (d *DB) GetLatestFixedDeposits(ctx context.Context) ([]models.FixedDepositRate, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT DISTINCT ON (r.bank_id, r.tenure_months, r.rate_type)
			r.id, r.bank_id, b.name, b.code, r.tenure_months, r.min_amount,
			r.interest_rate, r.rate_type, r.source_url, r.scraped_at
		FROM fixed_deposit_rates r
		JOIN banks b ON b.id = r.bank_id
		ORDER BY r.bank_id, r.tenure_months, r.rate_type, r.scraped_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("db: query latest rates: %w", err)
	}
	defer rows.Close()

	var out []models.FixedDepositRate
	for rows.Next() {
		var r models.FixedDepositRate
		if err := rows.Scan(
			&r.ID, &r.BankID, &r.BankName, &r.BankCode, &r.TenureMonths, &r.MinAmount,
			&r.InterestRate, &r.RateType, &r.SourceURL, &r.ScrapedAt,
		); err != nil {
			return nil, fmt.Errorf("db: scan rate row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate rate rows: %w", err)
	}

	// Sort by bank name then tenure for stable, human-friendly API output
	// (the DISTINCT ON query above must order by bank_id, not bank name,
	// since that's what the dedup key requires).
	sort.Slice(out, func(i, j int) bool {
		if out[i].BankName != out[j].BankName {
			return out[i].BankName < out[j].BankName
		}
		return out[i].TenureMonths < out[j].TenureMonths
	})

	return out, nil
}

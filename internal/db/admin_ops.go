package db

import (
	"context"
	"fmt"
	"time"
)

// ===== Dashboard =====

type DashboardStats struct {
	Banks          int `json:"banks"`
	Products       int `json:"products"`
	ActiveRates    int `json:"active_rates"`
	PendingReview  int `json:"pending_review"`
	OutdatedData   int `json:"outdated_data"`
	FailedSources  int `json:"failed_sources"`
}

// GetDashboardStats computes the 6 stat-card counts against real tables.
// "Outdated" means a source's latest rate for any of its products hasn't
// been refreshed in over 48 hours — the same threshold used for the
// "rates not updated" system alert, so the two stay consistent.
func (d *DB) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	var s DashboardStats
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM banks WHERE status = 'active'`).Scan(&s.Banks); err != nil {
		return s, fmt.Errorf("db: count banks: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM products`).Scan(&s.Products); err != nil {
		return s, fmt.Errorf("db: count products: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM product_rates`).Scan(&s.ActiveRates); err != nil {
		return s, fmt.Errorf("db: count rates: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM product_rates WHERE verification_status = 'pending'`).Scan(&s.PendingReview); err != nil {
		return s, fmt.Errorf("db: count pending: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `
		SELECT count(DISTINCT p.bank_id) FROM product_rates r
		JOIN products p ON p.id = r.product_id
		WHERE r.scraped_at < now() - interval '48 hours'
		AND p.bank_id NOT IN (
			SELECT p2.bank_id FROM product_rates r2 JOIN products p2 ON p2.id = r2.product_id
			WHERE r2.scraped_at >= now() - interval '48 hours'
		)
	`).Scan(&s.OutdatedData); err != nil {
		return s, fmt.Errorf("db: count outdated: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT DISTINCT ON (source_id) source_id, status FROM scrape_runs ORDER BY source_id, started_at DESC
		) latest WHERE status = 'failed'
	`).Scan(&s.FailedSources); err != nil {
		return s, fmt.Errorf("db: count failed sources: %w", err)
	}
	return s, nil
}

type ScrapingStatusRow struct {
	BankName      string     `json:"bank_name"`
	LastRun       *time.Time `json:"last_run"`
	Status        string     `json:"status"`
	RecordsFound  int        `json:"records_found"`
}

// GetScrapingStatus returns one row per bank: its most recently *started*
// scrape run across all of that bank's sources.
func (d *DB) GetScrapingStatus(ctx context.Context) ([]ScrapingStatusRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT b.name, latest.started_at, latest.status, latest.records_found
		FROM banks b
		JOIN LATERAL (
			SELECT sr.started_at, sr.status, sr.records_found
			FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id
			WHERE ds.bank_id = b.id
			ORDER BY sr.started_at DESC LIMIT 1
		) latest ON true
		WHERE b.status = 'active'
		ORDER BY latest.started_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("db: scraping status: %w", err)
	}
	defer rows.Close()
	out := []ScrapingStatusRow{}
	for rows.Next() {
		var r ScrapingStatusRow
		if err := rows.Scan(&r.BankName, &r.LastRun, &r.Status, &r.RecordsFound); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type SystemAlert struct {
	Level   string `json:"level"` // critical | warning | info
	Message string `json:"message"`
	At      time.Time `json:"at"`
}

// GetSystemAlerts derives alerts from real signals only: a source whose
// latest run failed (critical), a bank with no fresh data in 48h
// (warning), and the count of pending user reports (info). There is no
// fabricated "system health" score here.
func (d *DB) GetSystemAlerts(ctx context.Context) ([]SystemAlert, error) {
	alerts := []SystemAlert{}

	rows, err := d.pool.Query(ctx, `
		SELECT b.name, latest.started_at, latest.error_message
		FROM banks b
		JOIN LATERAL (
			SELECT sr.started_at, sr.status, sr.error_message
			FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id
			WHERE ds.bank_id = b.id ORDER BY sr.started_at DESC LIMIT 1
		) latest ON true
		WHERE latest.status = 'failed'
	`)
	if err != nil {
		return nil, fmt.Errorf("db: alerts (failed): %w", err)
	}
	for rows.Next() {
		var name string
		var at time.Time
		var errMsg *string
		if err := rows.Scan(&name, &at, &errMsg); err != nil {
			rows.Close()
			return nil, err
		}
		msg := name + " scraper failed"
		if errMsg != nil && *errMsg != "" {
			msg += " (" + *errMsg + ")"
		}
		alerts = append(alerts, SystemAlert{Level: "critical", Message: msg, At: at})
	}
	rows.Close()

	rows2, err := d.pool.Query(ctx, `
		SELECT b.name, max(r.scraped_at) FROM banks b
		JOIN products p ON p.bank_id = b.id
		JOIN product_rates r ON r.product_id = p.id
		WHERE b.status = 'active'
		GROUP BY b.name HAVING max(r.scraped_at) < now() - interval '48 hours'
	`)
	if err != nil {
		return nil, fmt.Errorf("db: alerts (stale): %w", err)
	}
	for rows2.Next() {
		var name string
		var last time.Time
		if err := rows2.Scan(&name, &last); err != nil {
			rows2.Close()
			return nil, err
		}
		alerts = append(alerts, SystemAlert{Level: "warning", Message: name + " rates not updated in 48h+", At: last})
	}
	rows2.Close()

	var pendingReports int
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM user_reports WHERE status = 'pending'`).Scan(&pendingReports); err != nil {
		return nil, fmt.Errorf("db: alerts (reports): %w", err)
	}
	if pendingReports > 0 {
		plural := "s"
		if pendingReports == 1 {
			plural = ""
		}
		alerts = append(alerts, SystemAlert{
			Level:   "info",
			Message: fmt.Sprintf("%d user report%s pending review", pendingReports, plural),
			At:      time.Now(),
		})
	}
	return alerts, nil
}

type RateChangeRow struct {
	BankName    string    `json:"bank_name"`
	ProductName string    `json:"product_name"`
	OldRate     *float64  `json:"old_rate"`
	NewRate     float64   `json:"new_rate"`
	RateID      int64     `json:"rate_id"`
	DetectedAt  time.Time `json:"detected_at"`
	Status      string    `json:"status"`
}

// GetRecentRateChanges returns the newest product_rates row per
// (product, tenure, rate_label) line alongside the value it replaced, for
// lines where a prior value actually exists (a brand-new product's first
// rate has no "old" to show and is a first-time insert, not a "change").
func (d *DB) GetRecentRateChanges(ctx context.Context, limit int) ([]RateChangeRow, error) {
	rows, err := d.pool.Query(ctx, `
		WITH ranked AS (
			SELECT r.id, r.product_id, r.tenure_value, r.tenure_label, r.rate_label,
				r.interest_rate, r.scraped_at, r.verification_status,
				ROW_NUMBER() OVER (PARTITION BY r.product_id, r.tenure_value, r.tenure_label, r.rate_label ORDER BY r.scraped_at DESC) rn
			FROM product_rates r
		)
		SELECT b.name, p.name, prev.interest_rate, cur.interest_rate, cur.id, cur.scraped_at, cur.verification_status
		FROM ranked cur
		JOIN ranked prev ON prev.product_id = cur.product_id AND prev.tenure_value IS NOT DISTINCT FROM cur.tenure_value
			AND prev.tenure_label = cur.tenure_label AND prev.rate_label = cur.rate_label AND prev.rn = 2
		JOIN products p ON p.id = cur.product_id
		JOIN banks b ON b.id = p.bank_id
		WHERE cur.rn = 1 AND prev.interest_rate IS DISTINCT FROM cur.interest_rate
		ORDER BY cur.scraped_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("db: recent rate changes: %w", err)
	}
	defer rows.Close()
	out := []RateChangeRow{}
	for rows.Next() {
		var c RateChangeRow
		if err := rows.Scan(&c.BankName, &c.ProductName, &c.OldRate, &c.NewRate, &c.RateID, &c.DetectedAt, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ===== Verification Queue =====

type VerificationRow struct {
	RateID       int64     `json:"rate_id"`
	BankName     string    `json:"bank_name"`
	ProductName  string    `json:"product_name"`
	CategoryCode string    `json:"category_code"`
	PrevRate     *float64  `json:"prev_rate"`
	NewRate      float64   `json:"new_rate"`
	SourceURL    string    `json:"source_url"`
	Confidence   string    `json:"confidence"`
	DetectedAt   time.Time `json:"detected_at"`
}

type VerificationFilter struct {
	CategoryGroup string
	BankName      string
}

func (d *DB) ListVerificationQueue(ctx context.Context, f VerificationFilter) ([]VerificationRow, error) {
	where := "WHERE r.verification_status = 'pending'"
	args := []any{}
	if f.CategoryGroup != "" {
		args = append(args, f.CategoryGroup)
		where += fmt.Sprintf(" AND (c.code = $%d OR c.parent_id = (SELECT id FROM product_categories WHERE code = $%d))", len(args), len(args))
	}
	if f.BankName != "" {
		args = append(args, f.BankName)
		where += fmt.Sprintf(" AND b.name = $%d", len(args))
	}

	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT r.id, b.name, p.name, c.code,
			(SELECT r2.interest_rate FROM product_rates r2
			 WHERE r2.product_id = r.product_id AND r2.tenure_value IS NOT DISTINCT FROM r.tenure_value
			   AND r2.tenure_label = r.tenure_label AND r2.rate_label = r.rate_label AND r2.scraped_at < r.scraped_at
			 ORDER BY r2.scraped_at DESC LIMIT 1),
			r.interest_rate, r.source_url, r.confidence, r.scraped_at
		FROM product_rates r
		JOIN products p ON p.id = r.product_id
		JOIN banks b ON b.id = p.bank_id
		JOIN product_categories c ON c.id = p.category_id
		%s
		ORDER BY r.scraped_at DESC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("db: verification queue: %w", err)
	}
	defer rows.Close()
	out := []VerificationRow{}
	for rows.Next() {
		var v VerificationRow
		var src *string
		if err := rows.Scan(&v.RateID, &v.BankName, &v.ProductName, &v.CategoryCode, &v.PrevRate, &v.NewRate, &src, &v.Confidence, &v.DetectedAt); err != nil {
			return nil, err
		}
		if src != nil {
			v.SourceURL = *src
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (d *DB) SetRateVerificationStatus(ctx context.Context, rateID int64, status string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE product_rates SET verification_status = $1 WHERE id = $2`, status, rateID)
	if err != nil {
		return fmt.Errorf("db: set rate verification status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ===== Rate Management =====

type RateManagementRow struct {
	RateID        int64     `json:"rate_id"`
	ProductID     int64     `json:"product_id"`
	BankName      string    `json:"bank_name"`
	ProductName   string    `json:"product_name"`
	TenureValue   *int      `json:"tenure_value"`
	TenureLabel   string    `json:"tenure_label"`
	RateLabel     string    `json:"rate_label"`
	CurrentRate   float64   `json:"current_rate"`
	EffectiveFrom time.Time `json:"effective_from"`
	Status        string    `json:"status"`
	LastChecked   time.Time `json:"last_checked"`
}

type RateManagementFilter struct {
	Search        string
	BankName      string
	CategoryGroup string
	Status        string
	Page, PerPage int
}

func (d *DB) ListRatesAdmin(ctx context.Context, f RateManagementFilter) ([]RateManagementRow, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += fmt.Sprintf(" AND (b.name ILIKE $%d OR p.name ILIKE $%d)", len(args), len(args))
	}
	if f.BankName != "" {
		args = append(args, f.BankName)
		where += fmt.Sprintf(" AND b.name = $%d", len(args))
	}
	if f.CategoryGroup != "" {
		args = append(args, f.CategoryGroup)
		where += fmt.Sprintf(" AND (c.code = $%d OR c.parent_id = (SELECT id FROM product_categories WHERE code = $%d))", len(args), len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND r.verification_status = $%d", len(args))
	}

	base := fmt.Sprintf(`
		FROM (
			SELECT DISTINCT ON (product_id, tenure_value, tenure_label, rate_label) *
			FROM product_rates ORDER BY product_id, tenure_value, tenure_label, rate_label, scraped_at DESC
		) r
		JOIN products p ON p.id = r.product_id
		JOIN banks b ON b.id = p.bank_id
		JOIN product_categories c ON c.id = p.category_id
		%s
	`, where)

	var total int
	if err := d.pool.QueryRow(ctx, "SELECT count(*) "+base, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("db: count rates: %w", err)
	}

	page, perPage := pagination(f.Page, f.PerPage)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.product_id, b.name, p.name, r.tenure_value, r.tenure_label, r.rate_label, r.interest_rate, r.scraped_at, r.verification_status, r.scraped_at
		%s ORDER BY b.name, p.name LIMIT $%d OFFSET $%d
	`, base, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("db: list rates: %w", err)
	}
	defer rows.Close()
	out := []RateManagementRow{}
	for rows.Next() {
		var r RateManagementRow
		if err := rows.Scan(&r.RateID, &r.ProductID, &r.BankName, &r.ProductName, &r.TenureValue, &r.TenureLabel, &r.RateLabel, &r.CurrentRate, &r.EffectiveFrom, &r.Status, &r.LastChecked); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// InsertManualRate appends a new, admin-authored product_rates row for the
// same logical line (product/tenure/rate label) as baseRateID — matching
// this project's append-only history convention rather than mutating a
// past row.
func (d *DB) InsertManualRate(ctx context.Context, baseRateID int64, newRate float64, adminID int64) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO product_rates (product_id, tenure_value, tenure_unit, tenure_label, rate_label, min_amount, interest_rate, source_url, confidence, verification_status, created_by_admin_id)
		SELECT product_id, tenure_value, tenure_unit, tenure_label, rate_label, min_amount, $2, source_url, 'manual', 'verified', $3
		FROM product_rates WHERE id = $1
		RETURNING id
	`, baseRateID, newRate, adminID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: insert manual rate: %w", err)
	}
	return id, nil
}

type RateHistoryAdminRow struct {
	RateID      int64     `json:"rate_id"`
	Rate        float64   `json:"rate"`
	ScrapedAt   time.Time `json:"effective_date"`
	ChangedBy   string    `json:"changed_by"`
	Status      string    `json:"status"`
}

func (d *DB) GetRateHistoryAdmin(ctx context.Context, productID int64, tenureValue *int, tenureLabel, rateLabel string) ([]RateHistoryAdminRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT r.id, r.interest_rate, r.scraped_at, r.verification_status, r.confidence, a.username
		FROM product_rates r
		LEFT JOIN admin_users a ON a.id = r.created_by_admin_id
		WHERE r.product_id = $1 AND r.tenure_value IS NOT DISTINCT FROM $2 AND r.tenure_label = $3 AND r.rate_label = $4
		ORDER BY r.scraped_at ASC
	`, productID, tenureValue, tenureLabel, rateLabel)
	if err != nil {
		return nil, fmt.Errorf("db: rate history admin: %w", err)
	}
	defer rows.Close()
	out := []RateHistoryAdminRow{}
	first := true
	for rows.Next() {
		var h RateHistoryAdminRow
		var confidence string
		var adminUsername *string
		if err := rows.Scan(&h.RateID, &h.Rate, &h.ScrapedAt, &h.Status, &confidence, &adminUsername); err != nil {
			return nil, err
		}
		switch {
		case adminUsername != nil:
			h.ChangedBy = "Approved by " + *adminUsername
		case first:
			h.ChangedBy = "Initial entry"
		default:
			h.ChangedBy = "Auto-detected"
		}
		first = false
		out = append(out, h)
	}
	return out, rows.Err()
}

// ===== Data Sources =====

type DataSourceRow struct {
	SourceID      int64      `json:"source_id"`
	BankName      string     `json:"bank_name"`
	Label         string     `json:"label"`
	SourceType    string     `json:"source_type"`
	SourceURL     string     `json:"source_url"`
	Status        string     `json:"status"`
	LastChecked   *time.Time `json:"last_checked"`
	LastSuccess   *time.Time `json:"last_success"`
	LastStatus    string     `json:"last_status"`
}

type DataSourceFilter struct {
	BankName   string
	SourceType string
	Status     string
}

func (d *DB) ListDataSources(ctx context.Context, f DataSourceFilter) ([]DataSourceRow, error) {
	where := "WHERE 1=1"
	args := []any{}
	if f.BankName != "" {
		args = append(args, f.BankName)
		where += fmt.Sprintf(" AND b.name = $%d", len(args))
	}
	if f.SourceType != "" {
		args = append(args, f.SourceType)
		where += fmt.Sprintf(" AND ds.source_type = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND ds.status = $%d", len(args))
	}
	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT ds.id, b.name, ds.label, ds.source_type, ds.source_url, ds.status,
			latest.started_at, success.started_at, latest.status
		FROM data_sources ds
		JOIN banks b ON b.id = ds.bank_id
		LEFT JOIN LATERAL (SELECT started_at, status FROM scrape_runs WHERE source_id = ds.id ORDER BY started_at DESC LIMIT 1) latest ON true
		LEFT JOIN LATERAL (SELECT started_at FROM scrape_runs WHERE source_id = ds.id AND status = 'success' ORDER BY started_at DESC LIMIT 1) success ON true
		%s ORDER BY b.name, ds.label
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("db: list data sources: %w", err)
	}
	defer rows.Close()
	out := []DataSourceRow{}
	for rows.Next() {
		var r DataSourceRow
		var lastStatus *string
		if err := rows.Scan(&r.SourceID, &r.BankName, &r.Label, &r.SourceType, &r.SourceURL, &r.Status, &r.LastChecked, &r.LastSuccess, &lastStatus); err != nil {
			return nil, err
		}
		if lastStatus != nil {
			r.LastStatus = *lastStatus
		} else {
			r.LastStatus = "never run"
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) SetDataSourceStatus(ctx context.Context, id int64, status string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE data_sources SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("db: set data source status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) AddDataSource(ctx context.Context, bankID int64, label, sourceURL, sourceType string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO data_sources (bank_id, label, source_url, source_type) VALUES ($1, $2, $3, $4)
		ON CONFLICT (bank_id, label) DO UPDATE SET source_url = EXCLUDED.source_url, source_type = EXCLUDED.source_type
		RETURNING id
	`, bankID, label, sourceURL, sourceType).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: add data source: %w", err)
	}
	return id, nil
}

// ===== Scraping Jobs =====

type ScrapeJobRow struct {
	RunID        int64      `json:"run_id"`
	BankName     string     `json:"bank_name"`
	SourceLabel  string     `json:"source_label"`
	SourceID     int64      `json:"source_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Found        int        `json:"found"`
	Changed      int        `json:"changed"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message"`
}

func (d *DB) ListScrapeRuns(ctx context.Context, limit int) ([]ScrapeJobRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT sr.id, b.name, ds.label, ds.id, sr.started_at, sr.completed_at, sr.records_found, sr.status, sr.error_message,
			(SELECT count(*) FROM product_rates r JOIN products p ON p.id = r.product_id
			 WHERE p.bank_id = ds.bank_id AND r.scraped_at BETWEEN sr.started_at AND coalesce(sr.completed_at, sr.started_at))
		FROM scrape_runs sr
		JOIN data_sources ds ON ds.id = sr.source_id
		JOIN banks b ON b.id = ds.bank_id
		ORDER BY sr.started_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("db: list scrape runs: %w", err)
	}
	defer rows.Close()
	out := []ScrapeJobRow{}
	for rows.Next() {
		var r ScrapeJobRow
		var errMsg *string
		if err := rows.Scan(&r.RunID, &r.BankName, &r.SourceLabel, &r.SourceID, &r.StartedAt, &r.CompletedAt, &r.Found, &r.Status, &errMsg, &r.Changed); err != nil {
			return nil, err
		}
		if errMsg != nil {
			r.ErrorMessage = *errMsg
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ScrapeJobDetail struct {
	ScrapeJobRow
	SourceURL         string `json:"source_url"`
	ConsecutiveFails  int    `json:"consecutive_fails"`
	ExpectedRecords   int    `json:"expected_records"`
}

func (d *DB) GetScrapeRunDetail(ctx context.Context, runID int64) (*ScrapeJobDetail, error) {
	var det ScrapeJobDetail
	var errMsg *string
	err := d.pool.QueryRow(ctx, `
		SELECT sr.id, b.name, ds.label, ds.id, sr.started_at, sr.completed_at, sr.records_found, sr.status, sr.error_message, ds.source_url
		FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id JOIN banks b ON b.id = ds.bank_id
		WHERE sr.id = $1
	`, runID).Scan(&det.RunID, &det.BankName, &det.SourceLabel, &det.SourceID, &det.StartedAt, &det.CompletedAt, &det.Found, &det.Status, &errMsg, &det.SourceURL)
	if err != nil {
		return nil, fmt.Errorf("db: scrape run detail: %w", err)
	}
	if errMsg != nil {
		det.ErrorMessage = *errMsg
	}

	// Consecutive fails: walk backwards from this run.
	rows, err := d.pool.Query(ctx, `
		SELECT status FROM scrape_runs WHERE source_id = $1 AND started_at <= $2 ORDER BY started_at DESC
	`, det.SourceID, det.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("db: consecutive fails: %w", err)
	}
	for rows.Next() {
		var st string
		if err := rows.Scan(&st); err != nil {
			rows.Close()
			return nil, err
		}
		if st != "failed" {
			break
		}
		det.ConsecutiveFails++
	}
	rows.Close()

	// Expected records: average records_found across the last 5 successful
	// runs for this source, as a plausibility baseline for "current" vs
	// "expected" — a real, derived figure, not an invented target.
	_ = d.pool.QueryRow(ctx, `
		SELECT coalesce(round(avg(records_found)), 0) FROM (
			SELECT records_found FROM scrape_runs WHERE source_id = $1 AND status = 'success' ORDER BY started_at DESC LIMIT 5
		) recent
	`, det.SourceID).Scan(&det.ExpectedRecords)

	return &det, nil
}

// ===== Banks =====

type BankAdminRow struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	BankType     string    `json:"bank_type"`
	Status       string    `json:"status"`
	Website      string    `json:"website"`
	ProductCount int       `json:"product_count"`
	SourceCount  int       `json:"source_count"`
	HealthPct    *int      `json:"health_pct"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// ListBanksAdmin's Health% is the real success rate of a bank's last 20
// scrape_runs (all of its sources combined) — not an invented score.
// null when the bank has no runs yet.
func (d *DB) ListBanksAdmin(ctx context.Context) ([]BankAdminRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT b.id, b.name, b.code, b.bank_type, b.status, coalesce(b.website, ''),
			(SELECT count(*) FROM products p WHERE p.bank_id = b.id),
			(SELECT count(*) FROM data_sources ds WHERE ds.bank_id = b.id),
			(SELECT round(100.0 * count(*) FILTER (WHERE status = 'success') / NULLIF(count(*), 0))
			 FROM (SELECT sr.status FROM scrape_runs sr JOIN data_sources ds ON ds.id = sr.source_id WHERE ds.bank_id = b.id ORDER BY sr.started_at DESC LIMIT 20) recent),
			(SELECT max(r.scraped_at) FROM product_rates r JOIN products p ON p.id = r.product_id WHERE p.bank_id = b.id)
		FROM banks b ORDER BY b.name
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list banks admin: %w", err)
	}
	defer rows.Close()
	out := []BankAdminRow{}
	for rows.Next() {
		var r BankAdminRow
		var health *float64
		if err := rows.Scan(&r.ID, &r.Name, &r.Code, &r.BankType, &r.Status, &r.Website, &r.ProductCount, &r.SourceCount, &health, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if health != nil {
			v := int(*health)
			r.HealthPct = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) AddBank(ctx context.Context, name, code, bankType, website string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO banks (name, code, slug, bank_type, website) VALUES ($1, $2, lower($2), $3, $4)
		RETURNING id
	`, name, code, bankType, nullIfEmpty(website)).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("db: add bank: %w", err)
	}
	return id, nil
}

func (d *DB) UpdateBank(ctx context.Context, id int64, name, bankType, website, status string) error {
	tag, err := d.pool.Exec(ctx, `
		UPDATE banks SET name = $1, bank_type = $2, website = $3, status = $4 WHERE id = $5
	`, name, bankType, nullIfEmpty(website), status, id)
	if err != nil {
		return fmt.Errorf("db: update bank: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ===== Products (category taxonomy) =====

type ProductCategoryRow struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	GroupName  string `json:"group_name"`
	BankCount  int    `json:"bank_count"`
	FieldCount int    `json:"field_count"`
}

// FieldCount is the number of distinct tenure/rate-label variations
// tracked under this category — a real, derivable proxy for "how many
// distinct fields this product type carries" since every category shares
// the same underlying product_rates columns.
func (d *DB) ListProductCategories(ctx context.Context) ([]ProductCategoryRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT c.id, c.code, c.name, coalesce(parent.name, c.name),
			(SELECT count(DISTINCT p.bank_id) FROM products p WHERE p.category_id = c.id),
			(SELECT count(DISTINCT (r.tenure_label, r.rate_label)) FROM product_rates r JOIN products p ON p.id = r.product_id WHERE p.category_id = c.id)
		FROM product_categories c
		LEFT JOIN product_categories parent ON parent.id = c.parent_id
		WHERE c.parent_id IS NOT NULL
		ORDER BY coalesce(parent.name, c.name), c.name
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list product categories: %w", err)
	}
	defer rows.Close()
	out := []ProductCategoryRow{}
	for rows.Next() {
		var r ProductCategoryRow
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.GroupName, &r.BankCount, &r.FieldCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type CategoryBankRow struct {
	BankName    string `json:"bank_name"`
	ProductName string `json:"product_name"`
}

func (d *DB) ListBanksForCategory(ctx context.Context, categoryID int64) ([]CategoryBankRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT b.name, p.name FROM products p JOIN banks b ON b.id = p.bank_id
		WHERE p.category_id = $1 ORDER BY b.name
	`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("db: banks for category: %w", err)
	}
	defer rows.Close()
	out := []CategoryBankRow{}
	for rows.Next() {
		var r CategoryBankRow
		if err := rows.Scan(&r.BankName, &r.ProductName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

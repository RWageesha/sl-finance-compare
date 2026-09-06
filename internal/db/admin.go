package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/wageesha/sl-finance-compare/internal/models"
)

var ErrNotFound = errors.New("db: not found")
var ErrConflict = errors.New("db: conflict")

// ===== Admin users / sessions =====

func (d *DB) CountSuperAdmins(ctx context.Context) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `SELECT count(*) FROM admin_users WHERE role = 'super_admin'`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("db: count super admins: %w", err)
	}
	return n, nil
}

func (d *DB) CreateAdminUser(ctx context.Context, username, email, passwordHash, role string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO admin_users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, username, email, passwordHash, role).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrConflict
		}
		return 0, fmt.Errorf("db: create admin user: %w", err)
	}
	return id, nil
}

func scanAdminUser(row pgx.Row) (*models.AdminUser, error) {
	var a models.AdminUser
	err := row.Scan(&a.ID, &a.Username, &a.Email, &a.PasswordHash, &a.Role, &a.Status, &a.CreatedAt, &a.UpdatedAt, &a.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: scan admin user: %w", err)
	}
	return &a, nil
}

const adminUserColumns = `id, username, email, password_hash, role, status, created_at, updated_at, last_login_at`

func (d *DB) GetAdminByUsername(ctx context.Context, username string) (*models.AdminUser, error) {
	row := d.pool.QueryRow(ctx, `SELECT `+adminUserColumns+` FROM admin_users WHERE username = $1`, username)
	return scanAdminUser(row)
}

func (d *DB) GetAdminByID(ctx context.Context, id int64) (*models.AdminUser, error) {
	row := d.pool.QueryRow(ctx, `SELECT `+adminUserColumns+` FROM admin_users WHERE id = $1`, id)
	return scanAdminUser(row)
}

func (d *DB) ListAdminUsers(ctx context.Context) ([]models.AdminUser, error) {
	rows, err := d.pool.Query(ctx, `SELECT `+adminUserColumns+` FROM admin_users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("db: list admin users: %w", err)
	}
	defer rows.Close()
	var out []models.AdminUser
	for rows.Next() {
		a, err := scanAdminUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (d *DB) UpdateAdminLastLogin(ctx context.Context, id int64) error {
	_, err := d.pool.Exec(ctx, `UPDATE admin_users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func (d *DB) UpdateAdminUsername(ctx context.Context, id int64, newUsername string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE admin_users SET username = $1, updated_at = now() WHERE id = $2`, newUsername, id)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("db: update admin username: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) UpdateAdminPassword(ctx context.Context, id int64, newHash string) error {
	_, err := d.pool.Exec(ctx, `UPDATE admin_users SET password_hash = $1, updated_at = now() WHERE id = $2`, newHash, id)
	if err != nil {
		return fmt.Errorf("db: update admin password: %w", err)
	}
	return nil
}

func (d *DB) UpdateAdminRoleEmail(ctx context.Context, id int64, email, role string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE admin_users SET email = $1, role = $2, updated_at = now() WHERE id = $3`, email, role, id)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("db: update admin role/email: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) SetAdminStatus(ctx context.Context, id int64, status string) error {
	tag, err := d.pool.Exec(ctx, `UPDATE admin_users SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("db: set admin status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) CreateSession(ctx context.Context, adminID int64, tokenHash string, expiresAt time.Time) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO admin_sessions (admin_id, token_hash, expires_at) VALUES ($1, $2, $3)`, adminID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("db: create session: %w", err)
	}
	return nil
}

// GetSessionAdmin resolves a session token hash to its admin, only if the
// session hasn't expired and the admin account is still active.
func (d *DB) GetSessionAdmin(ctx context.Context, tokenHash string) (*models.AdminUser, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT a.`+adminUserColumns2()+`
		FROM admin_sessions s
		JOIN admin_users a ON a.id = s.admin_id
		WHERE s.token_hash = $1 AND s.expires_at > now() AND a.status = 'active'
	`, tokenHash)
	return scanAdminUser(row)
}

func adminUserColumns2() string {
	return "id, a.username, a.email, a.password_hash, a.role, a.status, a.created_at, a.updated_at, a.last_login_at"
}

func (d *DB) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM admin_sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// ===== Audit logs =====

func (d *DB) InsertAuditLog(ctx context.Context, adminID *int64, adminUsername, action, recordRef, oldValue, newValue string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO audit_logs (admin_id, admin_username, action, record_ref, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, adminID, adminUsername, action, recordRef, nullIfEmpty(oldValue), nullIfEmpty(newValue))
	if err != nil {
		return fmt.Errorf("db: insert audit log: %w", err)
	}
	return nil
}

type AuditLogFilter struct {
	AdminUsername string
	Action        string
	Since         *time.Time
	Page, PerPage int
}

func (d *DB) ListAuditLogs(ctx context.Context, f AuditLogFilter) ([]models.AuditLog, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if f.AdminUsername != "" {
		args = append(args, f.AdminUsername)
		where += fmt.Sprintf(" AND admin_username = $%d", len(args))
	}
	if f.Action != "" {
		args = append(args, f.Action)
		where += fmt.Sprintf(" AND action = $%d", len(args))
	}
	if f.Since != nil {
		args = append(args, *f.Since)
		where += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}

	var total int
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("db: count audit logs: %w", err)
	}

	page, perPage := pagination(f.Page, f.PerPage)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, admin_id, admin_username, action, record_ref, old_value, new_value, created_at
		FROM audit_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("db: list audit logs: %w", err)
	}
	defer rows.Close()

	var out []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		var oldVal, newVal *string
		if err := rows.Scan(&l.ID, &l.AdminID, &l.AdminUsername, &l.Action, &l.RecordRef, &oldVal, &newVal, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("db: scan audit log: %w", err)
		}
		if oldVal != nil {
			l.OldValue = *oldVal
		}
		if newVal != nil {
			l.NewValue = *newVal
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (d *DB) ClearAuditLogs(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM audit_logs`)
	return err
}

func (d *DB) ListDistinctAuditAdmins(ctx context.Context) ([]string, error) {
	rows, err := d.pool.Query(ctx, `SELECT DISTINCT admin_username FROM audit_logs ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ===== User reports (public correction submissions) =====

func (d *DB) CreateUserReport(ctx context.Context, r models.UserReport) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO user_reports (bank_name, product_label, issue_type, current_value, correct_value, source_url, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, r.BankName, r.ProductLabel, r.IssueType, nullIfEmpty(r.CurrentValue), r.CorrectValue, r.SourceURL, nullIfEmpty(r.Description)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: create user report: %w", err)
	}
	return id, nil
}

type UserReportFilter struct {
	Status        string
	Since         *time.Time
	Page, PerPage int
}

func (d *DB) ListUserReports(ctx context.Context, f UserReportFilter) ([]models.UserReport, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if f.Since != nil {
		args = append(args, *f.Since)
		where += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	var total int
	if err := d.pool.QueryRow(ctx, `SELECT count(*) FROM user_reports `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("db: count user reports: %w", err)
	}
	page, perPage := pagination(f.Page, f.PerPage)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, bank_name, product_label, issue_type, current_value, correct_value, source_url, description, status, created_at, resolved_at, resolved_by
		FROM user_reports %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("db: list user reports: %w", err)
	}
	defer rows.Close()
	var out []models.UserReport
	for rows.Next() {
		var r models.UserReport
		var curVal, desc *string
		if err := rows.Scan(&r.ID, &r.BankName, &r.ProductLabel, &r.IssueType, &curVal, &r.CorrectValue, &r.SourceURL, &desc, &r.Status, &r.CreatedAt, &r.ResolvedAt, &r.ResolvedByID); err != nil {
			return nil, 0, fmt.Errorf("db: scan user report: %w", err)
		}
		if curVal != nil {
			r.CurrentValue = *curVal
		}
		if desc != nil {
			r.Description = *desc
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func (d *DB) SetUserReportStatus(ctx context.Context, id int64, status string, resolvedBy int64) error {
	tag, err := d.pool.Exec(ctx, `
		UPDATE user_reports SET status = $1, resolved_at = now(), resolved_by = $2 WHERE id = $3
	`, status, resolvedBy, id)
	if err != nil {
		return fmt.Errorf("db: set user report status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) GetUserReport(ctx context.Context, id int64) (*models.UserReport, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT id, bank_name, product_label, issue_type, current_value, correct_value, source_url, description, status, created_at, resolved_at, resolved_by
		FROM user_reports WHERE id = $1
	`, id)
	var r models.UserReport
	var curVal, desc *string
	err := row.Scan(&r.ID, &r.BankName, &r.ProductLabel, &r.IssueType, &curVal, &r.CorrectValue, &r.SourceURL, &desc, &r.Status, &r.CreatedAt, &r.ResolvedAt, &r.ResolvedByID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: get user report: %w", err)
	}
	if curVal != nil {
		r.CurrentValue = *curVal
	}
	if desc != nil {
		r.Description = *desc
	}
	return &r, nil
}

// ===== helpers =====

func pagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 200 {
		perPage = 20
	}
	return page, perPage
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

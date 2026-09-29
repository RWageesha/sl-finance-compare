package db

import (
	"context"
	"fmt"
	"time"
)

// ===== Ad slots & ads =====

type AdSlotRow struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

func (d *DB) ListAdSlots(ctx context.Context) ([]AdSlotRow, error) {
	rows, err := d.pool.Query(ctx, `SELECT id, key, label, kind FROM ad_slots ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("db: list ad slots: %w", err)
	}
	defer rows.Close()
	out := []AdSlotRow{}
	for rows.Next() {
		var r AdSlotRow
		if err := rows.Scan(&r.ID, &r.Key, &r.Label, &r.Kind); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type AdRow struct {
	ID        int64      `json:"id"`
	SlotID    int64      `json:"slot_id"`
	SlotKey   string     `json:"slot_key"`
	AdType    string     `json:"ad_type"`
	Title     string     `json:"title"`
	ImageURL  string     `json:"image_url"`
	TargetURL string     `json:"target_url"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListAdsAdmin returns every ad regardless of active/date-window status —
// the admin UI needs to show scheduled and expired ads too, not just what
// a visitor currently sees (that's ListActiveAdsForSlot below).
func (d *DB) ListAdsAdmin(ctx context.Context) ([]AdRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT a.id, a.slot_id, s.key, a.ad_type, a.title, a.image_url, a.target_url,
			a.sort_order, a.is_active, a.starts_at, a.ends_at, a.created_at
		FROM ads a JOIN ad_slots s ON s.id = a.slot_id
		ORDER BY s.id, a.sort_order, a.id
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list ads admin: %w", err)
	}
	defer rows.Close()
	out := []AdRow{}
	for rows.Next() {
		var r AdRow
		if err := rows.Scan(&r.ID, &r.SlotID, &r.SlotKey, &r.AdType, &r.Title, &r.ImageURL, &r.TargetURL,
			&r.SortOrder, &r.IsActive, &r.StartsAt, &r.EndsAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type PublicAdRow struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url"`
	TargetURL string `json:"target_url"`
}

// ListActiveAdsForSlot is the public, unauthenticated read: only ads that
// are active and inside their date window right now, for one slot key.
// Filtering the date window here (not in the frontend) means the
// frontend never has to reason about scheduling at all.
func (d *DB) ListActiveAdsForSlot(ctx context.Context, slotKey string) ([]PublicAdRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT a.id, a.title, a.image_url, a.target_url
		FROM ads a JOIN ad_slots s ON s.id = a.slot_id
		WHERE s.key = $1 AND a.is_active
			AND a.starts_at <= now() AND (a.ends_at IS NULL OR a.ends_at >= now())
		ORDER BY a.sort_order, a.id
	`, slotKey)
	if err != nil {
		return nil, fmt.Errorf("db: list active ads: %w", err)
	}
	defer rows.Close()
	out := []PublicAdRow{}
	for rows.Next() {
		var r PublicAdRow
		if err := rows.Scan(&r.ID, &r.Title, &r.ImageURL, &r.TargetURL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type AdInput struct {
	SlotID    int64
	AdType    string
	Title     string
	ImageURL  string
	TargetURL string
	SortOrder int
	IsActive  bool
	StartsAt  time.Time
	EndsAt    *time.Time
}

func (d *DB) CreateAd(ctx context.Context, in AdInput, createdByAdminID int64) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, `
		INSERT INTO ads (slot_id, ad_type, title, image_url, target_url, sort_order, is_active, starts_at, ends_at, created_by_admin_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, in.SlotID, in.AdType, in.Title, in.ImageURL, in.TargetURL, in.SortOrder, in.IsActive, in.StartsAt, in.EndsAt, createdByAdminID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: create ad: %w", err)
	}
	return id, nil
}

func (d *DB) UpdateAd(ctx context.Context, id int64, in AdInput) error {
	tag, err := d.pool.Exec(ctx, `
		UPDATE ads SET slot_id = $1, ad_type = $2, title = $3, image_url = $4, target_url = $5,
			sort_order = $6, is_active = $7, starts_at = $8, ends_at = $9, updated_at = now()
		WHERE id = $10
	`, in.SlotID, in.AdType, in.Title, in.ImageURL, in.TargetURL, in.SortOrder, in.IsActive, in.StartsAt, in.EndsAt, id)
	if err != nil {
		return fmt.Errorf("db: update ad: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) DeleteAd(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM ads WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("db: delete ad: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ===== Homepage section visibility/layout =====

type SiteSectionRow struct {
	ID                 int64   `json:"id"`
	Page               string  `json:"page"`
	SectionKey         string  `json:"section_key"`
	Label              string  `json:"label"`
	IsVisible          bool    `json:"is_visible"`
	LayoutVariant      string  `json:"layout_variant"`
	BackgroundImageURL *string `json:"background_image_url"`
	SortOrder          int     `json:"sort_order"`
}

func (d *DB) ListSiteSections(ctx context.Context, page string) ([]SiteSectionRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, page, section_key, label, is_visible, layout_variant, background_image_url, sort_order
		FROM site_sections WHERE page = $1 ORDER BY sort_order
	`, page)
	if err != nil {
		return nil, fmt.Errorf("db: list site sections: %w", err)
	}
	defer rows.Close()
	out := []SiteSectionRow{}
	for rows.Next() {
		var r SiteSectionRow
		if err := rows.Scan(&r.ID, &r.Page, &r.SectionKey, &r.Label, &r.IsVisible, &r.LayoutVariant, &r.BackgroundImageURL, &r.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) UpdateSiteSection(ctx context.Context, id int64, isVisible bool, layoutVariant, backgroundImageURL string, sortOrder int) error {
	tag, err := d.pool.Exec(ctx, `
		UPDATE site_sections SET is_visible = $1, layout_variant = $2, background_image_url = $3, sort_order = $4, updated_at = now()
		WHERE id = $5
	`, isVisible, layoutVariant, nullIfEmpty(backgroundImageURL), sortOrder, id)
	if err != nil {
		return fmt.Errorf("db: update site section: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ===== Page text content =====

type SiteContentRow struct {
	ID         int64  `json:"id"`
	Page       string `json:"page"`
	ContentKey string `json:"content_key"`
	Label      string `json:"label"`
	Body       string `json:"body"`
}

func (d *DB) ListSiteContent(ctx context.Context, page string) ([]SiteContentRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, page, content_key, label, body FROM site_content WHERE page = $1 ORDER BY id
	`, page)
	if err != nil {
		return nil, fmt.Errorf("db: list site content: %w", err)
	}
	defer rows.Close()
	out := []SiteContentRow{}
	for rows.Next() {
		var r SiteContentRow
		if err := rows.Scan(&r.ID, &r.Page, &r.ContentKey, &r.Label, &r.Body); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) UpdateSiteContent(ctx context.Context, id int64, body string, updatedByAdminID int64) error {
	tag, err := d.pool.Exec(ctx, `
		UPDATE site_content SET body = $1, updated_by_admin_id = $2, updated_at = now() WHERE id = $3
	`, body, updatedByAdminID, id)
	if err != nil {
		return fmt.Errorf("db: update site content: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ===== Bank logo overrides =====

type BankLogoOverrideRow struct {
	BankSlug     string  `json:"bank_slug"`
	LogoURL      *string `json:"logo_url"`
	LogoSmallURL *string `json:"logo_small_url"`
}

func (d *DB) ListBankLogoOverrides(ctx context.Context) ([]BankLogoOverrideRow, error) {
	rows, err := d.pool.Query(ctx, `SELECT bank_slug, logo_url, logo_small_url FROM bank_logo_overrides ORDER BY bank_slug`)
	if err != nil {
		return nil, fmt.Errorf("db: list bank logo overrides: %w", err)
	}
	defer rows.Close()
	out := []BankLogoOverrideRow{}
	for rows.Next() {
		var r BankLogoOverrideRow
		if err := rows.Scan(&r.BankSlug, &r.LogoURL, &r.LogoSmallURL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpsertBankLogoOverride sets either or both logo URLs for a bank slug —
// passing an empty string for one clears just that field rather than
// requiring both to be set together.
func (d *DB) UpsertBankLogoOverride(ctx context.Context, bankSlug, logoURL, logoSmallURL string, updatedByAdminID int64) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO bank_logo_overrides (bank_slug, logo_url, logo_small_url, updated_by_admin_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (bank_slug) DO UPDATE SET
			logo_url = $2, logo_small_url = $3, updated_by_admin_id = $4, updated_at = now()
	`, bankSlug, nullIfEmpty(logoURL), nullIfEmpty(logoSmallURL), updatedByAdminID)
	if err != nil {
		return fmt.Errorf("db: upsert bank logo override: %w", err)
	}
	return nil
}

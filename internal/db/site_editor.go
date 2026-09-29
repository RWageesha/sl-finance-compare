package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ===== Ad slots & ads =====
//
// An ad can be placed on several page+slot combinations at once
// (ad_placements) and carry several creative files (ad_creatives — 3-5
// for a slider, 2-3 for a shared-slot ad, exactly 1 otherwise). Both are
// replaced wholesale on every CreateAd/UpdateAd inside one transaction
// rather than diffed, since the admin wizard always resubmits the full
// set.

type AdSlotRow struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Label       string `json:"label"`
	PageKey     string `json:"page_key"`
	Orientation string `json:"orientation"`
	Size        string `json:"size"`
}

func (d *DB) ListAdSlots(ctx context.Context) ([]AdSlotRow, error) {
	rows, err := d.pool.Query(ctx, `SELECT id, key, label, page_key, orientation, size FROM ad_slots ORDER BY page_key, id`)
	if err != nil {
		return nil, fmt.Errorf("db: list ad slots: %w", err)
	}
	defer rows.Close()
	out := []AdSlotRow{}
	for rows.Next() {
		var r AdSlotRow
		if err := rows.Scan(&r.ID, &r.Key, &r.Label, &r.PageKey, &r.Orientation, &r.Size); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type AdCreativeRow struct {
	ID        int64   `json:"id"`
	Position  int     `json:"position"`
	MediaURL  string  `json:"media_url"`
	TargetURL string  `json:"target_url"`
	AltText   *string `json:"alt_text"`
	PosterURL *string `json:"poster_url"`
}

type AdPlacementRow struct {
	SlotID  int64  `json:"slot_id"`
	SlotKey string `json:"slot_key"`
	PageKey string `json:"page_key"`
}

type AdRow struct {
	ID         int64            `json:"id"`
	Title      string           `json:"title"`
	AdType     string           `json:"ad_type"`
	Advertiser *string          `json:"advertiser"`
	Layout     string           `json:"layout"`
	Style      string           `json:"style"`
	Settings   json.RawMessage  `json:"settings"`
	Devices    string           `json:"devices"`
	SortOrder  int              `json:"sort_order"`
	Weight     int              `json:"weight"`
	IsActive   bool             `json:"is_active"`
	StartsAt   time.Time        `json:"starts_at"`
	EndsAt     *time.Time       `json:"ends_at"`
	CreatedAt  time.Time        `json:"created_at"`
	Creatives  []AdCreativeRow  `json:"creatives"`
	Placements []AdPlacementRow `json:"placements"`
}

// ListAdsAdmin returns every ad regardless of active/date-window status —
// the admin UI needs to show scheduled and expired ads too, not just what
// a visitor currently sees (that's ListActiveAdsForSlot below). Three
// plain queries assembled in Go, rather than a jsonb_agg one-liner, to
// match this codebase's existing query style.
func (d *DB) ListAdsAdmin(ctx context.Context) ([]AdRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, title, ad_type, advertiser, layout, style, settings, devices,
			sort_order, weight, is_active, starts_at, ends_at, created_at
		FROM ads ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list ads admin: %w", err)
	}
	out := []AdRow{}
	byID := map[int64]*AdRow{}
	for rows.Next() {
		var r AdRow
		if err := rows.Scan(&r.ID, &r.Title, &r.AdType, &r.Advertiser, &r.Layout, &r.Style, &r.Settings, &r.Devices,
			&r.SortOrder, &r.Weight, &r.IsActive, &r.StartsAt, &r.EndsAt, &r.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		r.Creatives = []AdCreativeRow{}
		r.Placements = []AdPlacementRow{}
		out = append(out, r)
		byID[r.ID] = &out[len(out)-1]
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	crows, err := d.pool.Query(ctx, `SELECT ad_id, id, position, media_url, target_url, alt_text, poster_url FROM ad_creatives ORDER BY ad_id, position`)
	if err != nil {
		return nil, fmt.Errorf("db: list ad creatives: %w", err)
	}
	for crows.Next() {
		var adID int64
		var c AdCreativeRow
		if err := crows.Scan(&adID, &c.ID, &c.Position, &c.MediaURL, &c.TargetURL, &c.AltText, &c.PosterURL); err != nil {
			crows.Close()
			return nil, err
		}
		if ad, ok := byID[adID]; ok {
			ad.Creatives = append(ad.Creatives, c)
		}
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return nil, err
	}

	prows, err := d.pool.Query(ctx, `
		SELECT p.ad_id, s.id, s.key, s.page_key FROM ad_placements p JOIN ad_slots s ON s.id = p.slot_id ORDER BY p.ad_id, s.id
	`)
	if err != nil {
		return nil, fmt.Errorf("db: list ad placements: %w", err)
	}
	defer prows.Close()
	for prows.Next() {
		var adID int64
		var p AdPlacementRow
		if err := prows.Scan(&adID, &p.SlotID, &p.SlotKey, &p.PageKey); err != nil {
			return nil, err
		}
		if ad, ok := byID[adID]; ok {
			ad.Placements = append(ad.Placements, p)
		}
	}
	return out, prows.Err()
}

type PublicAdRow struct {
	ID        int64           `json:"id"`
	Title     string          `json:"title"`
	Layout    string          `json:"layout"`
	Style     string          `json:"style"`
	Settings  json.RawMessage `json:"settings"`
	Devices   string          `json:"devices"`
	Weight    int             `json:"weight"`
	Creatives []AdCreativeRow `json:"creatives"`
}

// ListActiveAdsForSlot is the public, unauthenticated read: only ads that
// are active and inside their date window right now, placed on this slot
// key. Filtering active/date here (not in the frontend) means the
// frontend never has to reason about scheduling — it only has to filter
// by device and pick one by weight (see AdSlot.vue), since neither of
// those is known server-side without user-agent sniffing.
func (d *DB) ListActiveAdsForSlot(ctx context.Context, slotKey string) ([]PublicAdRow, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT a.id, a.title, a.layout, a.style, a.settings, a.devices, a.weight
		FROM ads a
		JOIN ad_placements p ON p.ad_id = a.id
		JOIN ad_slots s ON s.id = p.slot_id
		WHERE s.key = $1 AND a.is_active
			AND a.starts_at <= now() AND (a.ends_at IS NULL OR a.ends_at >= now())
		ORDER BY a.sort_order, a.id
	`, slotKey)
	if err != nil {
		return nil, fmt.Errorf("db: list active ads: %w", err)
	}
	out := []PublicAdRow{}
	byID := map[int64]*PublicAdRow{}
	for rows.Next() {
		var r PublicAdRow
		if err := rows.Scan(&r.ID, &r.Title, &r.Layout, &r.Style, &r.Settings, &r.Devices, &r.Weight); err != nil {
			rows.Close()
			return nil, err
		}
		r.Creatives = []AdCreativeRow{}
		out = append(out, r)
		byID[r.ID] = &out[len(out)-1]
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	ids := make([]int64, len(out))
	for i, r := range out {
		ids[i] = r.ID
	}
	crows, err := d.pool.Query(ctx, `SELECT ad_id, id, position, media_url, target_url, alt_text, poster_url FROM ad_creatives WHERE ad_id = ANY($1) ORDER BY ad_id, position`, ids)
	if err != nil {
		return nil, fmt.Errorf("db: list active ad creatives: %w", err)
	}
	defer crows.Close()
	for crows.Next() {
		var adID int64
		var c AdCreativeRow
		if err := crows.Scan(&adID, &c.ID, &c.Position, &c.MediaURL, &c.TargetURL, &c.AltText, &c.PosterURL); err != nil {
			return nil, err
		}
		if ad, ok := byID[adID]; ok {
			ad.Creatives = append(ad.Creatives, c)
		}
	}
	return out, crows.Err()
}

type AdCreativeInput struct {
	Position  int
	MediaURL  string
	TargetURL string
	AltText   string
	PosterURL string
}

type AdInput struct {
	Title      string
	AdType     string
	Advertiser string
	Layout     string
	Style      string
	Settings   json.RawMessage
	Devices    string
	SortOrder  int
	Weight     int
	IsActive   bool
	StartsAt   time.Time
	EndsAt     *time.Time
	Creatives  []AdCreativeInput
	SlotIDs    []int64
}

func insertAdCreativesAndPlacements(ctx context.Context, tx pgx.Tx, adID int64, in AdInput) error {
	for _, c := range in.Creatives {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ad_creatives (ad_id, position, media_url, target_url, alt_text, poster_url)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, adID, c.Position, c.MediaURL, c.TargetURL, nullIfEmpty(c.AltText), nullIfEmpty(c.PosterURL)); err != nil {
			return fmt.Errorf("db: insert ad creative: %w", err)
		}
	}
	for _, slotID := range in.SlotIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO ad_placements (ad_id, slot_id) VALUES ($1, $2)`, adID, slotID); err != nil {
			return fmt.Errorf("db: insert ad placement: %w", err)
		}
	}
	return nil
}

func (d *DB) CreateAd(ctx context.Context, in AdInput, createdByAdminID int64) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("db: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO ads (title, ad_type, advertiser, layout, style, settings, devices, sort_order, weight, is_active, starts_at, ends_at, created_by_admin_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, in.Title, in.AdType, nullIfEmpty(in.Advertiser), in.Layout, in.Style, in.Settings, in.Devices, in.SortOrder, in.Weight, in.IsActive, in.StartsAt, in.EndsAt, createdByAdminID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: create ad: %w", err)
	}
	if err := insertAdCreativesAndPlacements(ctx, tx, id, in); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("db: commit tx: %w", err)
	}
	return id, nil
}

func (d *DB) UpdateAd(ctx context.Context, id int64, in AdInput) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE ads SET title = $1, ad_type = $2, advertiser = $3, layout = $4, style = $5, settings = $6,
			devices = $7, sort_order = $8, weight = $9, is_active = $10, starts_at = $11, ends_at = $12, updated_at = now()
		WHERE id = $13
	`, in.Title, in.AdType, nullIfEmpty(in.Advertiser), in.Layout, in.Style, in.Settings, in.Devices, in.SortOrder, in.Weight, in.IsActive, in.StartsAt, in.EndsAt, id)
	if err != nil {
		return fmt.Errorf("db: update ad: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ad_creatives WHERE ad_id = $1`, id); err != nil {
		return fmt.Errorf("db: clear ad creatives: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ad_placements WHERE ad_id = $1`, id); err != nil {
		return fmt.Errorf("db: clear ad placements: %w", err)
	}
	if err := insertAdCreativesAndPlacements(ctx, tx, id, in); err != nil {
		return err
	}
	return tx.Commit(ctx)
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

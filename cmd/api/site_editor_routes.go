package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/db"
)

// registerSiteEditorRoutes wires the admin-editable-content module: ad
// slots/ads, homepage section visibility, page text blocks, and bank logo
// overrides. Every /api/v1/site/* route is public and unauthenticated —
// the frontend is a static build regenerated only on deploy, so these
// have to be fetched at runtime for an admin edit to ever reach a
// visitor without a rebuild (see migrations/007_site_editor.sql).
func registerSiteEditorRoutes(mux *http.ServeMux, database *db.DB) {
	admin := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, h) }
	adminUp := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, requireRole("admin", h)) }

	mux.HandleFunc("GET /api/v1/admin/ad-slots", admin(handleAdminAdSlotsList(database)))
	mux.HandleFunc("GET /api/v1/admin/ads", admin(handleAdminAdsList(database)))
	mux.HandleFunc("POST /api/v1/admin/ads", adminUp(handleAdminAdCreate(database)))
	mux.HandleFunc("PATCH /api/v1/admin/ads/{id}", adminUp(handleAdminAdUpdate(database)))
	mux.HandleFunc("DELETE /api/v1/admin/ads/{id}", adminUp(handleAdminAdDelete(database)))

	mux.HandleFunc("GET /api/v1/admin/site-sections", admin(handleAdminSiteSectionsList(database)))
	mux.HandleFunc("PATCH /api/v1/admin/site-sections/{id}", adminUp(handleAdminSiteSectionUpdate(database)))

	mux.HandleFunc("GET /api/v1/admin/site-content", admin(handleAdminSiteContentList(database)))
	mux.HandleFunc("PATCH /api/v1/admin/site-content/{id}", adminUp(handleAdminSiteContentUpdate(database)))

	mux.HandleFunc("GET /api/v1/admin/bank-logos", admin(handleAdminBankLogosList(database)))
	mux.HandleFunc("PUT /api/v1/admin/bank-logos/{slug}", adminUp(handleAdminBankLogoUpsert(database)))

	mux.HandleFunc("POST /api/v1/admin/media/upload", adminUp(handleAdminMediaUpload()))

	mux.HandleFunc("GET /api/v1/site/ads", handlePublicAds(database))
	mux.HandleFunc("GET /api/v1/site/sections", handlePublicSiteSections(database))
	mux.HandleFunc("GET /api/v1/site/content", handlePublicSiteContent(database))
	mux.HandleFunc("GET /api/v1/site/bank-logos", handlePublicBankLogos(database))
}

// ===== Admin: ad slots & ads =====

func handleAdminAdSlotsList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slots, err := database.ListAdSlots(r.Context())
		if err != nil {
			log.Printf("admin: ad slots list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load ad slots")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": slots})
	}
}

func handleAdminAdsList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ads, err := database.ListAdsAdmin(r.Context())
		if err != nil {
			log.Printf("admin: ads list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load ads")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": ads})
	}
}

type adCreativeBody struct {
	Position  int    `json:"position"`
	MediaURL  string `json:"media_url"`
	TargetURL string `json:"target_url"`
	AltText   string `json:"alt_text"`
	PosterURL string `json:"poster_url"`
}

type adRequestBody struct {
	Title      string           `json:"title"`
	AdType     string           `json:"ad_type"`
	Advertiser string           `json:"advertiser"`
	Layout     string           `json:"layout"` // 'horizontal' | 'vertical'
	Style      string           `json:"style"`
	Settings   json.RawMessage  `json:"settings"`
	Devices    string           `json:"devices"`
	SortOrder  int              `json:"sort_order"`
	Weight     int              `json:"weight"`
	IsActive   bool             `json:"is_active"`
	StartsAt   *time.Time       `json:"starts_at"`
	EndsAt     *time.Time       `json:"ends_at"`
	Creatives  []adCreativeBody `json:"creatives"`
	SlotIDs    []int64          `json:"slot_ids"`
}

// adStyleLimits mirrors the wizard's own STYLES min/max (image/gif/video/
// overlay: exactly 1 creative; slider: 3-5; shared: 2-3) — enforced here
// too, not just in the form, since the API is the actual source of truth.
var adStyleLimits = map[string]struct{ Min, Max int }{
	"image": {1, 1}, "gif": {1, 1}, "video": {1, 1}, "overlay": {1, 1},
	"slider": {3, 5}, "shared": {2, 3},
}
var validAdTypes = map[string]bool{"house": true, "sponsored": true, "partner": true}
var validDevices = map[string]bool{"all": true, "desktop": true, "mobile": true}

var urlPattern = regexp.MustCompile(`^https?://[^\s]+$`)

// validateAdBody re-runs the wizard's own validation rules server-side,
// plus one it can't (a placement's slot orientation must match the ad's
// layout — the wizard only greys those out client-side).
func validateAdBody(ctx context.Context, database *db.DB, b *adRequestBody) string {
	limits, ok := adStyleLimits[b.Style]
	if !ok {
		return "style must be one of image, gif, video, slider, shared, or overlay"
	}
	if b.Layout != "horizontal" && b.Layout != "vertical" {
		return "layout must be horizontal or vertical"
	}
	if len(b.Creatives) < limits.Min || len(b.Creatives) > limits.Max {
		if limits.Min == limits.Max {
			return fmt.Sprintf("this style needs exactly %d file(s)", limits.Min)
		}
		return fmt.Sprintf("this style needs %d to %d files", limits.Min, limits.Max)
	}
	for _, c := range b.Creatives {
		if c.MediaURL == "" {
			return "every creative needs an uploaded file or media URL"
		}
		if !urlPattern.MatchString(c.TargetURL) {
			return "every creative needs a target URL starting with http:// or https://"
		}
	}
	if strings.TrimSpace(b.Title) == "" {
		return "title is required"
	}
	if !validAdTypes[b.AdType] {
		return "ad_type must be house, sponsored, or partner"
	}
	if b.AdType != "house" && strings.TrimSpace(b.Advertiser) == "" {
		return "advertiser is required for sponsored and partner ads"
	}
	if !validDevices[b.Devices] {
		return "devices must be all, desktop, or mobile"
	}
	if b.StartsAt == nil {
		return "starts_at is required"
	}
	if b.EndsAt != nil && !b.EndsAt.After(*b.StartsAt) {
		return "ends_at must be after starts_at"
	}
	if b.Weight < 1 || b.Weight > 10 {
		return "weight must be between 1 and 10"
	}
	if len(b.SlotIDs) == 0 {
		return "choose at least one placement"
	}

	wantOrientation := "h"
	if b.Layout == "vertical" {
		wantOrientation = "v"
	}
	if b.Style == "overlay" {
		wantOrientation = "overlay"
	}
	slots, err := database.ListAdSlots(ctx)
	if err != nil {
		return "failed to validate placements"
	}
	orientationByID := make(map[int64]string, len(slots))
	for _, s := range slots {
		orientationByID[s.ID] = s.Orientation
	}
	for _, id := range b.SlotIDs {
		o, ok := orientationByID[id]
		if !ok {
			return "one of the selected placements no longer exists"
		}
		if o != wantOrientation {
			return "a selected placement doesn't match this ad's layout/style"
		}
	}
	return ""
}

func (b adRequestBody) toInput() db.AdInput {
	creatives := make([]db.AdCreativeInput, len(b.Creatives))
	for i, c := range b.Creatives {
		creatives[i] = db.AdCreativeInput{Position: c.Position, MediaURL: c.MediaURL, TargetURL: c.TargetURL, AltText: c.AltText, PosterURL: c.PosterURL}
	}
	settings := b.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	var startsAt time.Time
	if b.StartsAt != nil {
		startsAt = *b.StartsAt
	}
	return db.AdInput{
		Title:      b.Title,
		AdType:     b.AdType,
		Advertiser: b.Advertiser,
		Layout:     b.Layout,
		Style:      b.Style,
		Settings:   settings,
		Devices:    b.Devices,
		SortOrder:  b.SortOrder,
		Weight:     b.Weight,
		IsActive:   b.IsActive,
		StartsAt:   startsAt,
		EndsAt:     b.EndsAt,
		Creatives:  creatives,
		SlotIDs:    b.SlotIDs,
	}
}

func handleAdminAdCreate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		var body adRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if msg := validateAdBody(r.Context(), database, &body); msg != "" {
			writeJSONError(w, http.StatusBadRequest, msg)
			return
		}
		id, err := database.CreateAd(r.Context(), body.toInput(), actor.ID)
		if err != nil {
			log.Printf("admin: create ad: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create ad")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Ad Created", body.Title, "", "")
		writeJSON(w, http.StatusOK, map[string]int64{"id": id})
	}
}

func handleAdminAdUpdate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body adRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if msg := validateAdBody(r.Context(), database, &body); msg != "" {
			writeJSONError(w, http.StatusBadRequest, msg)
			return
		}
		if err := database.UpdateAd(r.Context(), id, body.toInput()); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "ad not found")
				return
			}
			log.Printf("admin: update ad: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Ad Updated", body.Title, "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func handleAdminAdDelete(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := database.DeleteAd(r.Context(), id); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "ad not found")
				return
			}
			log.Printf("admin: delete ad: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Ad Deleted", strconv.FormatInt(id, 10), "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// ===== Admin: homepage sections =====

func handleAdminSiteSectionsList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "home"
		}
		sections, err := database.ListSiteSections(r.Context(), page)
		if err != nil {
			log.Printf("admin: site sections list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load sections")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": sections})
	}
}

func handleAdminSiteSectionUpdate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct {
			IsVisible          bool   `json:"is_visible"`
			LayoutVariant      string `json:"layout_variant"`
			BackgroundImageURL string `json:"background_image_url"`
			SortOrder          int    `json:"sort_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.LayoutVariant == "" {
			body.LayoutVariant = "default"
		}
		if err := database.UpdateSiteSection(r.Context(), id, body.IsVisible, body.LayoutVariant, body.BackgroundImageURL, body.SortOrder); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "section not found")
				return
			}
			log.Printf("admin: update site section: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Section Updated", strconv.FormatInt(id, 10), "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// ===== Admin: page text content =====

func handleAdminSiteContentList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			writeJSONError(w, http.StatusBadRequest, "page is required")
			return
		}
		content, err := database.ListSiteContent(r.Context(), page)
		if err != nil {
			log.Printf("admin: site content list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load content")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": content})
	}
}

func handleAdminSiteContentUpdate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if err := database.UpdateSiteContent(r.Context(), id, body.Body, actor.ID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "content block not found")
				return
			}
			log.Printf("admin: update site content: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Page Content Updated", strconv.FormatInt(id, 10), "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// ===== Admin: bank logo overrides =====

func handleAdminBankLogosList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overrides, err := database.ListBankLogoOverrides(r.Context())
		if err != nil {
			log.Printf("admin: bank logos list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load bank logo overrides")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": overrides})
	}
}

func handleAdminBankLogoUpsert(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		slug := r.PathValue("slug")
		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "slug is required")
			return
		}
		var body struct {
			LogoURL      string `json:"logo_url"`
			LogoSmallURL string `json:"logo_small_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if err := database.UpsertBankLogoOverride(r.Context(), slug, body.LogoURL, body.LogoSmallURL, actor.ID); err != nil {
			log.Printf("admin: upsert bank logo override: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Bank Logo Updated", slug, "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

// ===== Public site-content reads =====

func handlePublicAds(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slot := r.URL.Query().Get("slot")
		if slot == "" {
			writeJSONError(w, http.StatusBadRequest, "slot is required")
			return
		}
		ads, err := database.ListActiveAdsForSlot(r.Context(), slot)
		if err != nil {
			log.Printf("site: ads: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load ads")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": ads})
	}
}

func handlePublicSiteSections(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "home"
		}
		sections, err := database.ListSiteSections(r.Context(), page)
		if err != nil {
			log.Printf("site: sections: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load sections")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": sections})
	}
}

func handlePublicSiteContent(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			writeJSONError(w, http.StatusBadRequest, "page is required")
			return
		}
		content, err := database.ListSiteContent(r.Context(), page)
		if err != nil {
			log.Printf("site: content: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load content")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": content})
	}
}

func handlePublicBankLogos(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overrides, err := database.ListBankLogoOverrides(r.Context())
		if err != nil {
			log.Printf("site: bank logos: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load bank logos")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": overrides})
	}
}

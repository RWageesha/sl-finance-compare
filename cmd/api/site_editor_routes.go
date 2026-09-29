package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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

type adRequestBody struct {
	SlotID    int64      `json:"slot_id"`
	AdType    string     `json:"ad_type"`
	Title     string     `json:"title"`
	ImageURL  string     `json:"image_url"`
	TargetURL string     `json:"target_url"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	StartsAt  *time.Time `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at"`
}

func (b adRequestBody) toInput() db.AdInput {
	startsAt := time.Now()
	if b.StartsAt != nil {
		startsAt = *b.StartsAt
	}
	adType := b.AdType
	if adType == "" {
		adType = "house"
	}
	return db.AdInput{
		SlotID:    b.SlotID,
		AdType:    adType,
		Title:     b.Title,
		ImageURL:  b.ImageURL,
		TargetURL: b.TargetURL,
		SortOrder: b.SortOrder,
		IsActive:  b.IsActive,
		StartsAt:  startsAt,
		EndsAt:    b.EndsAt,
	}
}

func handleAdminAdCreate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		var body adRequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SlotID == 0 || body.Title == "" || body.ImageURL == "" || body.TargetURL == "" {
			writeJSONError(w, http.StatusBadRequest, "slot_id, title, image_url and target_url are required")
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SlotID == 0 || body.Title == "" || body.ImageURL == "" || body.TargetURL == "" {
			writeJSONError(w, http.StatusBadRequest, "slot_id, title, image_url and target_url are required")
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

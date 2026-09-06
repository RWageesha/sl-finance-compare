package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
)

func handleAdminDashboard(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		stats, err := database.GetDashboardStats(ctx)
		if err != nil {
			log.Printf("admin: dashboard stats: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load dashboard")
			return
		}
		scraping, err := database.GetScrapingStatus(ctx)
		if err != nil {
			log.Printf("admin: scraping status: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load dashboard")
			return
		}
		alerts, err := database.GetSystemAlerts(ctx)
		if err != nil {
			log.Printf("admin: system alerts: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load dashboard")
			return
		}
		changes, err := database.GetRecentRateChanges(ctx, 10)
		if err != nil {
			log.Printf("admin: recent rate changes: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load dashboard")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"stats":          stats,
			"scraping":       scraping,
			"alerts":         alerts,
			"recent_changes": changes,
		})
	}
}

func handleAdminVerificationQueue(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		rows, err := database.ListVerificationQueue(r.Context(), db.VerificationFilter{
			CategoryGroup: q.Get("category"),
			BankName:      q.Get("bank"),
		})
		if err != nil {
			log.Printf("admin: verification queue: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load verification queue")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func handleAdminRateDecision(database *db.DB, status, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := database.SetRateVerificationStatus(r.Context(), id, status); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "rate not found")
				return
			}
			log.Printf("admin: %s rate: %v", action, err)
			writeJSONError(w, http.StatusInternalServerError, action+" failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, action, "rate #"+strconv.FormatInt(id, 10), "", status)
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	}
}

func handleAdminRatesList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		rows, total, err := database.ListRatesAdmin(r.Context(), db.RateManagementFilter{
			Search:        q.Get("search"),
			BankName:      q.Get("bank"),
			CategoryGroup: q.Get("category"),
			Status:        q.Get("status"),
			Page:          page,
			PerPage:       10,
		})
		if err != nil {
			log.Printf("admin: rates list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load rates")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": total})
	}
}

func handleAdminRateEdit(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct {
			NewRate float64 `json:"new_rate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "new_rate is required")
			return
		}
		newID, err := database.InsertManualRate(r.Context(), id, body.NewRate, actor.ID)
		if err != nil {
			log.Printf("admin: edit rate: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "edit failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Rate Manually Edited", "rate #"+strconv.FormatInt(id, 10), "", strconv.FormatFloat(body.NewRate, 'f', 2, 64))
		writeJSON(w, http.StatusOK, map[string]int64{"id": newID})
	}
}

func handleAdminRateHistory(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.ParseInt(r.URL.Query().Get("product_id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "product_id is required")
			return
		}
		var tenureValue *int
		if v := r.URL.Query().Get("tenure_value"); v != "" {
			n, err := strconv.Atoi(v)
			if err == nil {
				tenureValue = &n
			}
		}
		rows, err := database.GetRateHistoryAdmin(r.Context(), productID, tenureValue, r.URL.Query().Get("tenure_label"), r.URL.Query().Get("rate_label"))
		if err != nil {
			log.Printf("admin: rate history: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load history")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func handleAdminDataSourcesList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		rows, err := database.ListDataSources(r.Context(), db.DataSourceFilter{
			BankName:   q.Get("bank"),
			SourceType: q.Get("type"),
			Status:     q.Get("status"),
		})
		if err != nil {
			log.Printf("admin: data sources: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load data sources")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func handleAdminDataSourceToggle(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct{ Status string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || (body.Status != "active" && body.Status != "disabled") {
			writeJSONError(w, http.StatusBadRequest, "status must be active or disabled")
			return
		}
		if err := database.SetDataSourceStatus(r.Context(), id, body.Status); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		action := "Source Enabled"
		if body.Status == "disabled" {
			action = "Source Disabled"
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, action, "source #"+strconv.FormatInt(id, 10), "", body.Status)
		writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
	}
}

// runScraperAsync shells out to the Python scraper. This genuinely works
// in local development (where the repo, its scraper/.venv, and the API
// process share a filesystem), but this Go binary and the scraper are
// deployed as SEPARATE services in production (see README/workflow
// comments — the live cron runs on Render as its own job) — invoking it
// from here will silently no-op there since scraper/main.py won't exist
// alongside the deployed API binary. Flagged as a known limitation rather
// than pretending this button reaches production infrastructure it can't.
func runScraperAsync() {
	repoRoot, err := filepath.Abs(filepath.Join(".", ".."))
	if err != nil {
		log.Printf("admin: resolve scraper path: %v", err)
		return
	}
	scraperDir := filepath.Join(repoRoot, "scraper")
	pythonBin := filepath.Join(scraperDir, ".venv", "Scripts", "python.exe")
	if _, statErr := os.Stat(pythonBin); statErr != nil {
		pythonBin = filepath.Join(scraperDir, ".venv", "bin", "python")
	}
	cmd := exec.Command(pythonBin, "main.py")
	cmd.Dir = scraperDir
	if err := cmd.Start(); err != nil {
		log.Printf("admin: start scraper: %v (this endpoint only works when the scraper's venv is available alongside the API process, e.g. local dev)", err)
		return
	}
	go func() { _ = cmd.Wait() }()
}

func handleAdminRunAllScrapers(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		go runScraperAsync()
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Scrapers Triggered", "all sources", "", "")
		writeJSON(w, http.StatusAccepted, map[string]string{
			"status": "started",
			"note":   "Runs scraper/main.py in-process if its venv is available alongside this API instance (true in local dev; production runs the scraper as a separate service, so this is a no-op there).",
		})
	}
}

func handleAdminSourceRunManual(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id := r.PathValue("id")
		go runScraperAsync()
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Manual Scrape Triggered", "source #"+id, "", "")
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

func handleAdminScrapingJobs(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runs, err := database.ListScrapeRuns(r.Context(), 50)
		if err != nil {
			log.Printf("admin: scraping jobs: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load scraping jobs")
			return
		}
		var total, success, failed int
		since := time.Now().Add(-24 * time.Hour)
		for _, run := range runs {
			if run.StartedAt.After(since) {
				total++
				switch run.Status {
				case "success", "partial":
					success++
				case "failed":
					failed++
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"data":              runs,
			"total_jobs_today":  total,
			"successful_runs":   success,
			"failed_attempts":   failed,
		})
	}
}

func handleAdminScrapeJobDetail(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		detail, err := database.GetScrapeRunDetail(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func handleAdminBanksList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		banks, err := database.ListBanksAdmin(r.Context())
		if err != nil {
			log.Printf("admin: banks list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load banks")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": banks})
	}
}

func handleAdminBankCreate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		var body struct {
			Name     string `json:"name"`
			Code     string `json:"code"`
			BankType string `json:"bank_type"`
			Website  string `json:"website"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Code == "" {
			writeJSONError(w, http.StatusBadRequest, "name and code are required")
			return
		}
		if body.BankType == "" {
			body.BankType = "Commercial Bank"
		}
		id, err := database.AddBank(r.Context(), body.Name, body.Code, body.BankType, body.Website)
		if errors.Is(err, db.ErrConflict) {
			writeJSONError(w, http.StatusConflict, "a bank with that code already exists")
			return
		}
		if err != nil {
			log.Printf("admin: add bank: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to add bank")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Bank Added", body.Name, "", "")
		writeJSON(w, http.StatusOK, map[string]int64{"id": id})
	}
}

func handleAdminBankUpdate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct {
			Name     string `json:"name"`
			BankType string `json:"bank_type"`
			Website  string `json:"website"`
			Status   string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Status == "" {
			writeJSONError(w, http.StatusBadRequest, "name and status are required")
			return
		}
		if err := database.UpdateBank(r.Context(), id, body.Name, body.BankType, body.Website, body.Status); err != nil {
			log.Printf("admin: update bank: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Bank Updated", body.Name, "", body.Status)
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func handleAdminProductsList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cats, err := database.ListProductCategories(r.Context())
		if err != nil {
			log.Printf("admin: products list: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load products")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": cats})
	}
}

func handleAdminProductBanks(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		rows, err := database.ListBanksForCategory(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to load banks")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows})
	}
}

func handleAdminUserReportsList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		rows, total, err := database.ListUserReports(r.Context(), db.UserReportFilter{
			Status:  q.Get("status"),
			Page:    page,
			PerPage: 20,
		})
		if err != nil {
			log.Printf("admin: user reports: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load reports")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": total})
	}
}

func handleAdminUserReportResolve(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		report, err := database.GetUserReport(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "report not found")
			return
		}
		if err := database.SetUserReportStatus(r.Context(), id, "resolved", actor.ID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "resolve failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Report Resolved", report.BankName+" - "+report.ProductLabel, report.CurrentValue, report.CorrectValue)
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	}
}

func handleAdminUserReportSetStatus(database *db.DB, status, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := database.SetUserReportStatus(r.Context(), id, status, actor.ID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, action+" failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, action, "report #"+strconv.FormatInt(id, 10), "", status)
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	}
}

func handleAdminAuditLogs(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		rows, total, err := database.ListAuditLogs(r.Context(), db.AuditLogFilter{
			AdminUsername: q.Get("admin"),
			Action:        q.Get("action"),
			Page:          page,
			PerPage:       20,
		})
		if err != nil {
			log.Printf("admin: audit logs: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load audit logs")
			return
		}
		admins, _ := database.ListDistinctAuditAdmins(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"data": rows, "total": total, "admins": admins})
	}
}

func handleAdminAuditLogsClear(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := database.ClearAuditLogs(r.Context()); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "clear failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
	}
}

// handlePublicUserReportCreate backs the public site's Report Issue form
// — no auth required, this is how an anonymous visitor's correction
// reaches Admin -> User Reports.
func handlePublicUserReportCreate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			BankName     string `json:"bank_name"`
			ProductLabel string `json:"product_label"`
			IssueType    string `json:"issue_type"`
			CurrentValue string `json:"current_value"`
			CorrectValue string `json:"correct_value"`
			SourceURL    string `json:"source_url"`
			Description  string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if body.BankName == "" || body.ProductLabel == "" || body.CorrectValue == "" || body.SourceURL == "" {
			writeJSONError(w, http.StatusBadRequest, "bank, product, correct value, and source URL are required")
			return
		}
		id, err := database.CreateUserReport(r.Context(), models.UserReport{
			BankName:     body.BankName,
			ProductLabel: body.ProductLabel,
			IssueType:    body.IssueType,
			CurrentValue: body.CurrentValue,
			CorrectValue: body.CorrectValue,
			SourceURL:    body.SourceURL,
			Description:  body.Description,
		})
		if err != nil {
			log.Printf("public: create user report: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to submit report")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"id": id})
	}
}

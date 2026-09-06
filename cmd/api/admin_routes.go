package main

import (
	"net/http"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/adminauth"
	"github.com/wageesha/sl-finance-compare/internal/db"
)

// registerAdminRoutes wires every /api/v1/admin/* route. login is the
// only one reachable without a valid session; everything else goes
// through requireAdmin, and the super-admin-only routes additionally go
// through requireRole("super_admin", ...) so the backend enforces access
// even if a request bypasses the frontend entirely.
func registerAdminRoutes(mux *http.ServeMux, database *db.DB) {
	loginLimiter := adminauth.NewLoginLimiter(5, 15*time.Minute)

	mux.HandleFunc("POST /api/v1/admin/login", handleAdminLogin(database, loginLimiter))
	mux.HandleFunc("POST /api/v1/admin/logout", handleAdminLogout(database))

	admin := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, h) }
	superOnly := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, requireRole("super_admin", h)) }
	// editor and above may act on verification/reports; admin and above may
	// change structural things (banks/products/data-sources/rate edits).
	editorUp := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, requireRole("editor", h)) }
	adminUp := func(h http.HandlerFunc) http.HandlerFunc { return requireAdmin(database, requireRole("admin", h)) }

	mux.HandleFunc("GET /api/v1/admin/me", admin(handleAdminMe))
	mux.HandleFunc("PATCH /api/v1/admin/me", admin(handleAdminUpdateUsername(database)))
	mux.HandleFunc("PATCH /api/v1/admin/me/password", admin(handleAdminUpdatePassword(database)))

	mux.HandleFunc("GET /api/v1/admin/dashboard", admin(handleAdminDashboard(database)))

	mux.HandleFunc("GET /api/v1/admin/verification-queue", admin(handleAdminVerificationQueue(database)))
	mux.HandleFunc("POST /api/v1/admin/rates/{id}/approve", editorUp(handleAdminRateDecision(database, "verified", "Rate Approved")))
	mux.HandleFunc("POST /api/v1/admin/rates/{id}/reject", editorUp(handleAdminRateDecision(database, "rejected", "Rate Rejected")))

	mux.HandleFunc("GET /api/v1/admin/rates", admin(handleAdminRatesList(database)))
	mux.HandleFunc("PATCH /api/v1/admin/rates/{id}", adminUp(handleAdminRateEdit(database)))
	mux.HandleFunc("GET /api/v1/admin/rates/history", admin(handleAdminRateHistory(database)))

	mux.HandleFunc("GET /api/v1/admin/data-sources", admin(handleAdminDataSourcesList(database)))
	mux.HandleFunc("PATCH /api/v1/admin/data-sources/{id}", adminUp(handleAdminDataSourceToggle(database)))
	mux.HandleFunc("POST /api/v1/admin/data-sources/{id}/run", adminUp(handleAdminSourceRunManual(database)))

	mux.HandleFunc("GET /api/v1/admin/scraping-jobs", admin(handleAdminScrapingJobs(database)))
	mux.HandleFunc("GET /api/v1/admin/scraping-jobs/{id}", admin(handleAdminScrapeJobDetail(database)))
	mux.HandleFunc("POST /api/v1/admin/scrapers/run-all", adminUp(handleAdminRunAllScrapers(database)))

	mux.HandleFunc("GET /api/v1/admin/banks", admin(handleAdminBanksList(database)))
	mux.HandleFunc("POST /api/v1/admin/banks", adminUp(handleAdminBankCreate(database)))
	mux.HandleFunc("PATCH /api/v1/admin/banks/{id}", adminUp(handleAdminBankUpdate(database)))

	mux.HandleFunc("GET /api/v1/admin/products", admin(handleAdminProductsList(database)))
	mux.HandleFunc("GET /api/v1/admin/products/{id}/banks", admin(handleAdminProductBanks(database)))

	mux.HandleFunc("GET /api/v1/admin/user-reports", admin(handleAdminUserReportsList(database)))
	mux.HandleFunc("POST /api/v1/admin/user-reports/{id}/resolve", editorUp(handleAdminUserReportResolve(database)))
	mux.HandleFunc("POST /api/v1/admin/user-reports/{id}/reject", editorUp(handleAdminUserReportSetStatus(database, "rejected", "Report Rejected")))
	mux.HandleFunc("POST /api/v1/admin/user-reports/{id}/request-info", editorUp(handleAdminUserReportSetStatus(database, "pending", "Report Info Requested")))

	mux.HandleFunc("GET /api/v1/admin/audit-logs", admin(handleAdminAuditLogs(database)))
	mux.HandleFunc("DELETE /api/v1/admin/audit-logs", adminUp(handleAdminAuditLogsClear(database)))

	mux.HandleFunc("GET /api/v1/admin/team", superOnly(handleAdminTeamList(database)))
	mux.HandleFunc("POST /api/v1/admin/team", superOnly(handleAdminTeamCreate(database)))
	mux.HandleFunc("PATCH /api/v1/admin/team/{id}", superOnly(handleAdminTeamUpdate(database)))
	mux.HandleFunc("PATCH /api/v1/admin/team/{id}/disable", superOnly(handleAdminTeamSetStatus(database, "disabled", "Team Member Disabled")))
	mux.HandleFunc("PATCH /api/v1/admin/team/{id}/enable", superOnly(handleAdminTeamSetStatus(database, "active", "Team Member Enabled")))
	mux.HandleFunc("POST /api/v1/admin/team/{id}/reset-password", superOnly(handleAdminTeamResetPassword(database)))
}

package main

import (
	"context"
	"net/http"

	"github.com/wageesha/sl-finance-compare/internal/adminauth"
	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
)

type ctxKey int

const ctxKeyAdmin ctxKey = 1

// requireAdmin validates the session cookie against admin_sessions
// (checking expiry and that the account is still active) before calling
// next — every /api/v1/admin/* route except login goes through this.
func requireAdmin(database *db.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(adminauth.SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJSONError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		admin, err := database.GetSessionAdmin(r.Context(), adminauth.HashToken(cookie.Value))
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "session expired or invalid")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAdmin, admin)
		next(w, r.WithContext(ctx))
	}
}

// requireRole wraps a handler that requireAdmin has already run in front
// of, rejecting with 403 if the session's role doesn't meet minRole. This
// is enforced server-side regardless of what the frontend shows/hides.
func requireRole(minRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := adminFromContext(r)
		if admin == nil || !adminauth.RoleAtLeast(admin.Role, minRole) {
			writeJSONError(w, http.StatusForbidden, "insufficient role")
			return
		}
		next(w, r)
	}
}

func adminFromContext(r *http.Request) *models.AdminUser {
	admin, _ := r.Context().Value(ctxKeyAdmin).(*models.AdminUser)
	return admin
}

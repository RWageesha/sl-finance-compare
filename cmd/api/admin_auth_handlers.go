package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/adminauth"
	"github.com/wageesha/sl-finance-compare/internal/db"
)

// clientIP extracts a best-effort client IP for the login rate limiter.
// Render (and most PaaS) sit behind a proxy that sets X-Forwarded-For;
// fall back to RemoteAddr for local dev.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func handleAdminLogin(database *db.DB, limiter *adminauth.LoginLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !limiter.Allow(ip) {
			writeJSONError(w, http.StatusTooManyRequests, "too many login attempts — try again later")
			return
		}

		var body struct{ Username, Password string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		admin, err := database.GetAdminByUsername(r.Context(), body.Username)
		if errors.Is(err, db.ErrNotFound) {
			writeJSONError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		if err != nil {
			log.Printf("admin: login lookup: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "login failed")
			return
		}
		if admin.Status != "active" {
			writeJSONError(w, http.StatusUnauthorized, "this account has been disabled")
			return
		}
		if !adminauth.CheckPassword(admin.PasswordHash, body.Password) {
			writeJSONError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}

		token, tokenHash, err := adminauth.NewSessionToken()
		if err != nil {
			log.Printf("admin: create session token: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "login failed")
			return
		}
		expiresAt := time.Now().Add(adminauth.SessionDuration)
		if err := database.CreateSession(r.Context(), admin.ID, tokenHash, expiresAt); err != nil {
			log.Printf("admin: create session: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "login failed")
			return
		}
		_ = database.UpdateAdminLastLogin(r.Context(), admin.ID)

		http.SetCookie(w, &http.Cookie{
			Name:     adminauth.SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			Expires:  expiresAt,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"username": admin.Username,
			"role":     admin.Role,
		})
	}
}

func handleAdminLogout(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(adminauth.SessionCookieName); err == nil && cookie.Value != "" {
			_ = database.DeleteSession(r.Context(), adminauth.HashToken(cookie.Value))
		}
		http.SetCookie(w, &http.Cookie{
			Name:     adminauth.SessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			SameSite: http.SameSiteStrictMode,
		})
		writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
	}
}

func handleAdminMe(w http.ResponseWriter, r *http.Request) {
	admin := adminFromContext(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       admin.ID,
		"username": admin.Username,
		"email":    admin.Email,
		"role":     admin.Role,
	})
}

func handleAdminUpdateUsername(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := adminFromContext(r)
		var body struct{ Username string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" {
			writeJSONError(w, http.StatusBadRequest, "username is required")
			return
		}
		err := database.UpdateAdminUsername(r.Context(), admin.ID, body.Username)
		if errors.Is(err, db.ErrConflict) {
			writeJSONError(w, http.StatusConflict, "that username is already taken")
			return
		}
		if err != nil {
			log.Printf("admin: update username: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"username": body.Username})
	}
}

func handleAdminUpdatePassword(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := adminFromContext(r)
		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if len(body.NewPassword) < 8 {
			writeJSONError(w, http.StatusBadRequest, "new password must be at least 8 characters")
			return
		}
		full, err := database.GetAdminByID(r.Context(), admin.ID)
		if err != nil || !adminauth.CheckPassword(full.PasswordHash, body.CurrentPassword) {
			writeJSONError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		hash, err := adminauth.HashPassword(body.NewPassword)
		if err != nil {
			log.Printf("admin: hash new password: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		if err := database.UpdateAdminPassword(r.Context(), admin.ID, hash); err != nil {
			log.Printf("admin: update password: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
	}
}

// ===== Team management (super_admin only — gated by requireRole in main.go) =====

func handleAdminTeamList(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := database.ListAdminUsers(r.Context())
		if err != nil {
			log.Printf("admin: list team: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load team members")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": users})
	}
}

func handleAdminTeamCreate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		var body struct{ Username, Email, Password, Role string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Email == "" || body.Password == "" {
			writeJSONError(w, http.StatusBadRequest, "username, email, and password are required")
			return
		}
		if body.Role != "admin" && body.Role != "editor" && body.Role != "viewer" {
			// super_admin is deliberately not assignable through this UI —
			// only by editing the database directly, to avoid an accidental
			// privilege-escalation bug in this form.
			writeJSONError(w, http.StatusBadRequest, "role must be admin, editor, or viewer")
			return
		}
		hash, err := adminauth.HashPassword(body.Password)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create member")
			return
		}
		id, err := database.CreateAdminUser(r.Context(), body.Username, body.Email, hash, body.Role)
		if errors.Is(err, db.ErrConflict) {
			writeJSONError(w, http.StatusConflict, "that username or email is already in use")
			return
		}
		if err != nil {
			log.Printf("admin: create team member: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create member")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Team Member Added", body.Username, "", body.Role)
		writeJSON(w, http.StatusOK, map[string]int64{"id": id})
	}
}

func handleAdminTeamUpdate(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct{ Email, Role string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Role == "" {
			writeJSONError(w, http.StatusBadRequest, "email and role are required")
			return
		}
		if body.Role != "admin" && body.Role != "editor" && body.Role != "viewer" {
			writeJSONError(w, http.StatusBadRequest, "role must be admin, editor, or viewer")
			return
		}
		if err := database.UpdateAdminRoleEmail(r.Context(), id, body.Email, body.Role); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "member not found")
				return
			}
			if errors.Is(err, db.ErrConflict) {
				writeJSONError(w, http.StatusConflict, "that email is already in use")
				return
			}
			log.Printf("admin: update team member: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Team Member Updated", body.Email, "", body.Role)
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func handleAdminTeamSetStatus(database *db.DB, status, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := database.SetAdminStatus(r.Context(), id, status); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "member not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "update failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, action, strconv.FormatInt(id, 10), "", status)
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	}
}

func handleAdminTeamResetPassword(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := adminFromContext(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var body struct {
			NewPassword string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.NewPassword) < 8 {
			writeJSONError(w, http.StatusBadRequest, "a new password of at least 8 characters is required")
			return
		}
		hash, err := adminauth.HashPassword(body.NewPassword)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "reset failed")
			return
		}
		if err := database.UpdateAdminPassword(r.Context(), id, hash); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "reset failed")
			return
		}
		_ = database.InsertAuditLog(r.Context(), &actor.ID, actor.Username, "Password Reset", strconv.FormatInt(id, 10), "", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "password reset"})
	}
}

// Package adminauth holds the password hashing, session token, and login
// rate-limiting primitives for the admin panel. Kept separate from
// internal/db so the crypto/security logic is easy to review in one
// place, independent of any particular storage backend.
package adminauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost matches the spec's "at least 12" requirement.
const BcryptCost = 12

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("adminauth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether plain matches hash, never returning the
// underlying bcrypt error detail (callers only need yes/no).
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// NewSessionToken returns a cryptographically random token (to hand to
// the browser as a cookie) and the SHA-256 hash of it (to store in
// admin_sessions — the raw token is never persisted, mirroring how
// passwords are never stored in plain text).
func NewSessionToken() (token string, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("adminauth: generate session token: %w", err)
	}
	token = hex.EncodeToString(buf)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

const SessionCookieName = "admin_session"
const SessionDuration = 12 * time.Hour

// LoginLimiter is a simple in-memory sliding-window limiter keyed by
// client IP — this runs as a single instance (see README's deployment
// notes), so an in-memory map is sufficient without a shared store.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{attempts: make(map[string][]time.Time), max: max, window: window}
}

// Allow records an attempt for key and reports whether it's within the
// limit. Call it once per login attempt (success or failure) — failures
// are what you're guarding against, but counting every attempt keeps a
// script that alternates usernames from resetting its own budget.
func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

// Role hierarchy: higher index = more privilege. RoleAtLeast reports
// whether `have` meets or exceeds `need`.
var roleRank = map[string]int{
	"viewer":      0,
	"editor":      1,
	"admin":       2,
	"super_admin": 3,
}

func RoleAtLeast(have, need string) bool {
	return roleRank[have] >= roleRank[need]
}

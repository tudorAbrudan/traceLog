package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// Rate limiter for login attempts: max failures per minute, then 15-minute lockout for that IP.
type loginRateLimiter struct {
	mu        sync.Mutex
	attempts  map[string][]time.Time
	lockUntil map[string]time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts:  make(map[string][]time.Time),
		lockUntil: make(map[string]time.Time),
	}
}

const (
	maxLoginAttempts  = 5
	loginWindowMinute = time.Minute
	lockoutDuration   = 15 * time.Minute
)

func (rl *loginRateLimiter) isBlocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if until, ok := rl.lockUntil[ip]; ok {
		if now.Before(until) {
			return true
		}
		delete(rl.lockUntil, ip)
	}

	attempts := rl.pruneAttemptsLocked(ip, now)
	var recentMinute int
	for _, t := range attempts {
		if now.Sub(t) < loginWindowMinute {
			recentMinute++
		}
	}
	return recentMinute >= maxLoginAttempts
}

func (rl *loginRateLimiter) pruneAttemptsLocked(ip string, now time.Time) []time.Time {
	var recent []time.Time
	for _, t := range rl.attempts[ip] {
		if now.Sub(t) < lockoutDuration {
			recent = append(recent, t)
		}
	}
	rl.attempts[ip] = recent
	return recent
}

func (rl *loginRateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.attempts[ip] = append(rl.attempts[ip], now)
	attempts := rl.pruneAttemptsLocked(ip, now)

	var inMinute int
	for _, t := range attempts {
		if now.Sub(t) < loginWindowMinute {
			inMinute++
		}
	}
	if inMinute >= maxLoginAttempts {
		rl.lockUntil[ip] = now.Add(lockoutDuration)
	}
}

func (rl *loginRateLimiter) reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
	delete(rl.lockUntil, ip)
}

func (h *Hub) cookiePath() string {
	p := models.NormalizeURLPathPrefix(h.cfg.URLPathPrefix)
	if p == "" {
		return "/"
	}
	return p
}

// Auth middleware

func (h *Hub) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("tracelog_session")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
			return
		}

		user, err := h.store.GetUserBySession(r.Context(), cookie.Value)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "tracelog_session",
				Value:    "",
				Path:     h.cookiePath(),
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Session expired"})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// CSRF protection - validate token on mutating requests

func (h *Hub) csrfMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		csrfHeader := r.Header.Get("X-CSRF-Token")
		cookie, err := r.Cookie("tracelog_csrf")
		if err != nil || csrfHeader == "" || csrfHeader != cookie.Value {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Invalid CSRF token"})
			return
		}

		next(w, r)
	}
}

// Login handler

func (h *Hub) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := extractIP(r)

	if h.rateLimiter.isBlocked(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": fmt.Sprintf("Too many failed login attempts. Wait up to %d minutes before trying again.", int(lockoutDuration.Minutes())),
		})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		h.rateLimiter.recordFailure(ip)
		slog.Warn("Login failed: user not found", "username", req.Username, "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.rateLimiter.recordFailure(ip)
		slog.Warn("Login failed: wrong password", "username", req.Username, "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
		return
	}

	h.rateLimiter.reset(ip)

	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	if err := h.store.CreateSession(r.Context(), token, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	csrfToken, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_session",
		Value:    token,
		Path:     h.cookiePath(),
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_csrf",
		Value:    csrfToken,
		Path:     h.cookiePath(),
		Expires:  expiresAt,
		SameSite: http.SameSiteStrictMode,
	})

	slog.Info("User logged in", "username", user.Username, "ip", ip)

	writeJSON(w, http.StatusOK, map[string]any{
		"user":       user,
		"csrf_token": csrfToken,
	})
}

func (h *Hub) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("tracelog_session")
	if err == nil {
		if err := h.store.DeleteSession(r.Context(), cookie.Value); err != nil {
			slog.Debug("logout delete session", "error", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_session",
		Value:    "",
		Path:     h.cookiePath(),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_csrf",
		Value:    "",
		Path:     h.cookiePath(),
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *Hub) handleMe(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
		return
	}
	csrfCookie, _ := r.Cookie("tracelog_csrf")
	csrfToken := ""
	if csrfCookie != nil {
		csrfToken = csrfCookie.Value
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       user,
		"csrf_token": csrfToken,
	})
}

func (h *Hub) handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, _ := h.store.UserCount(ctx)
	if count > 0 {
		writeError(w, http.StatusForbidden, "Setup already completed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	user, err := h.store.CreateUser(ctx, req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user: %v", err)
		return
	}

	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := h.store.CreateSession(ctx, token, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session: %v", err)
		return
	}

	csrfToken, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_session",
		Value:    token,
		Path:     h.cookiePath(),
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_csrf",
		Value:    csrfToken,
		Path:     h.cookiePath(),
		Expires:  expiresAt,
		SameSite: http.SameSiteStrictMode,
	})

	slog.Info("Setup completed, first user created", "username", req.Username)

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":       user,
		"csrf_token": csrfToken,
	})
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Real-IP"); xff != "" {
		return xff
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

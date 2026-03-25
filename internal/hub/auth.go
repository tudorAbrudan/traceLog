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
)

type contextKey string

const userContextKey contextKey = "user"

// Rate limiter for login attempts
type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts: make(map[string][]time.Time),
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
	attempts := rl.attempts[ip]

	// Clean old attempts
	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < lockoutDuration {
			recent = append(recent, t)
		}
	}
	rl.attempts[ip] = recent

	// Count attempts in the last minute
	var recentMinute int
	for _, t := range recent {
		if now.Sub(t) < loginWindowMinute {
			recentMinute++
		}
	}

	return recentMinute >= maxLoginAttempts
}

func (rl *loginRateLimiter) record(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts[ip] = append(rl.attempts[ip], time.Now())
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
				Path:     "/",
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
			"error": fmt.Sprintf("Too many login attempts. Try again in %d minutes.", int(lockoutDuration.Minutes())),
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
		h.rateLimiter.record(ip)
		slog.Warn("Login failed: user not found", "username", req.Username, "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.rateLimiter.record(ip)
		slog.Warn("Login failed: wrong password", "username", req.Username, "ip", ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
		return
	}

	token := generateToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	if err := h.store.CreateSession(r.Context(), token, user.ID, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	csrfToken := generateToken()

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_csrf",
		Value:    csrfToken,
		Path:     "/",
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
		h.store.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "tracelog_csrf",
		Value:    "",
		Path:     "/",
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
	writeJSON(w, http.StatusOK, user)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
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

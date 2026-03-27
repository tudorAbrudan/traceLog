package hub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tudorAbrudan/tracelog/internal/agent/detect"
	"github.com/tudorAbrudan/tracelog/internal/models"
)

func (h *Hub) handleDatabaseExport(w http.ResponseWriter, r *http.Request) {
	u, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok || u == nil || u.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}

	full, err := h.store.GetUserByID(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load user: %v", err)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(full.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	f, err := os.CreateTemp("", "tracelog-backup-*.db")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create temp file: %v", err)
		return
	}
	tmpPath := f.Name()
	_ = f.Close()
	defer os.Remove(tmpPath)

	if err := h.store.Backup(r.Context(), tmpPath); err != nil {
		writeError(w, http.StatusInternalServerError, "Backup failed: %v", err)
		return
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Read backup failed: %v", err)
		return
	}

	fname := fmt.Sprintf("tracelog-backup-%s.db", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fname))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Hub) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userCount, _ := h.store.UserCount(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    h.cfg.Version,
		"uptime":     time.Now().Unix(),
		"setup_done": userCount > 0,
	})
}

func (h *Hub) handleDetect(w http.ResponseWriter, r *http.Request) {
	d := detect.Run()
	writeJSON(w, http.StatusOK, d)
}

func (h *Hub) handleDashboard(w http.ResponseWriter, r *http.Request) {
	h.dashboardSPA().ServeHTTP(w, r)
}

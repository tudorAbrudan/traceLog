package hub

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tudorAbrudan/tracelog/internal/hub/notify"
	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

func (h *Hub) handleListNotificationChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.store.ListNotificationChannels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list notification channels")
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

func (h *Hub) handleCreateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var ch notify.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if ch.Name == "" || ch.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if err := h.store.CreateNotificationChannel(r.Context(), &ch); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create notification channel: %v", err)
		return
	}
	h.notify.AddChannel(&ch)
	writeJSON(w, http.StatusCreated, ch)
}

func (h *Hub) handleUpdateNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var ch notify.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	ch.ID = id
	if ch.Name == "" || ch.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if err := h.store.UpdateNotificationChannel(r.Context(), &ch); err != nil {
		if errors.Is(err, store.ErrNotificationChannelNotFound) {
			writeError(w, http.StatusNotFound, "Notification channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update notification channel: %v", err)
		return
	}
	h.notify.AddChannel(&ch)
	writeJSON(w, http.StatusOK, ch)
}

func (h *Hub) handleDeleteNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.notify.RemoveChannel(id)
	if err := h.store.DeleteNotificationChannel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete notification channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Hub) handleTestNotificationChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := h.notify.Send(r.Context(), id, "TraceLog Test", "This is a test notification from TraceLog.")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Test failed: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

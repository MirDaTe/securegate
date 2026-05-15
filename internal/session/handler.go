package session

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mirdate/securegate/internal/auth"
)

// Handler — 세션 HTTP 핸들러
type Handler struct {
	mgr *Manager
}

func NewHandler(mgr *Manager) *Handler {
	return &Handler{mgr: mgr}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/sessions", h.CreateSession)
	r.Delete("/sessions/{id}", h.EndSession)
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식")
		return
	}

	clientIP := r.Header.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	resp, err := h.mgr.CreateSession(r.Context(), userID, req.HostID, clientIP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) EndSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 ID")
		return
	}
	h.mgr.EndSession(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "세션이 종료되었습니다"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

var _ = auth.GetUserID

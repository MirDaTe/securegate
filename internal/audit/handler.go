package audit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mirdate/securegate/internal/auth"
)

type Handler struct {
	logger *Logger
}

func NewHandler(logger *Logger) *Handler { return &Handler{logger: logger} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/audit/logs", h.ListLogs)
	r.Get("/audit/verify", h.VerifyChain)
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 { limit = 100 }

	logs, err := h.logger.QueryLogs(r.Context(), action, limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, logs)
}

func (h *Handler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 { limit = 100 }

	valid, count, err := h.logger.VerifyChain(r.Context(), limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"valid": valid,
		"count": count,
	})
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

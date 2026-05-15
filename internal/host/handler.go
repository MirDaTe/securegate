package host

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mirdate/securegate/internal/auth"
)

// Handler — 호스트 관리 HTTP 핸들러
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/hosts", h.ListHosts)
	r.Post("/hosts", h.CreateHost)
	r.Get("/hosts/{id}", h.GetHost)
	r.Put("/hosts/{id}", h.UpdateHost)
	r.Delete("/hosts/{id}", h.DeleteHost)
}

func (h *Handler) CreateHost(w http.ResponseWriter, r *http.Request) {
	var req CreateHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식")
		return
	}
	if req.Name == "" || req.Hostname == "" || req.Protocol == "" || req.Port == 0 {
		writeError(w, http.StatusBadRequest, "이름, 호스트명, 프로토콜, 포트는 필수입니다")
		return
	}
	host, err := h.svc.CreateHost(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, host)
}

func (h *Handler) GetHost(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 ID")
		return
	}
	host, err := h.svc.GetHost(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, host)
}

func (h *Handler) ListHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := h.svc.ListHosts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hosts)
}

func (h *Handler) UpdateHost(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 ID")
		return
	}
	var req UpdateHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식")
		return
	}
	host, err := h.svc.UpdateHost(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, host)
}

func (h *Handler) DeleteHost(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 ID")
		return
	}
	if err := h.svc.DeleteHost(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "삭제되었습니다"})
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

package policy

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/policies", h.List)
	r.Post("/policies", h.Create)
	r.Put("/policies/{id}", h.Update)
	r.Delete("/policies/{id}", h.Delete)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var p Policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, 400, "잘못된 요청 형식")
		return
	}
	if err := h.svc.Create(r.Context(), &p); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	policies, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, policies)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	var p Policy
	json.NewDecoder(r.Body).Decode(&p)
	if err := h.svc.Update(r.Context(), id, &p); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"message": "수정되었습니다"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	h.svc.Delete(r.Context(), id)
	writeJSON(w, 200, map[string]string{"message": "삭제되었습니다"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler — 인증 HTTP 핸들러
type Handler struct {
	svc *Service
}

// NewHandler — Handler 생성자
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterPublicRoutes — 인증 불필요 라우트 (로그인, 회원가입, 토큰 갱신)
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/signup", h.Signup)
	r.Post("/auth/refresh", h.RefreshToken)
}

// RegisterProtectedRoutes — 인증 필요 라우트 (로그아웃, 비밀번호 변경, MFA)
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/auth/logout", h.Logout)
	r.Post("/auth/password/change", h.ChangePassword)
	r.Post("/auth/mfa/setup", h.SetupMFA)
	r.Post("/auth/mfa/verify", h.VerifyMFA)
	r.Post("/auth/mfa/enable", h.EnableMFA)
	r.Post("/auth/mfa/disable", h.DisableMFA)
}

// Login — POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식입니다")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "아이디와 비밀번호를 입력해주세요")
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Signup — POST /api/auth/signup
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식입니다")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "아이디와 비밀번호를 입력해주세요")
		return
	}

	if len(req.Password) < 10 {
		writeError(w, http.StatusBadRequest, "비밀번호는 최소 10자 이상이어야 합니다")
		return
	}

	user, err := h.svc.Signup(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// Logout — POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// refresh token을 요청 바디에서 추출
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token이 필요합니다")
		return
	}

	if err := h.svc.Logout(r.Context(), body.RefreshToken); err != nil {
		writeError(w, http.StatusInternalServerError, "로그아웃 처리 실패")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "로그아웃되었습니다"})
}

// ChangePassword — POST /api/auth/password/change
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식입니다")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "비밀번호가 변경되었습니다"})
}

// VerifyMFA — POST /api/auth/mfa/verify
func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "잘못된 요청 형식입니다")
		return
	}

	resp, err := h.svc.VerifyMFA(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// SetupMFA — POST /api/auth/mfa/setup
func (h *Handler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
		return
	}

	resp, err := h.svc.SetupMFA(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// EnableMFA — POST /api/auth/mfa/enable
func (h *Handler) EnableMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
		return
	}

	if err := h.svc.EnableMFA(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "MFA가 활성화되었습니다"})
}

// DisableMFA — POST /api/auth/mfa/disable
func (h *Handler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
		return
	}

	if err := h.svc.DisableMFA(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "MFA가 비활성화되었습니다"})
}

// RefreshToken — POST /api/auth/refresh (Access Token 갱신)
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token이 필요합니다")
		return
	}

	// 블랙리스트 확인
	if h.svc.IsTokenBlacklisted(r.Context(), body.RefreshToken) {
		writeError(w, http.StatusUnauthorized, "만료된 refresh token입니다")
		return
	}

	claims, err := ValidateToken(body.RefreshToken, h.svc.cfg.Secret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "유효하지 않은 refresh token입니다")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "잘못된 토큰 정보")
		return
	}

	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "사용자를 찾을 수 없습니다")
		return
	}

	// 기존 refresh token → 블랙리스트 (rotation)
	h.svc.Logout(r.Context(), body.RefreshToken)

	// 새 토큰 발급
	resp, err := h.svc.issueTokens(r.Context(), *user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "토큰 갱신 실패")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─── 헬퍼 ───

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

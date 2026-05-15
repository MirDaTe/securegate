package auth

import (
	"time"

	"github.com/google/uuid"
)

// User — 사용자 모델
type User struct {
	ID                 uuid.UUID  `json:"id"`
	Username           string     `json:"username"`
	Email              *string    `json:"email,omitempty"`
	PasswordHash       string     `json:"-"` // 절대 JSON 노출 금지
	MFASecret          *string    `json:"-"` // 절대 JSON 노출 금지
	MFAEnabled         bool       `json:"mfa_enabled"`
	Role               string     `json:"role"` // super_admin, admin, auditor, user
	Status             string     `json:"status"`
	MustChangePassword bool       `json:"must_change_password"`
	PasswordChangedAt  *time.Time `json:"password_changed_at,omitempty"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	LoginFailCount     int        `json:"-"`
	LockedUntil        *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// LoginRequest — 로그인 요청
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse — 로그인 응답
type LoginResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RequireMFA   bool   `json:"require_mfa,omitempty"`
	RequirePasswordChange bool `json:"require_password_change,omitempty"`
}

// SignupRequest — 회원가입 요청
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ChangePasswordRequest — 비밀번호 변경 요청
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// MFAVerifyRequest — MFA 인증 요청
type MFAVerifyRequest struct {
	SessionToken string `json:"session_token"` // MFA 대기 중인 임시 토큰
	Code         string `json:"code"`
}

// MFASetupResponse — MFA 설정 정보
type MFASetupResponse struct {
	Secret     string `json:"secret"`
	QRCodeURL  string `json:"qr_code_url"`
}

// TokenClaims — JWT claims
type TokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// UserGroup — 사용자 그룹
type UserGroup struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

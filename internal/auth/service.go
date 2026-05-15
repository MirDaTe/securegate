package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mirdate/securegate/internal/db"
)

// Service — 인증 비즈니스 로직
type Service struct {
	pool  *pgxpool.Pool
	redis *redis.Client
	cfg   JWTConfig
}

// NewService — Service 생성자
func NewService(cfg JWTConfig) *Service {
	return &Service{
		pool:  db.Pool(),
		redis: db.Redis(),
		cfg:   cfg,
	}
}

// Login — 로그인 처리 (성공 시 JWT 발급)
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.findUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("아이디 또는 비밀번호가 올바르지 않습니다")
	}

	// 계정 상태 확인
	switch user.Status {
	case "locked":
		if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
			return nil, fmt.Errorf("계정이 잠겼습니다. %s까지 로그인할 수 없습니다", user.LockedUntil.Format("15:04"))
		}
		// 잠금 해제 시간 지남 → 상태 복구
		s.unlockAccount(ctx, user.ID)
	case "pending":
		return nil, fmt.Errorf("관리자 승인 대기 중인 계정입니다")
	case "disabled":
		return nil, fmt.Errorf("비활성화된 계정입니다")
	}

	// 비밀번호 검증
	valid, err := VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		if err := s.incrementLoginFail(ctx, user); err != nil {
			return nil, fmt.Errorf("로그인 실패 처리 중 오류")
		}
		return nil, fmt.Errorf("아이디 또는 비밀번호가 올바르지 않습니다")
	}

	// 로그인 실패 카운트 초기화
	s.resetLoginFail(ctx, user.ID)

	// MFA가 활성화되어 있으면 MFA 검증 필요 플래그 반환
	if user.MFAEnabled {
		sessionToken, err := s.createMFASession(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("MFA 세션 생성 실패: %w", err)
		}
		return &LoginResponse{
			User:       *user,
			RequireMFA: true,
			AccessToken: sessionToken, // MFA 임시 토큰
		}, nil
	}

	// 비밀번호 강제 변경이 필요한 경우
	if user.MustChangePassword {
		token, err := GenerateAccessToken(s.cfg, TokenClaims{
			UserID:   user.ID.String(),
			Username: user.Username,
			Role:     user.Role,
		})
		if err != nil {
			return nil, fmt.Errorf("토큰 생성 실패: %w", err)
		}
		return &LoginResponse{
			User:                   *user,
			RequirePasswordChange:  true,
			AccessToken:            token,
		}, nil
	}

	// 정상 로그인: JWT 발급
	return s.issueTokens(ctx, *user)
}

// Signup — 회원가입 처리 (셀프 가입 모드)
func (s *Service) Signup(ctx context.Context, req SignupRequest) (*User, error) {
	// 사용자명 중복 확인
	exists, err := s.usernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("이미 사용 중인 아이디입니다")
	}

	// 이메일 중복 확인
	if req.Email != "" {
		exists, err := s.emailExists(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("이미 사용 중인 이메일입니다")
		}
	}

	// 비밀번호 해싱
	hashed, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("비밀번호 해싱 실패: %w", err)
	}

	user := &User{
		ID:                 uuid.New(),
		Username:           req.Username,
		PasswordHash:       hashed,
		Role:               "user",
		Status:             "pending", // 관리자 승인 대기
		MustChangePassword: false,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if req.Email != "" {
		user.Email = &req.Email
	}

	err = s.createUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateAdmin — 초기 관리자 계정 생성
func (s *Service) CreateAdmin(ctx context.Context, password string) error {
	exists, err := s.usernameExists(ctx, "admin")
	if err != nil {
		return err
	}
	if exists {
		return nil // 이미 존재하면 무시
	}

	hashed, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("관리자 비밀번호 해싱 실패: %w", err)
	}

	admin := &User{
		ID:                 uuid.New(),
		Username:           "admin",
		PasswordHash:       hashed,
		Role:               "super_admin",
		Status:             "active",
		MustChangePassword: true, // 최초 로그인 시 강제 변경
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	return s.createUser(ctx, admin)
}

// ChangePassword — 비밀번호 변경
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error {
	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("사용자를 찾을 수 없습니다")
	}

	valid, err := VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if err != nil || !valid {
		return fmt.Errorf("현재 비밀번호가 올바르지 않습니다")
	}

	hashed, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("비밀번호 해싱 실패: %w", err)
	}

	now := time.Now()
	_, err = s.pool.Exec(ctx,
		`UPDATE users SET password_hash=$1, must_change_password=false, password_changed_at=$2, updated_at=$3 WHERE id=$4`,
		hashed, now, now, userID,
	)
	if err != nil {
		return fmt.Errorf("비밀번호 변경 실패: %w", err)
	}

	return nil
}

// VerifyMFA — MFA 코드 검증 후 최종 토큰 발급
func (s *Service) VerifyMFA(ctx context.Context, req MFAVerifyRequest) (*LoginResponse, error) {
	// Redis에서 MFA 세션 조회
	userIDStr, err := s.redis.GetDel(ctx, "mfa:"+req.SessionToken).Result()
	if err != nil {
		return nil, fmt.Errorf("만료되었거나 유효하지 않은 MFA 세션입니다")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("잘못된 사용자 ID")
	}

	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다")
	}

	if user.MFASecret == nil {
		return nil, fmt.Errorf("MFA가 설정되지 않은 계정입니다")
	}

	if !VerifyTOTP(*user.MFASecret, req.Code) {
		return nil, fmt.Errorf("잘못된 인증 코드입니다")
	}

	return s.issueTokens(ctx, *user)
}

// SetupMFA — MFA 설정 (TOTP 시크릿 생성)
func (s *Service) SetupMFA(ctx context.Context, userID uuid.UUID) (*MFASetupResponse, error) {
	user, err := s.findUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다")
	}

	secret := GenerateTOTPSecret()
	qrURL := GetTOTPQRCodeURL(secret, user.Username)

	_, err = s.pool.Exec(ctx,
		`UPDATE users SET mfa_secret=$1, updated_at=$2 WHERE id=$3`,
		secret, time.Now(), userID,
	)
	if err != nil {
		return nil, fmt.Errorf("MFA 설정 실패: %w", err)
	}

	return &MFASetupResponse{
		Secret:    secret,
		QRCodeURL: qrURL,
	}, nil
}

// EnableMFA — MFA 활성화
func (s *Service) EnableMFA(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET mfa_enabled=true, updated_at=$1 WHERE id=$2`,
		time.Now(), userID,
	)
	return err
}

// DisableMFA — MFA 비활성화
func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET mfa_enabled=false, mfa_secret=NULL, updated_at=$1 WHERE id=$2`,
		time.Now(), userID,
	)
	return err
}

// GetUser — 사용자 정보 조회
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	return s.findUserByID(ctx, userID)
}

// Logout — 로그아웃 처리 (refresh token 블랙리스트)
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	// Refresh token을 Blacklist에 추가 (Redis, 7일 TTL)
	return s.redis.Set(ctx, "bl:"+refreshToken, "1", 7*24*time.Hour).Err()
}

// IsTokenBlacklisted — 토큰이 블랙리스트에 있는지 확인
func (s *Service) IsTokenBlacklisted(ctx context.Context, token string) bool {
	exists, _ := s.redis.Exists(ctx, "bl:"+token).Result()
	return exists > 0
}

// ─── 내부 헬퍼 메서드 ───

func (s *Service) issueTokens(ctx context.Context, user User) (*LoginResponse, error) {
	claims := TokenClaims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role,
	}

	accessToken, err := GenerateAccessToken(s.cfg, claims)
	if err != nil {
		return nil, fmt.Errorf("access token 생성 실패: %w", err)
	}

	refreshToken, err := GenerateRefreshToken(s.cfg, claims)
	if err != nil {
		return nil, fmt.Errorf("refresh token 생성 실패: %w", err)
	}

	// 마지막 로그인 시간 업데이트
	now := time.Now()
	s.pool.Exec(ctx,
		`UPDATE users SET last_login_at=$1, updated_at=$2 WHERE id=$3`,
		now, now, user.ID,
	)

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) createMFASession(ctx context.Context, userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	// 5분 TTL
	err := s.redis.Set(ctx, "mfa:"+token, userID.String(), 5*time.Minute).Err()
	return token, err
}

func (s *Service) incrementLoginFail(ctx context.Context, user *User) error {
	now := time.Now()
	newCount := user.LoginFailCount + 1

	if newCount >= 5 {
		// 30분 잠금
		lockedUntil := now.Add(30 * time.Minute)
		_, err := s.pool.Exec(ctx,
			`UPDATE users SET login_fail_count=$1, locked_until=$2, status='locked', updated_at=$3 WHERE id=$4`,
			newCount, lockedUntil, now, user.ID,
		)
		return err
	}

	_, err := s.pool.Exec(ctx,
		`UPDATE users SET login_fail_count=$1, updated_at=$2 WHERE id=$3`,
		newCount, now, user.ID,
	)
	return err
}

func (s *Service) resetLoginFail(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET login_fail_count=0, locked_until=NULL, status='active', updated_at=$1 WHERE id=$2`,
		time.Now(), userID,
	)
	return err
}

func (s *Service) unlockAccount(ctx context.Context, userID uuid.UUID) error {
	return s.resetLoginFail(ctx, userID)
}

// ─── DB 쿼리 헬퍼 (prepared statement 기반) ───

func (s *Service) findUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	var email *string
	var mfaSecret *string
	var passwordChangedAt, lastLoginAt, lockedUntil *time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, mfa_secret, mfa_enabled,
		        role, status, must_change_password, password_changed_at,
		        last_login_at, login_fail_count, locked_until, created_at, updated_at
		 FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.Username, &email, &u.PasswordHash, &mfaSecret, &u.MFAEnabled,
		&u.Role, &u.Status, &u.MustChangePassword, &passwordChangedAt,
		&lastLoginAt, &u.LoginFailCount, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}

	u.Email = email
	u.MFASecret = mfaSecret
	u.PasswordChangedAt = passwordChangedAt
	u.LastLoginAt = lastLoginAt
	u.LockedUntil = lockedUntil

	return &u, nil
}

func (s *Service) findUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var u User
	var email, mfaSecret *string
	var passwordChangedAt, lastLoginAt, lockedUntil *time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, mfa_secret, mfa_enabled,
		        role, status, must_change_password, password_changed_at,
		        last_login_at, login_fail_count, locked_until, created_at, updated_at
		 FROM users WHERE id=$1`, userID,
	).Scan(&u.ID, &u.Username, &email, &u.PasswordHash, &mfaSecret, &u.MFAEnabled,
		&u.Role, &u.Status, &u.MustChangePassword, &passwordChangedAt,
		&lastLoginAt, &u.LoginFailCount, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}

	u.Email = email
	u.MFASecret = mfaSecret
	u.PasswordChangedAt = passwordChangedAt
	u.LastLoginAt = lastLoginAt
	u.LockedUntil = lockedUntil

	return &u, nil
}

func (s *Service) createUser(ctx context.Context, u *User) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, role, status, must_change_password, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.Status, u.MustChangePassword, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (s *Service) usernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)", username).Scan(&exists)
	return exists, err
}

func (s *Service) emailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
	return exists, err
}


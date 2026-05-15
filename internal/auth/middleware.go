package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
	roleKey     contextKey = "role"
)

// Middleware — JWT 인증 미들웨어
func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "인증 토큰이 필요합니다")
				return
			}

			claims, err := ValidateToken(token, svc.cfg.Secret)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "유효하지 않은 토큰입니다")
				return
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "잘못된 토큰 정보")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, usernameKey, claims.Username)
			ctx = context.WithValue(ctx, roleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken — Authorization 헤더에서 Bearer 토큰 추출
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}

// GetUserID — 컨텍스트에서 사용자 ID 추출
func GetUserID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(userIDKey).(uuid.UUID)
	return id, ok
}

// GetUsername — 컨텍스트에서 사용자명 추출
func GetUsername(r *http.Request) (string, bool) {
	name, ok := r.Context().Value(usernameKey).(string)
	return name, ok
}

// GetRole — 컨텍스트에서 역할 추출
func GetRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(roleKey).(string)
	return role, ok
}

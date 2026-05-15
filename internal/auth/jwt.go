package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig — JWT 서명 설정
type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration // 기본: 15분
	RefreshExpiry  time.Duration // 기본: 7일
}

// GenerateAccessToken — 15분 짧은 만료의 access token
func GenerateAccessToken(cfg JWTConfig, claims TokenClaims) (string, error) {
	if cfg.AccessExpiry == 0 {
		cfg.AccessExpiry = 15 * time.Minute
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
		"exp":      time.Now().Add(cfg.AccessExpiry).Unix(),
		"iat":      time.Now().Unix(),
		"jti":      generateTokenID(),
	})

	return token.SignedString([]byte(cfg.Secret))
}

// GenerateRefreshToken — 7일 만료의 refresh token
func GenerateRefreshToken(cfg JWTConfig, claims TokenClaims) (string, error) {
	if cfg.RefreshExpiry == 0 {
		cfg.RefreshExpiry = 7 * 24 * time.Hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": claims.UserID,
		"type":    "refresh",
		"exp":     time.Now().Add(cfg.RefreshExpiry).Unix(),
		"iat":     time.Now().Unix(),
		"jti":     generateTokenID(),
	})

	return token.SignedString([]byte(cfg.Secret))
}

// ValidateToken — 토큰 검증 후 claims 반환
func ValidateToken(tokenString string, secret string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("예상치 못한 서명 알고리즘: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("토큰 검증 실패: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("유효하지 않은 토큰")
	}

	return &TokenClaims{
		UserID:   claims["user_id"].(string),
		Username: claims["username"].(string),
		Role:     claims["role"].(string),
	}, nil
}

// generateTokenID — 고유한 JWT ID 생성 (중복 방지, 블랙리스트 식별용)
func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

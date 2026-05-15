package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id 파라미터 — OWASP 권장값
const (
	argonTime    = 3         // 반복 횟수
	argonMemory  = 64 * 1024 // 64MB
	argonThreads = 4         // 병렬도
	argonKeyLen  = 32        // 해시 길이 (bytes)
	argonSaltLen = 16        // 솔트 길이 (bytes)
)

// HashPassword — Argon2id로 비밀번호 해싱
// 반환값 형식: $argon2id$v=19$m=65536,t=3,p=4$<base64salt>$<base64hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("솔트 생성 실패: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword — Argon2id 해시와 평문 비밀번호 비교
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("잘못된 해시 형식")
	}

	var memory, time, threads int
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("파라미터 파싱 실패: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("솔트 디코딩 실패: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("해시 디코딩 실패: %w", err)
	}

	computed := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

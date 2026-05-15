package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"math"
	"time"
)

// GenerateTOTPSecret — TOTP Secret 생성 (Google Authenticator 호환)
func GenerateTOTPSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// VerifyTOTP — TOTP 코드 검증
func VerifyTOTP(secret string, code string) bool {
	// 30초 윈도우, ±1 단계 허용 (앞뒤 시간대)
	now := time.Now().Unix()
	counter := uint64(now / 30)

	for i := int64(-1); i <= 1; i++ {
		expected := generateTOTP(secret, counter+uint64(i))
		if expected == code {
			return true
		}
	}
	return false
}

// generateTOTP — 주어진 카운터에 대한 TOTP 생성
func generateTOTP(secret string, counter uint64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(counter & 0xff)
		counter >>= 8
	}
	mac.Write(b)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	binary := int((hash[offset]&0x7f)<<24 |
		(hash[offset+1]&0xff)<<16 |
		(hash[offset+2]&0xff)<<8 |
		(hash[offset+3]&0xff))

	otp := int(math.Mod(float64(binary), math.Pow10(6)))
	return fmt.Sprintf("%06d", otp)
}

// GetTOTPQRCodeURL — Google Authenticator QR 코드 URL
func GetTOTPQRCodeURL(secret, username string) string {
	return fmt.Sprintf(
		"otpauth://totp/SecureGate:%s?secret=%s&issuer=SecureGate&algorithm=SHA1&digits=6&period=30",
		username,
		secret,
	)
}

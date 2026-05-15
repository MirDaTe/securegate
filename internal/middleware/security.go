package middleware

import "net/http"

// SecurityHeadersMiddleware — 보안 헤더 설정 (CSP, HSTS, X-Frame 등)
// K-ISMS / ISMS-P 컴플라이언스 요구사항 충족
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Security-Policy: 외부 리소스 차단 (망분리 핵심)
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"font-src 'self'; "+
				"connect-src 'self' ws: wss:; "+
				"media-src 'self' blob:; "+
				"frame-ancestors 'none'")

		// HSTS: 1년, subdomain 포함
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Clickjacking 방지
		w.Header().Set("X-Frame-Options", "DENY")

		// MIME sniffing 방지
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer 정보 최소화
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// XSS 필터 활성화
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}

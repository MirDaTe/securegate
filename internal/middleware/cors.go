package middleware

import "net/http"

// CORSMiddleware — CORS 설정 (망분리 환경: 개발 단계에서는 허용적, 프로덕션에서는 제한적)
func CORSMiddleware(cfg interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 개발 환경: 모든 origin 허용 (Vite dev server)
			// 프로덕션 환경: Nginx에서 처리하므로 API 서버는 관대하게
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

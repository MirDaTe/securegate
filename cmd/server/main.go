package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mirdate/securegate/internal/auth"
	"github.com/mirdate/securegate/internal/config"
	"github.com/mirdate/securegate/internal/db"
	"github.com/mirdate/securegate/internal/host"
	"github.com/mirdate/securegate/internal/middleware"
	"github.com/mirdate/securegate/internal/policy"
)

func main() {
	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}

	// DB 연결
	if err := db.InitPostgres(cfg); err != nil {
		log.Fatalf("PostgreSQL 연결 실패: %v", err)
	}
	defer db.ClosePostgres()

	if err := db.InitRedis(cfg); err != nil {
		log.Fatalf("Redis 연결 실패: %v", err)
	}
	defer db.CloseRedis()

	// DB 마이그레이션 실행
	if err := db.RunMigrations(cfg); err != nil {
		log.Fatalf("DB 마이그레이션 실패: %v", err)
	}

	// Auth 서비스 초기화
	authSvc := auth.NewService(auth.JWTConfig{
		Secret:        cfg.JWTSecret,
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	})

	// Host + Policy 서비스 초기화
	hostSvc := host.NewService()
	policySvc := policy.NewService()
	policyEngine := policy.NewEngine()

	_ = policyEngine // Step 4~5에서 WebSocket 세션 정책 평가에 사용

	// 초기 관리자 계정 생성
	if err := authSvc.CreateAdmin(context.Background(), cfg.AdminPass); err != nil {
		log.Fatalf("초기 관리자 계정 생성 실패: %v", err)
	}
	log.Println("초기 관리자 계정 확인 완료 (admin)")

	// 라우터 설정
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.SecurityHeadersMiddleware)
	r.Use(middleware.RateLimitMiddleware)
	r.Use(chimw.Recoverer)

	// 공개 엔드포인트
	r.Get("/api/health", healthHandler)

	// 인증 엔드포인트 (로그인, 회원가입 등)
	authHandler := auth.NewHandler(authSvc)
	r.Route("/api", func(r chi.Router) {
		authHandler.RegisterRoutes(r)
	})

	// 인증 필요한 API
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(authSvc))

		// 내 정보
		r.Get("/api/me", func(w http.ResponseWriter, r *http.Request) {
			userID, _ := auth.GetUserID(r)
			user, err := authSvc.GetUser(r.Context(), userID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
				return
			}
			writeJSON(w, http.StatusOK, user)
		})

		// 호스트 관리
		hostHandler := host.NewHandler(hostSvc)
		r.Route("/api", hostHandler.RegisterRoutes)

		// 정책 관리
		policyHandler := policy.NewHandler(policySvc)
		r.Route("/api", policyHandler.RegisterRoutes)
	})

	// 정적 파일 서빙 (프론트엔드 — 프로덕션에서만)
	if cfg.ServeStatic {
		fileServer := http.FileServer(http.Dir("./web/dist"))
		r.Handle("/*", fileServer)
	}

	// 서버 시작
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("서버 종료 중...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("SecureGate 서버 시작 — %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("서버 시작 실패: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	err := db.Ping()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"db":     fmt.Sprintf("error: %v", err),
		})
		return
	}

	redisErr := db.PingRedis()
	redisStatus := "connected"
	if redisErr != nil {
		redisStatus = fmt.Sprintf("error: %v", redisErr)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"db":     "connected",
		"redis":  redisStatus,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

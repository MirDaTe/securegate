package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mirdate/securegate/internal/config"
	"github.com/mirdate/securegate/internal/db"
	"github.com/mirdate/securegate/internal/middleware"
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

	// 라우터 설정
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.CORSMiddleware(cfg))
	r.Use(middleware.SecurityHeadersMiddleware)
	r.Use(middleware.RateLimitMiddleware)
	r.Use(chimw.Recoverer)

	// 헬스 체크
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
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

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"db":     "connected",
			"redis":  redisStatus,
		})
	})

	// 정적 파일 서빙 (프론트엔드 — 프로덕션에서만)
	// 개발 중에는 Vite dev server (port 5173) 사용
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

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"status":"%s","db":"%s","redis":"%s"}`,
		data.(map[string]string)["status"],
		data.(map[string]string)["db"],
		data.(map[string]string)["redis"],
	)
}

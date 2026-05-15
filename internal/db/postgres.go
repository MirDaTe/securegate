package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mirdate/securegate/internal/config"
)

var pool *pgxpool.Pool
var ctx = context.Background()

func InitPostgres(cfg *config.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	var err error
	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("PostgreSQL 연결 풀 생성 실패: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL ping 실패: %w", err)
	}

	return nil
}

func ClosePostgres() {
	if pool != nil {
		pool.Close()
	}
}

func Ping() error {
	if pool == nil {
		return fmt.Errorf("PostgreSQL 연결이 초기화되지 않았습니다")
	}
	return pool.Ping(ctx)
}

func Pool() *pgxpool.Pool {
	return pool
}

// RunMigrations — migrations 디렉터리의 모든 .up.sql 파일을 파일명 순서대로 실행
func RunMigrations(cfg *config.Config) error {
	// 마이그레이션 테이블 생성 (존재하지 않으면)
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("마이그레이션 테이블 생성 실패: %w", err)
	}

	// 마이그레이션 파일 목록 조회
	migrationFiles, err := filepath.Glob(filepath.Join(cfg.MigrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("마이그레이션 파일 검색 실패: %w", err)
	}
	sort.Strings(migrationFiles)

	for _, f := range migrationFiles {
		version := filepath.Base(f)
		// 이미 적용된 마이그레이션인지 확인
		var exists bool
		err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("마이그레이션 상태 확인 실패 (%s): %w", version, err)
		}
		if exists {
			continue
		}

		// SQL 파일 읽기
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("마이그레이션 파일 읽기 실패 (%s): %w", f, err)
		}

		// 트랜잭션으로 실행
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("트랜잭션 시작 실패 (%s): %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("마이그레이션 실행 실패 (%s): %w", version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("마이그레이션 버전 기록 실패 (%s): %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("마이그레이션 커밋 실패 (%s): %w", version, err)
		}

		fmt.Printf("마이그레이션 적용 완료: %s\n", version)
	}

	return nil
}

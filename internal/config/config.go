package config

import (
	"os"
	"strings"
)

// Config 구조체 — 모든 설정은 환경변수에서 로드 (망분리에서 12-factor 패턴)
type Config struct {
	ServerPort  string
	JWTSecret   string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	RedisHost   string
	RedisPort   string
	RedisPass   string
	AdminPass   string // INITIAL_ADMIN_PASSWORD — 최초 설치 시만 사용
	ServeStatic bool
	LogLevel    string
	MigrationsDir string
}

func Load() (*Config, error) {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		JWTSecret:     requireEnv("JWT_SECRET"),
		DBHost:        getEnv("DB_HOST", "postgres"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "securegate"),
		DBPassword:    requireEnv("DB_PASSWORD"),
		DBName:        getEnv("DB_NAME", "securegate"),
		RedisHost:     getEnv("REDIS_HOST", "redis"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPass:     getEnv("REDIS_PASSWORD", ""),
		AdminPass:     requireEnv("INITIAL_ADMIN_PASSWORD"),
		ServeStatic:   getEnvBool("SERVE_STATIC", "false"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "./migrations"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

// requireEnv — 필수 환경변수가 없으면 panic
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("필수 환경변수 " + key + " 가 설정되지 않았습니다")
	}
	return strings.TrimSpace(v)
}

func getEnvBool(key, fallback string) bool {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	return v == "true" || v == "1"
}

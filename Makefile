# SecureGate Makefile — 빌드, 테스트, 배포 자동화

.PHONY: help dev build test docker-up docker-down clean

help: ## 도움말 출력
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## 개발 모드 실행 (API 서버 + Vite dev server)
	@echo "SecureGate 개발 서버 시작..."
	go run ./cmd/server &
	cd web && npm run dev

build: ## 프로덕션 빌드
	@echo "🔨 프론트엔드 빌드..."
	cd web && npm ci && npm run build
	@echo "🔨 백엔드 빌드..."
	go build -o ./bin/securegate ./cmd/server

build-linux: ## Linux 크로스 컴파일 빌드
	@echo "🔨 Linux 크로스 컴파일..."
	cd web && npm ci && npm run build
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/securegate-linux ./cmd/server

build-windows: ## Windows 크로스 컴파일 빌드
	@echo "🔨 Windows 크로스 컴파일..."
	cd web && npm ci && npm run build
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/securegate.exe ./cmd/server

test: ## 테스트 실행
	go test ./internal/... -v -count=1

docker-up: ## Docker Compose 기동
	docker compose -f deployments/docker/docker-compose.yml up -d

docker-down: ## Docker Compose 종료
	docker compose -f deployments/docker/docker-compose.yml down

docker-logs: ## Docker Compose 로그 확인
	docker compose -f deployments/docker/docker-compose.yml logs -f

clean: ## 빌드 결과물 정리
	rm -rf ./bin ./web/dist

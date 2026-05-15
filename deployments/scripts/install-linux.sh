#!/bin/bash
# SecureGate - Linux 오프라인 설치 스크립트 (망분리 환경용)
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  SecureGate 설치 마법사${NC}"
echo -e "${GREEN}  망분리 환경 웹 원격 데스크톱 게이트웨이${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""

# ─── 1. 필수 의존성 확인 ───
echo -e "${YELLOW}[1/5] 의존성 확인...${NC}"
if ! command -v docker &>/dev/null; then
    echo -e "${RED}Docker가 설치되어 있지 않습니다.${NC}"
    exit 1
fi
echo -e "${GREEN}  OK Docker 확인 완료${NC}"

# ─── 2. TLS 인증서 ───
echo -e "${YELLOW}[2/5] TLS 인증서...${NC}"
SSL_DIR="./ssl"
if [ ! -f "$SSL_DIR/cert.pem" ]; then
    mkdir -p "$SSL_DIR"
    openssl req -x509 -newkey rsa:4096 -keyout "$SSL_DIR/key.pem" \
        -out "$SSL_DIR/cert.pem" -days 3650 -nodes \
        -subj "/CN=SecureGate" 2>/dev/null
fi
echo -e "${GREEN}  OK TLS 인증서 확인 완료${NC}"

# ─── 3. 환경변수 ───
echo -e "${YELLOW}[3/5] 환경변수...${NC}"
if [ ! -f ".env" ]; then
    cp .env.example .env
    ADMIN_PASS=$(openssl rand -base64 16)
    JWT_SECRET=$(openssl rand -base64 48)
    sed -i '' "s/INITIAL_ADMIN_PASSWORD=.*/INITIAL_ADMIN_PASSWORD=$ADMIN_PASS/" .env 2>/dev/null || \
    sed -i "s/INITIAL_ADMIN_PASSWORD=.*/INITIAL_ADMIN_PASSWORD=$ADMIN_PASS/" .env
    sed -i '' "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env 2>/dev/null || \
    sed -i "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
    echo ""
    echo "  ────────────────────────────────────"
    echo "  초기 관리자 로그인 정보"
    echo "  ID: admin"
    echo "  PW: $ADMIN_PASS"
    echo "  (최초 로그인 시 비밀번호 변경 필수)"
    echo "  ────────────────────────────────────"
fi

# ─── 4. 실행 ───
echo -e "${YELLOW}[4/5] SecureGate 실행...${NC}"
docker compose -f deployments/docker/docker-compose.yml up -d

# ─── 5. 완료 ───
echo ""
echo -e "${GREEN}설치 완료! https://localhost 로 접속하세요${NC}"
echo "로그: docker compose -f deployments/docker/docker-compose.yml logs -f"

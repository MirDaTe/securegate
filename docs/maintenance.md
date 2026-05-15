# SecureGate 유지보수 가이드

> **대상:** 개발자, 유지보수 담당자 | **버전:** v0.1.0

---

## 목차

1. [버전 업데이트](#1-버전-업데이트)
2. [디버깅 방법](#2-디버깅-방법)
3. [트러블슈팅](#3-트러블슈팅)
4. [DB 마이그레이션](#4-db-마이그레이션)
5. [성능 튜닝](#5-성능-튜닝)
6. [패치 적용 가이드](#6-패치-적용-가이드)

---

## 1. 버전 업데이트

### 1.1 온라인 환경 — Git pull

```bash
cd /opt/securegate
git pull origin main
docker compose -f deployments/docker/docker-compose.yml up -d --build
```

### 1.2 오프라인 환경 — 바이너리 교체

```bash
# 1. 현재 버전 백업
cp -r /opt/securegate /opt/securegate.bak

# 2. 새 바이너리/이미지 복사 (USB에서)
cp /mnt/usb/securegate-bin/securegate /opt/securegate/bin/
docker load < /mnt/usb/securegate-images/server.tar

# 3. DB 마이그레이션 확인
docker compose -f deployments/docker/docker-compose.yml up -d server
docker logs securegate-server | grep "마이그레이션"

# 4. 재시작
docker compose -f deployments/docker/docker-compose.yml restart
```

### 1.3 무중단 업데이트 (추후 지원)

```bash
# 현재는 단일 인스턴스이므로 약 10~30초 다운타임 발생
# 추후 블루-그린 배포 구성 예정
```

---

## 2. 디버깅 방법

### 2.1 Go 백엔드 디버깅

```bash
# 디버그 로그 활성화
LOG_LEVEL=debug go run ./cmd/server

# 특정 패키지만 테스트
go test ./internal/auth/... -v
go test ./internal/policy/... -v

# 레이스 컨디션 검사
go test ./internal/... -race
```

### 2.2 프론트엔드 디버깅

```bash
cd web
npm run dev  # Vite dev server + Hot Module Replacement

# 브라우저 DevTools에서:
#   - Network 탭: API 호출 확인
#   - Console: WebSocket 프레임 디버깅 로그
#   - Application > WebSocket: 프레임 내용 확인
```

### 2.3 WebSocket 디버깅

```bash
# wscat으로 직접 WS 연결 테스트
npm install -g wscat
wscat -c "wss://localhost/ws/session/<sessionId>?token=<wsToken>" --no-check

# 브라우저 콘솔 스니펫
const ws = new WebSocket('wss://localhost/ws/session/xxx?token=yyy');
ws.binaryType = 'arraybuffer';
ws.onmessage = (e) => console.log(new Uint8Array(e.data));
```

### 2.4 RDP 디버깅 (FreeRDP)

```bash
# FreeRDP CLI로 직접 연결 테스트
xfreerdp /v:<host>:3389 /u:<user> /p:<pass> /w:1024 /h:768 /cert:ignore

# 로그 레벨 DEBUG
xfreerdp /v:<host>:3389 /u:<user> /p:<pass> /log-level:DEBUG
```

---

## 3. 트러블슈팅

### 3.1 로그인 불가

**증상:** `{"error":"아이디 또는 비밀번호가 올바르지 않습니다"}`

**조치:**
```sql
-- 사용자 상태 확인
SELECT username, status, login_fail_count, locked_until FROM users WHERE username='admin';

-- 계정 잠금 해제 (관리자가 잠긴 경우)
-- PostgreSQL 직접 접속
UPDATE users SET status='active', login_fail_count=0, locked_until=NULL WHERE username='admin';
```

### 3.2 세션 연결 실패

**증상:** WebSocket 연결 후 바로 끊김

**조치:**
```bash
# 1. 대상 호스트 연결 확인 (SecureGate 서버에서)
nc -zv <host> <port>  # RDP:3389, SSH:22

# 2. FreeRDP 상태 확인
which xfreerdp  # 설치되어 있어야 함
apt-get install freerdp2-x11  # 없으면 설치

# 3. Redis 상태 확인
docker exec securegate-redis redis-cli ping  # PONG 응답 필수
```

### 3.3 한/영 키 동작 불량

**증상:** 한/영 키가 게스트에 전달되지 않음

**조치:**
- SessionPage에서 IME 모드가 "게스트 IME"인지 확인
- 브라우저 콘솔에서 `codeToScanCode('Lang1')` → `114` 반환 확인
- RDP: xfreerdp가 `/gfx` 모드인지 확인
- Chrome의 경우 `chrome://flags/#ime-service` 비활성화 필요할 수 있음

### 3.4 DB 마이그레이션 실패

**증상:** `"DB 마이그레이션 실패"`

**조치:**
```bash
# 마이그레이션 상태 확인
docker exec securegate-postgres psql -U securegate securegate \
  -c "SELECT * FROM schema_migrations ORDER BY applied_at;"

# 실패한 마이그레이션 롤백
docker exec securegate-postgres psql -U securegate securegate \
  -c "DELETE FROM schema_migrations WHERE version LIKE '%실패한파일%';"
```

---

## 4. DB 마이그레이션

### 4.1 새 마이그레이션 추가

```bash
# 1. migrations/ 디렉터리에 새 파일 생성
cat > migrations/002_add_sessions_index.up.sql << 'EOF'
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at);
EOF

# 2. 롤백 파일도 함께 생성
cat > migrations/002_add_sessions_index.down.sql << 'EOF'
DROP INDEX IF EXISTS idx_sessions_created;
EOF

# 3. 배포 (서버 재시작 시 자동 적용)
docker compose -f deployments/docker/docker-compose.yml up -d server
```

### 4.2 마이그레이션 상태 확인

```sql
SELECT version, applied_at FROM schema_migrations ORDER BY applied_at;
```

---

## 5. 성능 튜닝

### 5.1 PostgreSQL

```sql
-- 동시 접속 수 확인
SHOW max_connections;  -- 기본 100

-- 활성 쿼리 확인
SELECT pid, state, query, query_start FROM pg_stat_activity WHERE state='active';
```

커넥션 풀 조정:
```yaml
# docker-compose.yml
server:
  environment:
    PG_MAX_CONNS: "25"  # connection pool size
```

### 5.2 Redis

```bash
# 메모리 사용량
docker exec securegate-redis redis-cli INFO memory

# 키 개수
docker exec securegate-redis redis-cli DBSIZE

# 불필요한 세션 데이터 정리 (TTL로 자동 만료됨)
docker exec securegate-redis redis-cli --scan --pattern "ws_session:*" | wc -l
```

### 5.3 WebSocket

```bash
# 최대 동시 연결 수 확인
docker logs securegate-server | grep "Hub 등록" | wc -l

# 컨테이너 리소스 제한
# docker-compose.yml
server:
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 1G
```

---

## 6. 패치 적용 가이드

### 6.1 핫픽스 적용 절차

```bash
# 1. 현재 상태 백업
bash deployments/scripts/backup.sh

# 2. 코드 수정
# (patch 도구로 파일 수정 또는 git cherry-pick)

# 3. 빌드
go build -o ./bin/securegate ./cmd/server

# 4. 배포
docker compose -f deployments/docker/docker-compose.yml up -d --build server

# 5. 검증
curl -k https://localhost/api/health
```

### 6.2 롤백 절차

```bash
# 1. 이전 커밋으로 되돌리기
git log --oneline -5
git revert <커밋해시>  # 또는 git reset --hard <커밋해시>

# 2. 재배포
docker compose -f deployments/docker/docker-compose.yml up -d --build

# 3. 검증
curl -k https://localhost/api/health
```

---

## 부록: 자주 발생하는 문제와 해결책

| 문제 | 가능한 원인 | 해결 방법 |
|------|------------|-----------|
| 502 Bad Gateway | API 서버 다운 | `docker compose restart server` |
| 504 Gateway Timeout | WebSocket 세션 과부하 | 서버 리소스 증설, idle 세션 정리 |
| Connection refused | 방화벽 차단 | 443 포트 개방 확인 |
| SSL 인증서 오류 | 자체 서명 인증서 | 브라우저에서 예외 추가 또는 공인 인증서 적용 |
| 느린 RDP 응답 | 네트워크 지연 | `/gfx` 대신 `/rfx` 모드 시도 |
| Redis OOM | 메모리 부족 | `maxmemory-policy allkeys-lru` 설정 |

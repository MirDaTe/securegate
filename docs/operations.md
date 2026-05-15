# SecureGate 운영 매뉴얼

> **대상:** 시스템 관리자, 운영자 | **버전:** v0.1.0

---

## 목차

1. [시스템 시작](#1-시스템-시작)
2. [시스템 중지](#2-시스템-중지)
3. [상태 확인](#3-상태-확인)
4. [로그 확인](#4-로그-확인)
5. [백업 및 복구](#5-백업-및-복구)
6. [모니터링](#6-모니터링)
7. [비상 상황 대응](#7-비상-상황-대응)

---

## 1. 시스템 시작

### 1.1 Docker Compose (권장)

```bash
# 기본 시작 (백그라운드)
cd /opt/securegate  # 또는 설치 경로
docker compose -f deployments/docker/docker-compose.yml up -d

# 최초 설치 시 (TLS 인증서 자동 생성, 관리자 비밀번호 자동 발급)
bash deployments/scripts/install-linux.sh
```

### 1.2 개별 서비스 시작

```bash
# PostgreSQL만 시작
docker compose -f deployments/docker/docker-compose.yml up -d postgres redis

# API 서버만 시작
docker compose -f deployments/docker/docker-compose.yml up -d server nginx
```

### 1.3 시작 확인

```bash
# 헬스 체크
curl -k https://localhost/api/health
# 정상 응답: {"status":"ok","db":"connected","redis":"connected"}

# 컨테이너 상태 확인
docker compose -f deployments/docker/docker-compose.yml ps
```

### 1.4 Windows 서비스 등록

```powershell
# 관리자 권한 PowerShell
New-Service -Name "SecureGate" `
  -BinaryPathName "docker compose -f C:\securegate\deployments\docker\docker-compose.yml up" `
  -StartupType Automatic

Start-Service SecureGate
```

---

## 2. 시스템 중지

### 2.1 정상 중지

```bash
# 모든 서비스 중지 (데이터 보존)
docker compose -f deployments/docker/docker-compose.yml down

# 컨테이너만 중지 (볼륨 유지)
docker compose -f deployments/docker/docker-compose.yml stop
```

### 2.2 완전 삭제 (데이터 포함)

```bash
# ⚠️ 주의: 모든 데이터가 삭제됩니다
docker compose -f deployments/docker/docker-compose.yml down -v
```

### 2.3 개별 서비스 재시작

```bash
# API 서버만 재시작 (무중단 불가능 — 단일 인스턴스)
docker compose -f deployments/docker/docker-compose.yml restart server

# Nginx만 재시작
docker compose -f deployments/docker/docker-compose.yml restart nginx
```

---

## 3. 상태 확인

### 3.1 API 헬스 체크

```bash
curl -k https://localhost/api/health
```

### 3.2 활성 세션 수 확인

```bash
# Hub에 등록된 세션 수 (로그에서 확인)
docker compose -f deployments/docker/docker-compose.yml logs server | grep "Hub 등록" | wc -l
```

### 3.3 감사 로그 무결성 검증

```bash
# 관리자 API 토큰 필요
JWT="Bearer <your-admin-token>"
curl -k -H "Authorization: $JWT" https://localhost/api/audit/verify?limit=100
# 응답: {"valid":true,"count":100}
```

---

## 4. 로그 확인

### 4.1 전체 서비스 로그

```bash
# 실시간 로그
docker compose -f deployments/docker/docker-compose.yml logs -f

# 최근 200줄
docker compose -f deployments/docker/docker-compose.yml logs --tail=200

# 특정 서비스만
docker compose -f deployments/docker/docker-compose.yml logs server
docker compose -f deployments/docker/docker-compose.yml logs nginx
```

### 4.2 로그 파일 위치

| 서비스 | 로그 경로 |
|--------|-----------|
| API 서버 | `docker logs securegate-server` |
| Nginx | `/var/log/nginx/access.log` (컨테이너 내부) |
| PostgreSQL | `docker logs securegate-postgres` |
| Redis | `docker logs securegate-redis` |

### 4.3 로그 레벨 변경

`.env` 파일에서 `LOG_LEVEL` 수정:
```
LOG_LEVEL=debug  # debug, info, warn, error
```

변경 후 서버 재시작:
```bash
docker compose -f deployments/docker/docker-compose.yml restart server
```

---

## 5. 백업 및 복구

### 5.1 PostgreSQL 백업

```bash
# 전체 DB 덤프
docker exec securegate-postgres pg_dump -U securegate securegate > backup_$(date +%Y%m%d).sql

# 감사 로그만 별도 백업
docker exec securegate-postgres pg_dump -U securegate securegate \
  --table=audit_logs --table=file_transfers > audit_backup_$(date +%Y%m%d).sql
```

### 5.2 자동 백업 (cron)

```bash
# 매일 새벽 2시에 백업
0 2 * * * cd /opt/securegate && docker exec securegate-postgres pg_dump -U securegate securegate | gzip > backups/daily_$(date +\%Y\%m\%d).sql.gz

# 주간 백업 (매주 일요일)
0 3 * * 0 cd /opt/securegate && docker exec securegate-postgres pg_dump -U securegate securegate | gzip > backups/weekly_$(date +\%Y\%W).sql.gz
```

### 5.3 복구

```bash
# DB 복구 (⚠️ 기존 데이터 덮어씀)
docker exec -i securegate-postgres psql -U securegate securegate < backup_20260515.sql

# 서버 재시작
docker compose -f deployments/docker/docker-compose.yml restart server
```

### 5.4 데이터 볼륨 백업

```bash
# Docker 볼륨 백업
docker run --rm -v securegate_postgres_data:/data -v $(pwd)/backups:/backup \
  alpine tar czf /backup/postgres_volume_$(date +%Y%m%d).tar.gz -C /data .

docker run --rm -v securegate_redis_data:/data -v $(pwd)/backups:/backup \
  alpine tar czf /backup/redis_volume_$(date +%Y%m%d).tar.gz -C /data .
```

---

## 6. 모니터링

### 6.1 기본 리소스 모니터링

```bash
# 컨테이너 리소스 사용량
docker stats securegate-server securegate-postgres securegate-redis

# 디스크 사용량
docker system df
```

### 6.2 감사 로그 이상 징후 모니터링

```bash
# 특정 시간 동안 로그인 실패 횟수
docker exec securegate-postgres psql -U securegate securegate \
  -c "SELECT actor_ip, COUNT(*) FROM audit_logs WHERE action='login' AND detail::text LIKE '%fail%' AND event_ts > NOW() - INTERVAL '1 hour' GROUP BY actor_ip HAVING COUNT(*) > 5;"
```

### 6.3 Prometheus + Grafana (선택)

```yaml
# docker-compose.yml에 추가
  prometheus:
    image: prom/prometheus
    ports: ["9090:9090"]
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports: ["3000:3000"]
```

---

## 7. 비상 상황 대응

### 7.1 서버 다운

```bash
# 1. 로그 확인
docker compose -f deployments/docker/docker-compose.yml logs server --tail=50

# 2. DB/Redis 연결 확인
docker compose -f deployments/docker/docker-compose.yml exec server wget -qO- http://localhost:8080/api/health

# 3. 재시작
docker compose -f deployments/docker/docker-compose.yml restart server
```

### 7.2 디스크 공간 부족

```bash
# 1. 사용량 확인
df -h
docker system df

# 2. 불필요한 이미지/컨테이너 정리
docker system prune -a

# 3. 오래된 로그 정리
truncate -s 0 /var/log/securegate/*.log
```

### 7.3 의심스러운 활동 탐지 시

```bash
# 1. 해당 사용자의 모든 세션 강제 종료
# (관리자 콘솔에서 수행 또는 audit log로 session ID 확인)

# 2. 사용자 계정 잠금
docker exec securegate-postgres psql -U securegate securegate \
  -c "UPDATE users SET status='locked', locked_until=NOW()+INTERVAL'24 hours' WHERE username='<의심계정>';"

# 3. 감사 로그 무결성 검증
curl -k https://localhost/api/audit/verify?limit=1000

# 4. 전체 로그 보관
docker compose -f deployments/docker/docker-compose.yml logs > incident_$(date +%Y%m%d_%H%M%S).log
```

---

## 부록: 유용한 명령어 모음

```bash
# 초기 관리자 비밀번호 확인 (.env 파일)
grep INITIAL_ADMIN_PASSWORD .env

# DB 직접 접속
docker exec -it securegate-postgres psql -U securegate securegate

# Redis 접속
docker exec -it securegate-redis redis-cli

# 활성 세션 수 (Redis)
docker exec securegate-redis redis-cli DBSIZE

# 서버 버전 확인
curl -k https://localhost/api/health | python3 -m json.tool
```

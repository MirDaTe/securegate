# SecureGate — 개발 가이드

> **대상:** 프로젝트 개발자 | **버전:** v0.1.0

---

## 목차

1. [개발 환경 설정](#1-개발-환경-설정)
2. [프로젝트 구조 이해하기](#2-프로젝트-구조-이해하기)
3. [코드 작성 규칙](#3-코드-작성-규칙)
4. [테스트 작성 및 실행](#4-테스트-작성-및-실행)
5. [커밋 메시지 규칙](#5-커밋-메시지-규칙)
6. [PR 및 코드 리뷰](#6-pr-및-코드-리뷰)
7. [빌드 및 배포](#7-빌드-및-배포)

---

## 1. 개발 환경 설정

### 1.1 필수 도구

| 도구 | 버전 | 용도 |
|------|------|------|
| Go | 1.22+ | 백엔드 API 서버 |
| Node.js | 18+ | 프론트엔드 개발 |
| Docker | 20.10+ | 컨테이너 실행 |
| Docker Compose | v2+ | 멀티 서비스 오케스트레이션 |
| Git | 2.40+ | 버전 관리 |

선택:
- `xfreerdp` — RDP 연결 테스트
- `wscat` — WebSocket 디버깅

### 1.2 초기 설정

```bash
# 1. 저장소 클론
git clone https://github.com/MirDaTe/securegate.git
cd securegate

# 2. 환경변수 파일 생성
cp .env.example .env
# .env 파일을 열어 JWT_SECRET, DB_PASSWORD, INITIAL_ADMIN_PASSWORD 수정

# 3. 의존성 설치
go mod download
cd web && npm install && cd ..

# 4. DB 및 Redis 기동
docker compose -f deployments/docker/docker-compose.yml up -d postgres redis

# 5. API 서버 실행
go run ./cmd/server

# 6. 별도 터미널에서 프론트엔드 개발 서버
cd web && npm run dev
```

### 1.3 IDE 설정

**VS Code 추천 확장:**
- Go (`golang.go`)
- ESLint (`dbaeumer.vscode-eslint`)
- Tailwind CSS IntelliSense (`bradlc.vscode-tailwindcss`)
- Prettier (`esbenp.prettier-vscode`)

---

## 2. 프로젝트 구조 이해하기

### 2.1 레이어드 아키텍처

```
Handler (HTTP) → Service (비즈니스 로직) → DB / Redis
     │
     └── Middleware (인증/로깅/CORS/보안)
```

### 2.2 주요 컴포넌트 의존성

```
cmd/server/main.go
  ├── config.Load()     → 환경변수 파싱
  ├── db.InitPostgres() → PostgreSQL 연결 풀
  ├── db.InitRedis()    → Redis 연결
  ├── auth.NewService() → 인증 서비스
  ├── host.NewService() → 호스트 관리
  ├── policy.NewEngine() → 정책 평가
  ├── session.NewManager() → 세션 관리
  ├── relay.NewHub()    → WebSocket 허브
  └── audit.NewLogger() → 감사 로그
```

### 2.3 API 엔드포인트 라우팅

```
/api
├── /health           → 상태 확인 (공개)
├── /auth/*           → 인증 (공개)
│   ├── /login
│   ├── /signup
│   ├── /logout
│   ├── /password/change
│   ├── /mfa/setup
│   ├── /mfa/verify
│   └── /refresh
├── /me               → 내 정보 (인증 필요)
├── /hosts/*          → 호스트 관리 (인증 필요)
├── /policies/*       → 정책 관리 (인증 필요)
├── /sessions/*       → 세션 관리 (인증 필요)
└── /audit/*          → 감사 로그 (인증+권한 필요)

/ws/session/{id}      → WebSocket (토큰 인증)
```

---

## 3. 코드 작성 규칙

### 3.1 Go 코드 스타일

```go
// 좋은 예: 명확한 패키지명, 에러 처리, 컨텍스트 전달
package auth

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
    if req.Username == "" {
        return nil, fmt.Errorf("사용자명이 비어있습니다")
    }
    
    user, err := s.findUserByUsername(ctx, req.Username)
    if err != nil {
        return nil, fmt.Errorf("아이디 또는 비밀번호가 올바르지 않습니다")
    }
    
    return &LoginResponse{User: *user}, nil
}
```

### 3.2 SQL 안전 규칙

```go
// ✅ Prepared Statement (항상)
pool.QueryRow(ctx, "SELECT * FROM users WHERE id=$1", userID)

// ❌ 문자열 연결 (절대 금지)
pool.QueryRow(ctx, "SELECT * FROM users WHERE id=" + userID)
```

### 3.3 React 컴포넌트 규칙

```tsx
// ✅ 함수 컴포넌트 + hooks
export default function MyPage() {
  const { t } = useTranslation();
  const [data, setData] = useState<MyType[]>([]);
  
  useEffect(() => {
    api.get('/api/data').then(r => setData(r.data));
  }, []);
  
  return <div>{data.map(d => <p key={d.id}>{d.name}</p>)}</div>;
}

// ❌ dangerouslySetInnerHTML (절대 금지)
// <div dangerouslySetInnerHTML={{__html: userInput}} />
```

---

## 4. 테스트 작성 및 실행

### 4.1 테스트 실행

```bash
# 전체 테스트
make test

# 특정 패키지
go test ./internal/auth/... -v

# 커버리지 리포트
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 레이스 컨디션 검사
go test ./internal/... -race
```

### 4.2 테스트 예시 (auth/password)

```go
package auth_test

import (
    "testing"
    "github.com/mirdate/securegate/internal/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
    password := "SecurePass123!"
    
    hash, err := auth.HashPassword(password)
    if err != nil {
        t.Fatalf("해싱 실패: %v", err)
    }
    
    valid, err := auth.VerifyPassword(password, hash)
    if err != nil || !valid {
        t.Fatal("올바른 비밀번호 검증 실패")
    }
    
    valid, _ = auth.VerifyPassword("wrong", hash)
    if valid {
        t.Fatal("잘못된 비밀번호가 검증됨")
    }
}
```

### 4.3 E2E 테스트 (Playwright)

```typescript
// tests/e2e/login.spec.ts
import { test, expect } from '@playwright/test';

test('초기 관리자 로그인 → 비밀번호 강제 변경', async ({ page }) => {
  await page.goto('https://localhost/login');
  await page.fill('[name="username"]', 'admin');
  await page.fill('[name="password"]', process.env.TEST_ADMIN_PASS!);
  await page.click('button[type="submit"]');
  
  // 비밀번호 변경 페이지로 리다이렉트 확인
  await expect(page).toHaveURL(/\/change-password/);
});
```

---

## 5. 커밋 메시지 규칙

### 형식
```
<type>: <한글 설명>

- 세부 변경사항 1
- 세부 변경사항 2
```

### type 종류
| type | 용도 |
|------|------|
| `feat` | 새로운 기능 |
| `fix` | 버그 수정 |
| `docs` | 문서 추가/수정 |
| `refactor` | 코드 리팩토링 |
| `test` | 테스트 추가 |
| `security` | 보안 관련 변경 |
| `chore` | 빌드/설정 등 |

### 예시
```
feat: RDP 한/영 키 scan code 전송 추가

- Lang1(0x72), Lang2(0x71) 매핑
- IME bypass 모드 추가
- 게스트/호스트 IME 토글 옵션
```

---

## 6. PR 및 코드 리뷰

### 6.1 PR 생성 전 체크리스트

- [ ] 모든 테스트 통과 (`make test`)
- [ ] Go: `go vet ./...` 통과
- [ ] 프론트엔드: `npm run lint` 통과
- [ ] 새로운 의존성 없음 (`go.mod` / `package.json` 변경 최소화)
- [ ] 보안 검토 (SQL Injection, XSS, SSRF 확인)
- [ ] 감사 로그가 필요한 작업이면 audit.Log 호출 확인

### 6.2 코드 리뷰 중점 사항

1. **보안** — SQL Injection, XSS, SSRF, 인증 우회 가능성
2. **에러 처리** — 모든 오류가 적절히 처리되고 로깅되는가
3. **정책 적용** — 새로운 릴레이 채널이 Policy Engine을 거치는가
4. **감사 로그** — 중요 작업이 audit.Logger로 기록되는가
5. **망분리 호환** — 외부 CDN/API 호출이 없는가

---

## 7. 빌드 및 배포

### 7.1 로컬 빌드

```bash
# 전체 빌드 (프론트엔드 + 백엔드)
make build

# 크로스 컴파일
make build-linux      # Linux AMD64
make build-windows    # Windows AMD64
```

### 7.2 Docker 빌드

```bash
# Docker 이미지 빌드
docker build -f deployments/docker/Dockerfile.server -t securegate:latest .

# Docker Compose로 전체 스택 실행
docker compose -f deployments/docker/docker-compose.yml up -d
```

### 7.3 오프라인 배포 번들 생성

```bash
# 1. 빌드
make build-linux

# 2. 번들 생성
tar czf securegate-offline-$(date +%Y%m%d).tar.gz \
  bin/securegate-linux \
  migrations/ \
  web/dist/ \
  deployments/docker/ \
  .env.example \
  deployments/scripts/install-linux.sh \
  docs/

# 3. USB에 복사하여 폐쇄망으로 전달
```

---

## 부록: Makefile 명령어

```bash
make help        # 전체 명령어 확인
make dev         # 개발 모드 (API + Vite)
make build       # 프로덕션 빌드
make test        # 테스트 실행
make docker-up   # Docker Compose 기동
make docker-down # Docker Compose 종료
make clean       # 빌드 결과물 정리
```

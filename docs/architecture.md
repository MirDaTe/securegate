# SecureGate — 아키텍처 설계 문서

> **최종 수정일:** 2026-05-15 | **버전:** v0.1.0

---

## 1. 시스템 개요

SecureGate는 **단일 포트(HTTPS 443)** 로 모든 원격 데스크톱 트래픽을 중계하는 게이트웨이입니다. 사용자는 웹 브라우저만으로 RDP, SSH, VNC 대상에 접속하며, 관리자는 통합 콘솔에서 정책·세션·감사를 관제합니다.

### 핵심 설계 원칙

1. **제로 클라이언트** — 별도 에이전트/확장 불필요
2. **단일 진입점** — 방화벽 정책 한 줄로 라우팅
3. **망분리 네이티브** — 외부 호출 없음, 오프라인 설치 가능
4. **계층형 보안** — TLS 1.3 + Argon2id + AES-256-GCM + 해시 체인

---

## 2. 컴포넌트 구성도

```
┌──────────────────────────────────────────────────────────────────────┐
│                        업무망 (사용자 PC)                             │
│                                                                       │
│   [브라우저] ───────── WSS (바이너리 프레임) ────────┐               │
│        │                                              │               │
│        └─────────── HTTPS/REST API ──────────────────┤               │
└──────────────────────────────────────────────────────┼───────────────┘
                                                       │
                                           방화벽 (443/TCP 단일)
                                                       │
┌──────────────────────────────────────────────────────┼───────────────┐
│                         DMZ (SecureGate 서버)        │               │
│                                                      ▼               │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                 Nginx (TLS 1.3 종단)                          │   │
│  │  • /api/*   → API 서버 (8080)                                 │   │
│  │  • /ws/*    → API 서버 (WebSocket upgrade)                    │   │
│  │  • /*       → React 정적 파일 (SPA)                           │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                              │                                       │
│  ┌──────────────────────────▼───────────────────────────────────┐   │
│  │                 Go API 서버 (:8080)                           │   │
│  │                                                                │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │   │
│  │  │ Auth     │  │ Policy   │  │ Session  │  │ Audit    │    │   │
│  │  │ Middle   │  │ Engine   │  │ Manager  │  │ Logger   │    │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │   │
│  │                                                                │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │            WebSocket Hub (relay.Hub)                  │    │   │
│  │  │  • 클라이언트 WS 연결 등록/해제                         │    │   │
│  │  │  • 바이너리 프레임 라우팅                               │    │   │
│  │  │  • 입력 채널 (InputCh) 기반 릴레이 파이프라인           │    │   │
│  │  └──────────────────────┬───────────────────────────────┘    │   │
│  └──────────────────────────┼────────────────────────────────────┘   │
│                              │                                       │
│  ┌──────────────────────────▼───────────────────────────────────┐   │
│  │                Protocol Relay (gatewayd)                      │   │
│  │                                                                │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐   │   │
│  │  │ SSH Handler    │  │ RDP Handler    │  │ VNC Handler  │   │   │
│  │  │ (Go native)    │  │ (FreeRDP sub)  │  │ (예정)       │   │   │
│  │  │                │  │                │  │              │   │   │
│  │  │ crypto/ssh     │  │ xfreerdp       │  │ RFB 프로토콜  │   │   │
│  │  │ PTY 할당       │  │ /gfx + /rfx    │  │              │   │   │
│  │  │ stdin/stdout   │  │ H.264 캡처     │  │              │   │   │
│  │  └────────────────┘  └────────────────┘  └──────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐                  │
│  │PostgreSQL│  │  Redis   │  │ Audit DB (별도)  │                  │
│  │ (메인DB) │  │(세션/캐시)│  │ (해시 체인)      │                  │
│  └──────────┘  └──────────┘  └──────────────────┘                  │
└──────────────────────────────────────────────────────────────────────┘
                              │
                    방화벽 (프로토콜별 포트 제한)
                              │
┌─────────────────────────────┼──────────────────────────────────────┐
│                      폐쇄망 (대상 서버군)                           │
│                                                                      │
│    [RDP Windows :3389]   [SSH Linux :22]   [VNC Any :5900]         │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 3. 데이터 흐름 (세션 라이프사이클)

```
1. 사용자 인증
   POST /api/auth/login → JWT (15분 access + 7일 refresh)
   
2. 세션 생성
   POST /api/sessions {host_id}
   → Policy Engine 평가 (사용자×호스트×OS×시간)
   → Session DB 저장 + WS 토큰 생성 (Redis, 5분 TTL)
   → Response: {session_id, ws_token, ws_endpoint}

3. WebSocket 연결
   WSS /ws/session/{sessionId}?token={ws_token}
   → WS 토큰 검증 (Redis GETDEL → 1회용)
   → Hub에 RelaySession 등록 (InputCh 채널 생성)

4. 데이터 릴레이 (루프)
   브라우저 ──(key/mouse)──→ WS ──→ Hub.ForwardToRelay() ──→ ssh.Handler.Start()
   ssh.Handler ──(output)──→ Hub.SendToClient() ──→ WS ──→ xterm.js/Cavnas

5. 정책 검사 (모든 프레임)
   Policy Engine → 클립보드/파일전송/프린터 허용 여부
   거부 시 → 감사 로그 기록 + 클라이언트에 Toast 알림

6. 세션 종료
   명시적 종료 or 타임아웃(idle/max) or 관리자 강제 종료
   → WS 연결 해제 → Relay 정리 → Session DB status='ended'
   → Audit Log 기록 (해시 체인 연결)
```

---

## 4. 망분리 토폴로지

```
[업무망 사용자 PC] ──방화벽 단일포트(443)──> [DMZ: SecureGate 서버] ──방화벽──> [폐쇄망 서버군]
                                              │
                                              ├── PostgreSQL (5432, 내부)
                                              ├── Redis (6379, 내부)
                                              └── xfreerdp / SSH (외부로만 outbound)
```

- **사용자→SecureGate:** 443/TCP 단일 포트
- **SecureGate→폐쇄망:** RDP(3389), SSH(22), VNC(5900) 등 프로토콜별 허용
- **DMZ 내부:** PostgreSQL, Redis는 Docker 내부 네트워크로 격리

---

## 5. 암호화 계층

| 계층 | 알고리즘 | 용도 |
|------|----------|------|
| 전송 | TLS 1.3 (TLS_AES_256_GCM_SHA384) | 브라우저 ↔ Nginx |
| 비밀번호 | Argon2id (t=3, m=64MB, p=4) | 사용자 인증 |
| 자격증명 | AES-256-GCM | credentials 테이블 |
| 감사 로그 | SHA-256 해시 체인 | 위변조 방지 |
| JWT | HMAC-SHA256 | API 인증 |

---

## 6. WebSocket 바이너리 프레임 프로토콜

### 프레임 구조
```
[1 byte: FrameType] [4 bytes: PayloadLength (big-endian)] [N bytes: Payload]
```

### FrameType 명세

| 코드 | 타입 | 방향 | 페이로드 |
|------|------|------|----------|
| 0x01 | KEY | C→S | `[down:1][scancode:2]` |
| 0x02 | MOUSE | C→S | `[x:2][y:2][buttons:1]` |
| 0x03 | FRAME | S→C | PNG/H.264 바이너리 |
| 0x04 | CLIPBOARD | 양방향 | 텍스트 데이터 |
| 0x05 | AUDIO | S→C | Opus/Ogg 스트림 |
| 0x06 | RESIZE | C→S | `[width:2][height:2]` |
| 0x07 | PING | 양방향 | Keep-alive (빈 페이로드) |
| 0x08 | FILE_CHUNK | 양방향 | 파일 전송 청크 |
| 0x09 | IME_STATE | S→C | `[state:1]` (한/영 알림) |
| 0x0A | OUTPUT | S→C | SSH 터미널 출력 |

---

## 7. 디렉터리 구조

```
securegate/
├── cmd/server/           # Go API 서버 진입점
│   ├── main.go           #   DI, 라우트 등록, 서버 시작
│   └── doc.go            #   문서/구현 상태 기록
│
├── internal/
│   ├── config/           # 환경변수 기반 설정
│   ├── auth/             # 인증
│   │   ├── model.go      #   User, 로그인/회원가입 요청/응답
│   │   ├── password.go   #   Argon2id 해싱/검증
│   │   ├── jwt.go        #   Access/Refresh 토큰 생성/검증
│   │   ├── mfa.go        #   TOTP 생성/검증 (Google Authenticator)
│   │   ├── service.go    #   로그인/회원가입/비밀번호 변경/MFA
│   │   ├── handler.go    #   REST 핸들러
│   │   └── middleware.go #   JWT Bearer 토큰 검증
│   │
│   ├── rbac/             # 역할 기반 접근 제어 (예정)
│   │
│   ├── host/             # 호스트 관리
│   │   ├── model.go      #   Host, Credential, OSInfo
│   │   ├── service.go    #   호스트 CRUD
│   │   ├── handler.go    #   REST API
│   │   └── osdetect.go   #   SSH/RDP OS 감지
│   │
│   ├── policy/           # 정책 엔진
│   │   ├── model.go      #   Policy, SessionContext, PolicyDecision
│   │   ├── engine.go     #   평가 로직 (user>group>global 우선순위)
│   │   ├── service.go    #   정책 CRUD
│   │   └── handler.go    #   REST API
│   │
│   ├── session/          # 세션 관리
│   │   ├── model.go      #   Session, 생성 요청/응답
│   │   ├── manager.go    #   세션 생명주기, WS 토큰
│   │   └── handler.go    #   REST API
│   │
│   ├── relay/            # 프로토콜 릴레이
│   │   ├── hub.go        #   WS 클라이언트 허브
│   │   ├── ws_handler.go #   WS 업그레이드/라우팅
│   │   ├── framer.go     #   바이너리 프레임 인코딩/디코딩
│   │   ├── clipboard.go  #   클립보드/파일전송/프린터 관리
│   │   │
│   │   ├── ssh/
│   │   │   └── handler.go #  SSH ↔ WS 릴레이
│   │   │
│   │   ├── rdp/
│   │   │   ├── handler.go #  FreeRDP subprocess 관리
│   │   │   └── scanner.go #  Scan code 매핑 (한/영)
│   │   │
│   │   └── vnc/          # (예정)
│   │
│   ├── audit/            # 감사 로그
│   │   ├── logger.go     #   INSERT-only + 해시 체인
│   │   └── handler.go    #   REST API
│   │
│   ├── db/               # 데이터베이스
│   │   ├── postgres.go   #   PG 연결 풀 + 마이그레이션
│   │   └── redis.go      #   Redis 연결
│   │
│   └── middleware/       # 공통 미들웨어
│       ├── logging.go    #   HTTP 요청 로깅
│       ├── cors.go       #   CORS
│       ├── security.go   #   보안 헤더
│       └── ratelimit.go  #   Rate limiting
│
├── web/                   # React 프론트엔드
│   └── src/
│       ├── pages/
│       │   ├── LoginPage.tsx
│       │   ├── ChangePasswordPage.tsx
│       │   ├── DashboardPage.tsx
│       │   ├── SessionPage.tsx        # SSH (xterm.js)
│       │   └── RdpSessionPage.tsx     # RDP (Canvas)
│       ├── store/
│       │   ├── authStore.ts
│       │   └── uiStore.ts
│       ├── lib/
│       │   ├── api.ts
│       │   └── scancode.ts
│       └── i18n/
│           ├── ko.json
│           └── en.json
│
├── migrations/           # SQL 마이그레이션
├── deployments/
│   ├── docker/
│   │   ├── docker-compose.yml
│   │   ├── Dockerfile.server
│   │   └── nginx.conf
│   └── scripts/
│       └── install-linux.sh
│
└── docs/                 # 문서
    ├── architecture.md   # 본 문서
    ├── api-spec.md
    ├── db-schema.md
    ├── development.md
    ├── operations.md
    ├── maintenance.md
    └── compliance-mapping.md
```

# SecureGate API 명세

> **버전:** v0.1.0 | **Base URL:** `https://<host>/api`

---

## 인증 방식

모든 인증 필요 API는 `Authorization: Bearer <access_token>` 헤더를 요구합니다.

- **Access Token**: JWT, 15분 만료
- **Refresh Token**: JWT, 7일 만료, 1회 사용 후 rotation
- **로그아웃** 시 Refresh Token은 Redis 블랙리스트에 등록

---

## 1. 상태 확인

### `GET /api/health`

서비스 상태 확인 (인증 불필요).

**응답 200:**
```json
{
  "status": "ok",
  "db": "connected",
  "redis": "connected"
}
```

**응답 503:**
```json
{
  "status": "unhealthy",
  "db": "error: connection refused"
}
```

---

## 2. 인증 (Auth)

### 2.1 로그인

`POST /api/auth/login`

**요청:**
```json
{
  "username": "admin",
  "password": "SecurePass123!"
}
```

**응답 200 (정상):**
```json
{
  "user": {
    "id": "uuid",
    "username": "admin",
    "email": null,
    "role": "super_admin",
    "status": "active",
    "must_change_password": false,
    "last_login_at": "2026-05-15T14:30:00Z",
    "created_at": "2026-05-15T12:00:00Z",
    "updated_at": "2026-05-15T14:30:00Z"
  },
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi..."
}
```

**응답 200 (MFA 필요):**
```json
{
  "user": { "...": "..." },
  "require_mfa": true,
  "access_token": "<mfa-session-token>"
}
```

**응답 200 (비밀번호 변경 필요):**
```json
{
  "user": { "...": "...", "must_change_password": true },
  "require_password_change": true,
  "access_token": "<jwt-token>"
}
```

**응답 401:**
```json
{"error": "아이디 또는 비밀번호가 올바르지 않습니다"}
```

---

### 2.2 회원가입

`POST /api/auth/signup`

**요청:**
```json
{
  "username": "newuser",
  "email": "user@company.com",
  "password": "SecurePass123!"
}
```

**응답 201:**
```json
{
  "id": "uuid",
  "username": "newuser",
  "email": "user@company.com",
  "role": "user",
  "status": "pending",
  "created_at": "2026-05-15T15:00:00Z",
  "updated_at": "2026-05-15T15:00:00Z"
}
```

**응답 409:**
```json
{"error": "이미 사용 중인 아이디입니다"}
```

---

### 2.3 로그아웃

`POST /api/auth/logout`

**요청:**
```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

**응답 200:**
```json
{"message": "로그아웃되었습니다"}
```

---

### 2.4 비밀번호 변경

`POST /api/auth/password/change` (인증 필요)

**요청:**
```json
{
  "current_password": "old-pass",
  "new_password": "NewSecurePass123!"
}
```

**응답 200:**
```json
{"message": "비밀번호가 변경되었습니다"}
```

---

### 2.5 MFA 설정

`POST /api/auth/mfa/setup` (인증 필요)

**응답 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/SecureGate:admin?secret=JBSWY3DPEHPK3PXP&issuer=SecureGate"
}
```

---

### 2.6 MFA 인증 검증

`POST /api/auth/mfa/verify`

**요청:**
```json
{
  "session_token": "<mfa-session-token>",
  "code": "123456"
}
```

**응답 200:**
```json
{
  "user": { "...": "..." },
  "access_token": "<jwt>",
  "refresh_token": "<jwt>"
}
```

---

### 2.7 MFA 활성화 / 비활성화

`POST /api/auth/mfa/enable` (인증 필요)
`POST /api/auth/mfa/disable` (인증 필요)

---

### 2.8 토큰 갱신

`POST /api/auth/refresh`

**요청:**
```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

**응답 200:**
```json
{
  "user": { "...": "..." },
  "access_token": "<new-jwt>",
  "refresh_token": "<new-rotated-jwt>"
}
```

---

## 3. 내 정보

### `GET /api/me` (인증 필요)

현재 로그인한 사용자 정보 조회. 응답은 로그인의 `user` 객체와 동일.

---

## 4. 호스트 관리

### 4.1 호스트 목록

`GET /api/hosts` (인증 필요)

**응답 200:**
```json
[
  {
    "id": "uuid",
    "name": "운영서버 #1",
    "hostname": "10.0.1.100",
    "protocol": "ssh",
    "port": 22,
    "detected_os": "linux",
    "detected_os_version": "Ubuntu 22.04.3 LTS",
    "status": "active",
    "created_at": "2026-05-15T12:00:00Z",
    "updated_at": "2026-05-15T12:00:00Z"
  }
]
```

### 4.2 호스트 생성

`POST /api/hosts` (인증 필요, 관리자)

**요청:**
```json
{
  "name": "개발 DB 서버",
  "hostname": "192.168.1.50",
  "protocol": "rdp",
  "port": 3389,
  "credential_id": null,
  "host_group_id": null
}
```

### 4.3 호스트 조회/수정/삭제

`GET /api/hosts/{id}`
`PUT /api/hosts/{id}`
`DELETE /api/hosts/{id}`

---

## 5. 정책 관리

### 5.1 정책 목록

`GET /api/policies` (인증 필요)

### 5.2 정책 생성

`POST /api/policies`

**요청:**
```json
{
  "name": "기본 보안 정책",
  "scope_type": "global",
  "clipboard_mode": "both",
  "file_upload_enabled": true,
  "file_download_enabled": true,
  "file_max_size_mb": 100,
  "printer_enabled": false,
  "audio_enabled": true,
  "max_session_minutes": 480,
  "idle_timeout_minutes": 30,
  "priority": 0,
  "enabled": true
}
```

---

## 6. 세션 관리

### 6.1 세션 생성

`POST /api/sessions` (인증 필요)

**요청:**
```json
{
  "host_id": "uuid",
  "width": 80,
  "height": 24
}
```

**응답 201:**
```json
{
  "session": {
    "id": "uuid",
    "user_id": "uuid",
    "host_id": "uuid",
    "started_at": "2026-05-15T14:30:00Z",
    "status": "active"
  },
  "ws_token": "64-char-hex-token",
  "ws_endpoint": "/ws/session/session-uuid-here"
}
```

### 6.2 세션 종료

`DELETE /api/sessions/{id}` (인증 필요)

---

## 7. 감사 로그

### 7.1 로그 조회

`GET /api/audit/logs?action=login&limit=50` (인증 + Auditor 권한 필요)

**응답 200:**
```json
[
  {
    "id": "uuid",
    "event_ts": "2026-05-15T14:30:00Z",
    "actor_user_id": "uuid",
    "actor_ip": "10.0.0.1",
    "action": "login",
    "target_type": "user",
    "target_id": "uuid",
    "detail": {"success": true},
    "verified": true
  }
]
```

### 7.2 체인 검증

`GET /api/audit/verify?limit=100` (인증 + Auditor 권한 필요)

**응답 200:**
```json
{
  "valid": true,
  "count": 100
}
```

---

## 8. WebSocket

### `GET /ws/session/{sessionId}?token={ws_token}`

클라이언트→서버, 서버→클라이언트 모두 바이너리 프레임 사용.

**연결 성공 (초기):**
```json
{
  "type": "connected",
  "session_id": "uuid",
  "host_id": "uuid",
  "user_id": "uuid"
}
```

### 프레임 명세

| Type | Hex | 방향 | 설명 |
|------|-----|------|------|
| KEY | 0x01 | C→S | 키 이벤트 `[down:1][scancode:2]` |
| MOUSE | 0x02 | C→S | 마우스 이벤트 |
| FRAME | 0x03 | S→C | 화면 프레임 (PNG/H.264) |
| CLIPBOARD | 0x04 | 양방향 | 클립보드 텍스트 |
| OUTPUT | 0x0A | S→C | SSH 터미널 출력 |
| RESIZE | 0x06 | C→S | 창 크기 `[w:2][h:2]` |
| PING | 0x07 | 양방향 | Keep-alive |

---

## 에러 응답 형식

모든 에러는 동일한 형식:
```json
{
  "error": "사용자 친화적인 한글 에러 메시지"
}
```

### HTTP 상태 코드

| 코드 | 의미 |
|------|------|
| 200 | 성공 |
| 201 | 생성 성공 |
| 400 | 잘못된 요청 |
| 401 | 인증 실패 / 토큰 만료 |
| 403 | 권한 부족 |
| 404 | 리소스 없음 |
| 409 | 중복 충돌 |
| 429 | Rate limit 초과 |
| 500 | 서버 오류 |
| 503 | 서비스 불가 |

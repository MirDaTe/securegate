# SecureGate DB 스키마

> **DB:** PostgreSQL 15 | **마이그레이션:** `migrations/001_init_schema.up.sql`

---

## ER 다이어그램 (텍스트)

```
users ──< user_group_members >── user_groups
  │
  ├──< sessions >── hosts ──< hosts_groups
  │       │            │
  │       └── policies  └── credentials
  │
  ├──< audit_logs
  │
  └──< file_transfers >── sessions
```

---

## 테이블 명세

### 1. `users` — 사용자

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 사용자 고유 ID |
| `username` | VARCHAR(64) UNIQUE | 로그인 아이디 |
| `email` | VARCHAR(255) UNIQUE | 이메일 (선택) |
| `password_hash` | VARCHAR(255) | Argon2id 해시 |
| `mfa_secret` | TEXT | 암호화된 TOTP secret |
| `mfa_enabled` | BOOLEAN | MFA 활성화 여부 |
| `role` | VARCHAR(20) | super_admin, admin, auditor, user |
| `status` | VARCHAR(20) | active, pending, locked, disabled |
| `must_change_password` | BOOLEAN | 비밀번호 강제 변경 플래그 |
| `password_changed_at` | TIMESTAMPTZ | 마지막 비밀번호 변경 시각 |
| `last_login_at` | TIMESTAMPTZ | 마지막 로그인 시각 |
| `login_fail_count` | INTEGER | 연속 로그인 실패 횟수 |
| `locked_until` | TIMESTAMPTZ | 계정 잠금 해제 시각 |
| `created_at` | TIMESTAMPTZ | 생성일 |
| `updated_at` | TIMESTAMPTZ | 수정일 |

---

### 2. `user_groups` — 사용자 그룹

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 그룹 ID |
| `name` | VARCHAR(128) UNIQUE | 그룹명 |
| `description` | TEXT | 설명 |
| `created_at` | TIMESTAMPTZ | 생성일 |

### `user_group_members` — 그룹 멤버십

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `user_id` | UUID FK → users | 사용자 |
| `group_id` | UUID FK → user_groups | 그룹 |

---

### 3. `hosts` — 접속 대상 호스트

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 호스트 ID |
| `name` | VARCHAR(128) | 표시명 |
| `hostname` | VARCHAR(255) | IP 또는 도메인 |
| `protocol` | VARCHAR(10) | rdp, ssh, vnc, telnet |
| `port` | INTEGER | 포트 번호 |
| `credential_id` | UUID FK → credentials | 자격증명 참조 |
| `host_group_id` | UUID FK → host_groups | 그룹 참조 |
| `detected_os` | VARCHAR(50) | 감지된 OS (windows/linux/macos) |
| `detected_os_version` | VARCHAR(100) | OS 버전 문자열 |
| `last_detected_at` | TIMESTAMPTZ | 마지막 감지 시각 |
| `status` | VARCHAR(20) | active, inactive |
| `created_at` | TIMESTAMPTZ | 생성일 |
| `updated_at` | TIMESTAMPTZ | 수정일 |

---

### 4. `credentials` — 자격증명 (암호화)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 자격증명 ID |
| `type` | VARCHAR(20) | password, private_key, certificate |
| `encrypted_payload` | BYTEA | AES-256-GCM 암호화된 데이터 |
| `created_at` | TIMESTAMPTZ | 생성일 |

---

### 5. `policies` — 접속 정책

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 정책 ID |
| `name` | VARCHAR(255) | 정책명 |
| `scope_type` | VARCHAR(20) | user, group, global |
| `scope_id` | UUID | user_id 또는 group_id |
| `host_id` | UUID FK → hosts | 특정 호스트 대상 (NULL=전체) |
| `clipboard_mode` | VARCHAR(20) | both, host_to_guest, guest_to_host, disabled |
| `file_upload_enabled` | BOOLEAN | 업로드 허용 |
| `file_download_enabled` | BOOLEAN | 다운로드 허용 |
| `file_ext_whitelist` | TEXT[] | 허용 확장자 (빈 배열=전체 허용) |
| `file_ext_blacklist` | TEXT[] | 차단 확장자 |
| `file_max_size_mb` | INTEGER | 최대 파일 크기 (MB) |
| `printer_enabled` | BOOLEAN | 프린터 허용 |
| `audio_enabled` | BOOLEAN | 오디오 허용 |
| `drive_redirect_enabled` | BOOLEAN | 드라이브 매핑 허용 |
| `max_session_minutes` | INTEGER | 최대 세션 시간 (분) |
| `idle_timeout_minutes` | INTEGER | 유휴 타임아웃 (분) |
| `allowed_time_range` | VARCHAR(50) | 허용 시간대 "09:00-18:00" |
| `priority` | INTEGER | 우선순위 (높을수록 먼저 적용) |
| `enabled` | BOOLEAN | 활성화 여부 |
| `created_at` | TIMESTAMPTZ | 생성일 |
| `updated_at` | TIMESTAMPTZ | 수정일 |

---

### 6. `sessions` — 원격 접속 세션

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 세션 ID |
| `user_id` | UUID FK → users | 접속 사용자 |
| `host_id` | UUID FK → hosts | 대상 호스트 |
| `policy_id` | UUID FK → policies | 적용된 정책 |
| `started_at` | TIMESTAMPTZ | 시작 시간 |
| `ended_at` | TIMESTAMPTZ | 종료 시간 |
| `status` | VARCHAR(20) | active, ended, terminated, timeout |
| `client_ip` | INET | 사용자 IP |
| `ws_token_hash` | VARCHAR(64) | WS 토큰의 SHA-256 |
| `recording_path` | TEXT | 녹화 파일 경로 |
| `last_activity_at` | TIMESTAMPTZ | 마지막 활동 시간 |
| `total_bytes_in` | BIGINT | 수신 바이트 |
| `total_bytes_out` | BIGINT | 송신 바이트 |
| `detected_os` | VARCHAR(50) | 세션 중 감지된 OS |
| `detected_os_version` | VARCHAR(100) | OS 버전 |
| `created_at` | TIMESTAMPTZ | 레코드 생성일 |

---

### 7. `audit_logs` — 감사 로그 (해시 체인)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 로그 ID |
| `event_ts` | TIMESTAMPTZ | 이벤트 발생 시간 |
| `actor_user_id` | UUID FK → users | 행위자 |
| `actor_ip` | INET | 행위자 IP |
| `action` | VARCHAR(100) | 액션 타입 |
| `target_type` | VARCHAR(50) | 대상 유형 (user, host, policy, system) |
| `target_id` | UUID | 대상 ID |
| `detail` | JSONB | 상세 정보 (변경 전/후 값 포함) |
| `prev_hash` | VARCHAR(64) | 이전 레코드의 this_hash |
| `this_hash` | VARCHAR(64) | SHA-256(prev_hash + ts + action + detail) |
| `verified` | BOOLEAN | 검증 상태 |

**⚠️ INSERT-only**: 백엔드 계정은 UPDATE/DELETE 권한 없음.

---

### 8. `file_transfers` — 파일 전송 이력

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `id` | UUID PK | 전송 ID |
| `session_id` | UUID FK → sessions | 세션 |
| `direction` | VARCHAR(10) | upload, download |
| `filename` | VARCHAR(512) | 파일명 |
| `file_size` | BIGINT | 파일 크기 (bytes) |
| `sha256` | VARCHAR(64) | 파일 해시 |
| `mime_type` | VARCHAR(255) | MIME 유형 |
| `status` | VARCHAR(20) | completed, denied, failed |
| `transfer_ts` | TIMESTAMPTZ | 전송 시간 |

---

## 인덱스

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_hosts_protocol ON hosts(protocol);
CREATE INDEX idx_hosts_status ON hosts(status);
CREATE INDEX idx_policies_scope ON policies(scope_type, scope_id);
CREATE INDEX idx_policies_host ON policies(host_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_active ON sessions(status) WHERE status = 'active';
CREATE INDEX idx_sessions_host ON sessions(host_id);
CREATE INDEX idx_audit_ts ON audit_logs(event_ts);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_actor ON audit_logs(actor_user_id);
CREATE INDEX idx_file_transfers_session ON file_transfers(session_id);
```

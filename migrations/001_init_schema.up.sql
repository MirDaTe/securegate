-- SecureGate 초기 스키마 마이그레이션
-- Step 1 (Skeleton): 기본 테이블만 생성
-- 이후 Step에서 auth/host/policy/session/audit 테이블 추가

-- ─────────────────────────────────────
-- 사용자 테이블
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    mfa_secret TEXT,                            -- AES-256-GCM 암호화된 TOTP secret
    mfa_enabled BOOLEAN DEFAULT FALSE,
    role VARCHAR(20) NOT NULL DEFAULT 'user',   -- super_admin, admin, auditor, user
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, pending, locked, disabled
    must_change_password BOOLEAN DEFAULT FALSE,
    password_changed_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    login_fail_count INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- ─────────────────────────────────────
-- 사용자 그룹
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_group_members (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

-- ─────────────────────────────────────
-- 자격증명 (암호화 저장)
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL,                  -- password, private_key, certificate
    encrypted_payload BYTEA NOT NULL,           -- AES-256-GCM 암호화
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─────────────────────────────────────
-- 호스트 그룹
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS host_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─────────────────────────────────────
-- 호스트
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS hosts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    protocol VARCHAR(10) NOT NULL,              -- rdp, ssh, vnc, telnet
    port INTEGER NOT NULL,
    credential_id UUID REFERENCES credentials(id) ON DELETE SET NULL,
    host_group_id UUID REFERENCES host_groups(id) ON DELETE SET NULL,
    detected_os VARCHAR(50),                    -- windows, linux, macos
    detected_os_version VARCHAR(100),
    last_detected_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hosts_protocol ON hosts(protocol);
CREATE INDEX IF NOT EXISTS idx_hosts_status ON hosts(status);

-- ─────────────────────────────────────
-- 접속 정책
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    scope_type VARCHAR(20) NOT NULL,            -- user, group, global
    scope_id UUID,                              -- user_id 또는 group_id (global이면 NULL)
    host_id UUID REFERENCES hosts(id) ON DELETE CASCADE,
    -- 클립보드 제어
    clipboard_mode VARCHAR(20) DEFAULT 'both',  -- both, host_to_guest, guest_to_host, disabled
    -- 파일 전송 제어
    file_upload_enabled BOOLEAN DEFAULT TRUE,
    file_download_enabled BOOLEAN DEFAULT TRUE,
    file_ext_whitelist TEXT[],                  -- 허용 확장자 목록 (비어있으면 모두 허용)
    file_ext_blacklist TEXT[],                  -- 차단 확장자 목록
    file_max_size_mb INTEGER DEFAULT 100,       -- 최대 파일 크기 (MB)
    -- 프린터 제어
    printer_enabled BOOLEAN DEFAULT TRUE,
    -- 오디오 제어
    audio_enabled BOOLEAN DEFAULT TRUE,
    -- 드라이브 매핑
    drive_redirect_enabled BOOLEAN DEFAULT TRUE,
    -- 세션 제어
    max_session_minutes INTEGER DEFAULT 480,    -- 최대 세션 시간 (분)
    idle_timeout_minutes INTEGER DEFAULT 30,    -- 유휴 세션 타임아웃
    allowed_time_range VARCHAR(50),             -- 예: "09:00-18:00" (비어있으면 항상 허용)
    priority INTEGER DEFAULT 0,                 -- 우선순위 (높을수록 우선)
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_policies_scope ON policies(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_policies_host ON policies(host_id);

-- ─────────────────────────────────────
-- 세션
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    host_id UUID NOT NULL REFERENCES hosts(id),
    policy_id UUID REFERENCES policies(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, ended, terminated, timeout
    client_ip INET,
    ws_token_hash VARCHAR(64),                  -- SHA-256 of WS token
    recording_path TEXT,
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    total_bytes_in BIGINT DEFAULT 0,
    total_bytes_out BIGINT DEFAULT 0,
    detected_os VARCHAR(50),
    detected_os_version VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_sessions_host ON sessions(host_id);

-- ─────────────────────────────────────
-- 감사 로그 (해시 체인 적용)
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_ip INET,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id UUID,
    detail JSONB NOT NULL DEFAULT '{}',
    prev_hash VARCHAR(64) NOT NULL,
    this_hash VARCHAR(64) NOT NULL,
    verified BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_logs(event_ts);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_user_id);

-- ─────────────────────────────────────
-- 파일 전송 이력
-- ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS file_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id),
    direction VARCHAR(10) NOT NULL,             -- upload, download
    filename VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    sha256 VARCHAR(64),
    mime_type VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'completed', -- completed, denied, failed
    transfer_ts TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_file_transfers_session ON file_transfers(session_id);

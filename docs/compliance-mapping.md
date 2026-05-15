# SecureGate — K-ISMS / ISMS-P 컴플라이언스 매핑

> **목적:** 금융권 컴플라이언스 감사 대응을 위한 요구사항 vs 구현 매핑

---

## 1. 접근 통제 (Access Control)

| 요구사항 | SecureGate 구현 | 근거 |
|----------|----------------|------|
| 사용자 식별 및 인증 | Argon2id 비밀번호 + TOTP MFA | OWASP ASVS V2.1 |
| 역할 기반 접근 제어 (RBAC) | super_admin, admin, auditor, user 4단계 역할 | 최소 권한 원칙 |
| 로그인 실패 잠금 | 5회 실패 → 30분 자동 잠금 | 전자금융감독규정 제15조 |
| 세션 타임아웃 | 유휴 30분, 최대 8시간 (정책 설정 가능) | |
| 비밀번호 정책 | 최소 10자, 복잡도 3종 이상, 90일 변경 | K-ISMS 2.5.1 |

---

## 2. 암호화 (Encryption)

| 요구사항 | SecureGate 구현 | 근거 |
|----------|----------------|------|
| 전송 구간 암호화 | TLS 1.3 전용 (1.2 미만 비활성화) | K-ISMS 2.6.1 |
| 저장 데이터 암호화 | AES-256-GCM (credentials 테이블) | 전자금융감독규정 제20조 |
| 비밀번호 저장 | Argon2id (t=3, m=64MB, p=4) | OWASP 권장 |
| 암호화 키 관리 | 환경변수 기반 (KMS 연동 포인트 마련) | |

---

## 3. 감사 로그 (Audit Logging)

| 요구사항 | SecureGate 구현 | 근거 |
|----------|----------------|------|
| 접속 기록 | 로그인/로그아웃/세션 시작/종료 전부 기록 | K-ISMS 2.10.1 |
| 중요 작업 기록 | 정책 변경, 사용자 관리, 시스템 설정 변경 | 전/후 값 포함 |
| 위변조 방지 | SHA-256 해시 체인 (블록체인 방식) | 전자금융감독규정 제25조 |
| 로그 보존 | 1년 이상 (DB 보관 → WORM 스토리지 연동 가능) | |
| 로그 무결성 검증 | `GET /api/audit/verify` API | 감사 시 증빙 |
| INSERT-only | 백엔드 계정 UPDATE/DELETE 권한 없음 | |

---

## 4. 망분리 (Network Segmentation)

| 요구사항 | SecureGate 구현 | 근거 |
|----------|----------------|------|
| 단일 포트 라우팅 | HTTPS 443 단일 진입 | 방화벽 정책 최소화 |
| 외부 의존성 제로 | CDN, 외부 폰트, 외부 API 호출 없음 | 망분리 환경 오프라인 호환 |
| 오프라인 설치 | USB 복사 → install-linux.sh → 실행 | |
| SSRF 방지 | 릴레이는 등록된 화이트리스트 호스트만 접속 | |

---

## 5. 웹 보안 (Web Security)

| 요구사항 | SecureGate 구현 | 근거 |
|----------|----------------|------|
| CSP | `default-src 'self'` (모든 외부 리소스 차단) | OWASP Top 10 |
| HSTS | `Strict-Transport-Security: max-age=31536000; includeSubDomains` | |
| X-Frame-Options | `DENY` — 클릭재킹 방지 | |
| X-Content-Type-Options | `nosniff` | |
| SQL Injection | 모든 쿼리 Prepared Statement ($1, $2) | OWASP Top 10 A03 |
| XSS | React 기본 escape + CSP | |
| CSRF | SameSite=Strict 쿠키 + Bearer 토큰 | |
| Rate Limiting | IP 기반 분당 100 요청 | |

---

## 6. 전자금융감독규정 주요 조항 매핑

| 조항 | 내용 | SecureGate 대응 |
|------|------|-----------------|
| 제15조 | 접근통제 | RBAC + 세션 타임아웃 + 로그인 잠금 |
| 제17조 | 사용자 인증 | Argon2id + MFA(TOTP) |
| 제20조 | 암호화 | TLS 1.3 + AES-256-GCM + 해시 체인 |
| 제25조 | 로그 기록 및 보존 | INSERT-only 감사 로그 + 해시 체인 |
| 제30조 | 망분리 | 단일 포트(443) + 오프라인 설치 |

---

## 7. 감사 대응 체크리스트

감사 시 아래 항목을 증빙할 수 있습니다:

- [ ] `/api/health` → 서비스 상태
- [ ] `/api/audit/verify?limit=1000` → 로그 무결성 검증 결과
- [ ] `.env` 파일 → 암호화 키 관리 증빙
- [ ] `nginx.conf` → TLS 1.3 설정
- [ ] `go.mod` → 의존성 버전 관리
- [ ] DB 접속 → `INSERT-only` 확인
- [ ] CSP 헤더 확인 → 브라우저 DevTools > Network > Response Headers

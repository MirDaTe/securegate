# SecureGate — 망분리형 웹 원격 데스크톱 게이트웨이

> **Apache Guacamole를 대체하는, 한국 금융권 망분리 환경에 최적화된 웹 기반 원격 데스크톱 관리/접속 플랫폼**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.5-3178C6?logo=typescript)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)

---

## 개요

SecureGate는 **별도 클라이언트 설치 없이 웹 브라우저만으로** RDP, SSH, VNC 원격 접속을 제공하는 게이트웨이입니다. 한국 금융권 망분리 환경을 기본 설계로 하여 **단일 포트(HTTPS 443)** 로 모든 트래픽을 처리합니다.

### Apache Guacamole 대비 차별점

| 항목 | Apache Guacamole | SecureGate |
|------|------------------|------------|
| 한/영 키 전달 | 불완전 | **Scan code 기반 완전 해결** |
| OS 자동 감지 | 제한적 | SSH+RDP 협상 단계에서 자동 감지 |
| 망분리 적합성 | 별도 튜닝 필요 | **외부 호출 제로, 오프라인 설치 지원** |
| 정책 세분화 | 연결 단위 | 사용자×OS×시간대 단위 |
| 감사 로그 | 기본 수준 | **해시 체인 무결성, K-ISMS/ISMS-P 준수** |
| 한글 UI | 영문 위주 | **한국어 기본 (i18n ko/en)** |
| 배포 | WAR/컨테이너 | **단일 바이너리 + Docker Compose** |

---

## 빠른 시작

### 사전 요구사항
- Docker 20.10+ & Docker Compose v2+
- 또는 Go 1.22+ & Node.js 18+ (개발 환경)

### 원클릭 설치 (Docker Compose)

```bash
# 1. 저장소 클론 (또는 USB로 복사)
git clone https://github.com/MirDaTe/securegate.git
cd securegate

# 2. 설치 스크립트 실행 (TLS 인증서 자동 생성, 관리자 비밀번호 자동 발급)
bash deployments/scripts/install-linux.sh

# 3. 웹 브라우저로 접속
# https://<서버IP> 
# 초기 관리자: admin / <화면에 출력된 비밀번호>
```

### 수동 실행 (개발 환경)

```bash
# 백엔드
cp .env.example .env        # 환경변수 파일 생성 및 값 수정
make dev                    # API 서버 (8080) + Vite dev server (5173)

# 프론트엔드만
cd web && npm install && npm run dev
```

---

## 기술 스택

| 계층 | 기술 | 근거 |
|------|------|------|
| **백엔드** | Go 1.22+ (chi router) | 단일 바이너리 배포, 크로스 컴파일, 고동시성 |
| **프론트엔드** | React 18 + TypeScript + Vite | shadcn/ui (CDN 의존성 없음) |
| **UI** | Tailwind CSS + shadcn/ui | 커스터마이징 자유도, 망분리 호환 |
| **DB** | PostgreSQL 15 | 메인 데이터 저장 |
| **캐시** | Redis 7 | 세션, JWT 블랙리스트 |
| **TLS** | Nginx (TLS 1.3 종단) | 단일 포트(443) 진입 |
| **터미널** | xterm.js | SSH 웹 터미널 |
| **RDP** | FreeRDP (subprocess) | GFX/H.264 화면 캡처 |
| **폰트** | Pretendard (자체 호스팅) | CDN 의존성 제로 |
| **암호화** | Argon2id + AES-256-GCM | OWASP 권장 |

---

## 문서

| 문서 | 내용 |
|------|------|
| [📐 아키텍처 설계](docs/architecture.md) | 시스템 구성도, 데이터 흐름, 디렉터리 구조 |
| [🔌 API 명세](docs/api-spec.md) | REST 엔드포인트, WebSocket 프로토콜 |
| [🗄️ DB 스키마](docs/db-schema.md) | 테이블 정의, 인덱스, 관계도 |
| [🛠️ 개발 가이드](docs/development.md) | 환경 설정, 빌드, 테스트, 기여 방법 |
| [🚀 운영 매뉴얼](docs/operations.md) | 시작/중지, 백업/복구, 모니터링 |
| [🔧 유지보수 가이드](docs/maintenance.md) | 업데이트, 디버깅, 트러블슈팅 |
| [📋 컴플라이언스 매핑](docs/compliance-mapping.md) | K-ISMS, ISMS-P, 전자금융감독규정 |

---

## 프로젝트 구조

```
securegate/
├── cmd/server/           # Go API 서버 진입점
├── internal/
│   ├── auth/             # 인증 (Argon2id, JWT, MFA/TOTP)
│   ├── rbac/             # 역할 기반 접근 제어
│   ├── host/             # 호스트 관리 + OS 감지
│   ├── policy/           # 정책 엔진 (우선순위 평가)
│   ├── session/          # 세션 생명주기 관리
│   ├── relay/
│   │   ├── ssh/          # SSH ↔ WebSocket 릴레이
│   │   ├── rdp/          # RDP ↔ WebSocket 릴레이 (FreeRDP)
│   │   └── vnc/          # VNC 릴레이 (예정)
│   ├── audit/            # 감사 로그 + 해시 체인
│   ├── db/               # PostgreSQL + Redis 연결
│   └── middleware/       # 로깅, CORS, 보안, RateLimit
├── web/                  # React 프론트엔드 (Vite + Tailwind)
├── migrations/           # SQL 마이그레이션
└── deployments/          # Docker Compose, 설치 스크립트
```

---

## 보안

- **TLS 1.3** 전용 (1.2 미만 비활성화)
- **Argon2id** 비밀번호 해싱 (OWASP 권장)
- **AES-256-GCM** 자격증명 저장 암호화
- **CSP**: `default-src 'self'` — 외부 리소스 완전 차단
- **Prepared Statements** — SQL Injection 방지
- **INSERT-only** 감사 로그 — 위변조 불가
- **해시 체인** — 로그 무결성 검증
- **SSRF 방지** — 화이트리스트 호스트만 접속

---

## 라이선스

MIT License — 자유로운 사용, 수정, 배포가 가능합니다.

package host

import (
	"time"

	"github.com/google/uuid"
)

// Host — 원격 접속 대상 서버
type Host struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Hostname          string     `json:"hostname"`
	Protocol          string     `json:"protocol"` // rdp, ssh, vnc, telnet
	Port              int        `json:"port"`
	CredentialID      *uuid.UUID `json:"credential_id,omitempty"`
	HostGroupID       *uuid.UUID `json:"host_group_id,omitempty"`
	DetectedOS        *string    `json:"detected_os,omitempty"`
	DetectedOSVersion *string    `json:"detected_os_version,omitempty"`
	LastDetectedAt    *time.Time `json:"last_detected_at,omitempty"`
	Status            string     `json:"status"` // active, inactive
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Credential — 접속 자격증명 (AES-256-GCM 암호화 저장)
type Credential struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"` // password, private_key, certificate
	EncryptedPayload []byte    `json:"-"`    // JSON 노출 금지
	CreatedAt        time.Time `json:"created_at"`
}

// CreateHostRequest — 호스트 생성 요청
type CreateHostRequest struct {
	Name         string     `json:"name"`
	Hostname     string     `json:"hostname"`
	Protocol     string     `json:"protocol"`
	Port         int        `json:"port"`
	CredentialID *uuid.UUID `json:"credential_id,omitempty"`
	HostGroupID  *uuid.UUID `json:"host_group_id,omitempty"`
}

// UpdateHostRequest — 호스트 수정 요청
type UpdateHostRequest struct {
	Name         *string    `json:"name,omitempty"`
	Hostname     *string    `json:"hostname,omitempty"`
	Protocol     *string    `json:"protocol,omitempty"`
	Port         *int       `json:"port,omitempty"`
	CredentialID *uuid.UUID `json:"credential_id,omitempty"`
	HostGroupID  *uuid.UUID `json:"host_group_id,omitempty"`
	Status       *string    `json:"status,omitempty"`
}

// HostGroup — 호스트 그룹
type HostGroup struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// OS 정보
type OSInfo struct {
	OS      string `json:"os"`      // windows, linux, macos
	Version string `json:"version"` // 상세 버전 문자열
}

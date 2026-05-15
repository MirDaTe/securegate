package session

import (
	"time"

	"github.com/google/uuid"
)

// Session — 원격 접속 세션
type Session struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	HostID            uuid.UUID  `json:"host_id"`
	PolicyID          *uuid.UUID `json:"policy_id,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	EndedAt           *time.Time `json:"ended_at,omitempty"`
	Status            string     `json:"status"`
	ClientIP          *string    `json:"client_ip,omitempty"`
	WSTokenHash       *string    `json:"-"` // 노출 금지
	RecordingPath     *string    `json:"recording_path,omitempty"`
	LastActivityAt    time.Time  `json:"last_activity_at"`
	TotalBytesIn      int64      `json:"total_bytes_in"`
	TotalBytesOut     int64      `json:"total_bytes_out"`
	DetectedOS        *string    `json:"detected_os,omitempty"`
	DetectedOSVersion *string    `json:"detected_os_version,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CreateSessionRequest — 세션 생성 요청
type CreateSessionRequest struct {
	HostID    uuid.UUID `json:"host_id"`
	Width     int       `json:"width"`  // 터미널 너비 (cols)
	Height    int       `json:"height"` // 터미널 높이 (rows)
}

// CreateSessionResponse — 세션 생성 응답
type CreateSessionResponse struct {
	Session   Session `json:"session"`
	WSToken   string  `json:"ws_token"`
	WSEndpoint string `json:"ws_endpoint"`
}

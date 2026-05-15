package policy

import (
	"time"

	"github.com/google/uuid"
)

// Policy — 접속 정책 모델
type Policy struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	ScopeType             string     `json:"scope_type"` // user, group, global
	ScopeID               *uuid.UUID `json:"scope_id,omitempty"`
	HostID                *uuid.UUID `json:"host_id,omitempty"`
	ClipboardMode         string     `json:"clipboard_mode"` // both, host_to_guest, guest_to_host, disabled
	FileUploadEnabled     bool       `json:"file_upload_enabled"`
	FileDownloadEnabled   bool       `json:"file_download_enabled"`
	FileExtWhitelist      []string   `json:"file_ext_whitelist,omitempty"`
	FileExtBlacklist      []string   `json:"file_ext_blacklist,omitempty"`
	FileMaxSizeMB         int        `json:"file_max_size_mb"`
	PrinterEnabled        bool       `json:"printer_enabled"`
	AudioEnabled          bool       `json:"audio_enabled"`
	DriveRedirectEnabled  bool       `json:"drive_redirect_enabled"`
	MaxSessionMinutes     int        `json:"max_session_minutes"`
	IdleTimeoutMinutes    int        `json:"idle_timeout_minutes"`
	AllowedTimeRange      *string    `json:"allowed_time_range,omitempty"` // "09:00-18:00"
	Priority              int        `json:"priority"`
	Enabled               bool       `json:"enabled"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// SessionContext — 세션별 정책 평가 컨텍스트
type SessionContext struct {
	UserID   uuid.UUID
	Username string
	Role     string
	HostID   uuid.UUID
	HostOS   *string
	Protocol string
	Now      time.Time
}

// PolicyDecision — 정책 평가 결과
type PolicyDecision struct {
	ClipboardMode        string `json:"clipboard_mode"`
	FileUploadAllowed    bool   `json:"file_upload_allowed"`
	FileDownloadAllowed  bool   `json:"file_download_allowed"`
	FileMaxSizeMB        int    `json:"file_max_size_mb"`
	PrinterAllowed       bool   `json:"printer_allowed"`
	AudioAllowed         bool   `json:"audio_allowed"`
	DriveRedirectAllowed bool   `json:"drive_redirect_allowed"`
	MaxSessionMinutes    int    `json:"max_session_minutes"`
	IdleTimeoutMinutes   int    `json:"idle_timeout_minutes"`
	SessionAllowed       bool   `json:"session_allowed"`
	DenyReason           string `json:"deny_reason,omitempty"`
}

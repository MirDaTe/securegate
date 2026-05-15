package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mirdate/securegate/internal/db"
)

// Engine — 정책 평가 엔진
type Engine struct {
	pool *pgxpool.Pool
}

func NewEngine() *Engine {
	return &Engine{pool: db.Pool()}
}

// Evaluate — 세션 컨텍스트에 대한 정책 평가
// 우선순위: user-specific > group > global > default
func (e *Engine) Evaluate(ctx context.Context, sc SessionContext) *PolicyDecision {
	decision := e.defaultDecision()

	// 1. global 정책 조회
	policies, _ := e.findPolicies(ctx, sc)
	if len(policies) == 0 {
		return decision
	}

	// 2. 최고 우선순위 정책 적용
	p := policies[0] // priority desc 정렬되어 있음

	// 시간대 확인
	if p.AllowedTimeRange != nil && *p.AllowedTimeRange != "" {
		if !e.isTimeAllowed(*p.AllowedTimeRange, sc.Now) {
			decision.SessionAllowed = false
			decision.DenyReason = fmt.Sprintf("허용 시간대(%s)가 아닙니다", *p.AllowedTimeRange)
			return decision
		}
	}

	decision = PolicyDecision{
		SessionAllowed:       p.Enabled,
		ClipboardMode:        p.ClipboardMode,
		FileUploadAllowed:    p.FileUploadEnabled,
		FileDownloadAllowed:  p.FileDownloadEnabled,
		FileMaxSizeMB:        p.FileMaxSizeMB,
		PrinterAllowed:       p.PrinterEnabled,
		AudioAllowed:         p.AudioEnabled,
		DriveRedirectAllowed: p.DriveRedirectEnabled,
		MaxSessionMinutes:    p.MaxSessionMinutes,
		IdleTimeoutMinutes:   p.IdleTimeoutMinutes,
	}

	return &decision
}

// CheckClipboard — 클립보드 작업 허용 여부
func (e *Engine) CheckClipboard(decision PolicyDecision, direction string) bool {
	switch decision.ClipboardMode {
	case "both":
		return true
	case "host_to_guest":
		return direction == "host_to_guest"
	case "guest_to_host":
		return direction == "guest_to_host"
	default:
		return false
	}
}

// CheckFileExtension — 파일 확장자 검증
func (e *Engine) CheckFileExtension(filename string, whitelist, blacklist []string) bool {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])

	// 블랙리스트 우선
	for _, blocked := range blacklist {
		if strings.EqualFold(ext, blocked) {
			return false
		}
	}

	// 화이트리스트가 비어있지 않으면 해당 확장자만 허용
	if len(whitelist) > 0 {
		for _, allowed := range whitelist {
			if strings.EqualFold(ext, allowed) {
				return true
			}
		}
		return false
	}

	return true
}

// ─── 내부 헬퍼 ───

func (e *Engine) defaultDecision() *PolicyDecision {
	return &PolicyDecision{
		SessionAllowed:       true,
		ClipboardMode:        "both",
		FileUploadAllowed:    true,
		FileDownloadAllowed:  true,
		FileMaxSizeMB:        100,
		PrinterAllowed:       false,
		AudioAllowed:         true,
		DriveRedirectAllowed: false,
		MaxSessionMinutes:    480,
		IdleTimeoutMinutes:   30,
	}
}

func (e *Engine) findPolicies(ctx context.Context, sc SessionContext) ([]Policy, error) {
	rows, err := e.pool.Query(ctx,
		`SELECT id, name, scope_type, scope_id, host_id,
		        clipboard_mode, file_upload_enabled, file_download_enabled,
		        file_ext_whitelist, file_ext_blacklist, file_max_size_mb,
		        printer_enabled, audio_enabled, drive_redirect_enabled,
		        max_session_minutes, idle_timeout_minutes, allowed_time_range,
		        priority, enabled, created_at, updated_at
		 FROM policies WHERE enabled=true
		   AND (host_id IS NULL OR host_id=$1)
		   AND (scope_type='global'
		        OR (scope_type='user' AND scope_id=$2)
		        OR (scope_type='group' AND scope_id IN (SELECT group_id FROM user_group_members WHERE user_id=$2)))
		 ORDER BY priority DESC, created_at ASC
		 LIMIT 1`, sc.HostID, sc.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID, &p.Name, &p.ScopeType, &p.ScopeID, &p.HostID,
			&p.ClipboardMode, &p.FileUploadEnabled, &p.FileDownloadEnabled,
			&p.FileExtWhitelist, &p.FileExtBlacklist, &p.FileMaxSizeMB,
			&p.PrinterEnabled, &p.AudioEnabled, &p.DriveRedirectEnabled,
			&p.MaxSessionMinutes, &p.IdleTimeoutMinutes, &p.AllowedTimeRange,
			&p.Priority, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (e *Engine) isTimeAllowed(timeRange string, now time.Time) bool {
	parts := strings.Split(timeRange, "-")
	if len(parts) != 2 {
		return true
	}

	start, err := time.Parse("15:04", strings.TrimSpace(parts[0]))
	if err != nil {
		return true
	}
	end, err := time.Parse("15:04", strings.TrimSpace(parts[1]))
	if err != nil {
		return true
	}

	current := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	startT := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
	endT := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)

	return (current.Equal(startT) || current.After(startT)) && current.Before(endT)
}

var _ = pgx.ErrNoRows

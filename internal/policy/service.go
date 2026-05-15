package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mirdate/securegate/internal/db"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService() *Service {
	return &Service{pool: db.Pool()}
}

func (s *Service) Create(ctx context.Context, p *Policy) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.ClipboardMode == "" { p.ClipboardMode = "both" }
	if p.FileMaxSizeMB == 0 { p.FileMaxSizeMB = 100 }
	if p.MaxSessionMinutes == 0 { p.MaxSessionMinutes = 480 }
	if p.IdleTimeoutMinutes == 0 { p.IdleTimeoutMinutes = 30 }

	_, err := s.pool.Exec(ctx,
		`INSERT INTO policies (id,name,scope_type,scope_id,host_id,clipboard_mode,file_upload_enabled,file_download_enabled,
		 file_ext_whitelist,file_ext_blacklist,file_max_size_mb,printer_enabled,audio_enabled,drive_redirect_enabled,
		 max_session_minutes,idle_timeout_minutes,allowed_time_range,priority,enabled,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		p.ID,p.Name,p.ScopeType,p.ScopeID,p.HostID,p.ClipboardMode,p.FileUploadEnabled,p.FileDownloadEnabled,
		p.FileExtWhitelist,p.FileExtBlacklist,p.FileMaxSizeMB,p.PrinterEnabled,p.AudioEnabled,p.DriveRedirectEnabled,
		p.MaxSessionMinutes,p.IdleTimeoutMinutes,p.AllowedTimeRange,p.Priority,p.Enabled,p.CreatedAt,p.UpdatedAt)
	return err
}

func (s *Service) List(ctx context.Context) ([]Policy, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,name,scope_type,scope_id,host_id,clipboard_mode,file_upload_enabled,file_download_enabled,
		 file_ext_whitelist,file_ext_blacklist,file_max_size_mb,printer_enabled,audio_enabled,drive_redirect_enabled,
		 max_session_minutes,idle_timeout_minutes,allowed_time_range,priority,enabled,created_at,updated_at
		 FROM policies ORDER BY priority DESC`)
	if err != nil { return nil, err }
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID,&p.Name,&p.ScopeType,&p.ScopeID,&p.HostID,&p.ClipboardMode,&p.FileUploadEnabled,&p.FileDownloadEnabled,
			&p.FileExtWhitelist,&p.FileExtBlacklist,&p.FileMaxSizeMB,&p.PrinterEnabled,&p.AudioEnabled,&p.DriveRedirectEnabled,
			&p.MaxSessionMinutes,&p.IdleTimeoutMinutes,&p.AllowedTimeRange,&p.Priority,&p.Enabled,&p.CreatedAt,&p.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	if policies == nil { policies = []Policy{} }
	return policies, rows.Err()
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, p *Policy) error {
	p.UpdatedAt = time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE policies SET name=$1,scope_type=$2,scope_id=$3,host_id=$4,clipboard_mode=$5,
		 file_upload_enabled=$6,file_download_enabled=$7,file_ext_whitelist=$8,file_ext_blacklist=$9,
		 file_max_size_mb=$10,printer_enabled=$11,audio_enabled=$12,drive_redirect_enabled=$13,
		 max_session_minutes=$14,idle_timeout_minutes=$15,allowed_time_range=$16,priority=$17,enabled=$18,updated_at=$19
		 WHERE id=$20`,
		p.Name,p.ScopeType,p.ScopeID,p.HostID,p.ClipboardMode,p.FileUploadEnabled,p.FileDownloadEnabled,
		p.FileExtWhitelist,p.FileExtBlacklist,p.FileMaxSizeMB,p.PrinterEnabled,p.AudioEnabled,p.DriveRedirectEnabled,
		p.MaxSessionMinutes,p.IdleTimeoutMinutes,p.AllowedTimeRange,p.Priority,p.Enabled,p.UpdatedAt,id)
	if err != nil { return fmt.Errorf("정책 수정 실패: %w", err) }
	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM policies WHERE id=$1`, id)
	return err
}

var _ = pgx.ErrNoRows

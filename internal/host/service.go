package host

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mirdate/securegate/internal/db"
)

// Service — 호스트 관리 비즈니스 로직
type Service struct {
	pool    *pgxpool.Pool
	detector *OSDetector
}

// NewService — 생성자
func NewService() *Service {
	return &Service{
		pool:    db.Pool(),
		detector: NewOSDetector(),
	}
}

// CreateHost — 호스트 생성
func (s *Service) CreateHost(ctx context.Context, req CreateHostRequest) (*Host, error) {
	host := &Host{
		ID:        uuid.New(),
		Name:      req.Name,
		Hostname:  req.Hostname,
		Protocol:  req.Protocol,
		Port:      req.Port,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.CredentialID != nil {
		host.CredentialID = req.CredentialID
	}
	if req.HostGroupID != nil {
		host.HostGroupID = req.HostGroupID
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO hosts (id, name, hostname, protocol, port, credential_id, host_group_id, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		host.ID, host.Name, host.Hostname, host.Protocol, host.Port,
		host.CredentialID, host.HostGroupID, host.Status, host.CreatedAt, host.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("호스트 생성 실패: %w", err)
	}

	return host, nil
}

// GetHost — 단일 호스트 조회
func (s *Service) GetHost(ctx context.Context, id uuid.UUID) (*Host, error) {
	var h Host
	var credID, groupID *uuid.UUID
	var detectedOS, detectedOSVer *string
	var lastDetected *time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT id, name, hostname, protocol, port, credential_id, host_group_id,
		        detected_os, detected_os_version, last_detected_at, status, created_at, updated_at
		 FROM hosts WHERE id=$1`, id,
	).Scan(&h.ID, &h.Name, &h.Hostname, &h.Protocol, &h.Port, &credID, &groupID,
		&detectedOS, &detectedOSVer, &lastDetected, &h.Status, &h.CreatedAt, &h.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("호스트를 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("호스트 조회 실패: %w", err)
	}

	h.CredentialID = credID
	h.HostGroupID = groupID
	h.DetectedOS = detectedOS
	h.DetectedOSVersion = detectedOSVer
	h.LastDetectedAt = lastDetected

	return &h, nil
}

// ListHosts — 호스트 목록 조회 (사용자 접속 가능 호스트만)
func (s *Service) ListHosts(ctx context.Context) ([]Host, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, hostname, protocol, port, credential_id, host_group_id,
		        detected_os, detected_os_version, last_detected_at, status, created_at, updated_at
		 FROM hosts WHERE status='active' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("호스트 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var h Host
		var credID, groupID *uuid.UUID
		var detectedOS, detectedOSVer *string
		var lastDetected *time.Time

		if err := rows.Scan(&h.ID, &h.Name, &h.Hostname, &h.Protocol, &h.Port, &credID, &groupID,
			&detectedOS, &detectedOSVer, &lastDetected, &h.Status, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}

		h.CredentialID = credID
		h.HostGroupID = groupID
		h.DetectedOS = detectedOS
		h.DetectedOSVersion = detectedOSVer
		h.LastDetectedAt = lastDetected
		hosts = append(hosts, h)
	}

	if hosts == nil {
		hosts = []Host{}
	}
	return hosts, rows.Err()
}

// UpdateHost — 호스트 수정
func (s *Service) UpdateHost(ctx context.Context, id uuid.UUID, req UpdateHostRequest) (*Host, error) {
	host, err := s.GetHost(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		host.Name = *req.Name
	}
	if req.Hostname != nil {
		host.Hostname = *req.Hostname
	}
	if req.Protocol != nil {
		host.Protocol = *req.Protocol
	}
	if req.Port != nil {
		host.Port = *req.Port
	}
	if req.CredentialID != nil {
		host.CredentialID = req.CredentialID
	}
	if req.HostGroupID != nil {
		host.HostGroupID = req.HostGroupID
	}
	if req.Status != nil {
		host.Status = *req.Status
	}
	host.UpdatedAt = time.Now()

	_, err = s.pool.Exec(ctx,
		`UPDATE hosts SET name=$1, hostname=$2, protocol=$3, port=$4, credential_id=$5,
		 host_group_id=$6, status=$7, updated_at=$8 WHERE id=$9`,
		host.Name, host.Hostname, host.Protocol, host.Port,
		host.CredentialID, host.HostGroupID, host.Status, host.UpdatedAt, host.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("호스트 수정 실패: %w", err)
	}

	return host, nil
}

// DeleteHost — 호스트 삭제
func (s *Service) DeleteHost(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM hosts WHERE id=$1`, id)
	return err
}

// UpdateOSDetection — OS 감지 결과 업데이트
func (s *Service) UpdateOSDetection(ctx context.Context, hostID uuid.UUID, info *OSInfo) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE hosts SET detected_os=$1, detected_os_version=$2, last_detected_at=$3, updated_at=$4 WHERE id=$5`,
		info.OS, info.Version, now, now, hostID,
	)
	return err
}

// CreateCredential — 자격증명 저장 (AES-256-GCM 암호화)
func (s *Service) CreateCredential(ctx context.Context, credType string, encryptedPayload []byte) (*Credential, error) {
	c := &Credential{
		ID:               uuid.New(),
		Type:             credType,
		EncryptedPayload: encryptedPayload,
		CreatedAt:        time.Now(),
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO credentials (id, type, encrypted_payload, created_at) VALUES ($1,$2,$3,$4)`,
		c.ID, c.Type, c.EncryptedPayload, c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("자격증명 저장 실패: %w", err)
	}
	return c, nil
}

// GetCredential — 자격증명 조회
func (s *Service) GetCredential(ctx context.Context, id uuid.UUID) (*Credential, error) {
	var c Credential
	err := s.pool.QueryRow(ctx,
		`SELECT id, type, encrypted_payload, created_at FROM credentials WHERE id=$1`, id,
	).Scan(&c.ID, &c.Type, &c.EncryptedPayload, &c.CreatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("자격증명을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

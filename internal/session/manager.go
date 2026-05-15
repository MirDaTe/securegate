package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mirdate/securegate/internal/db"
)

// Manager — 세션 생명주기 관리
type Manager struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewManager() *Manager {
	return &Manager{
		pool:  db.Pool(),
		redis: db.Redis(),
	}
}

// CreateSession — 새 세션 생성 + WS 토큰 발급
func (m *Manager) CreateSession(ctx context.Context, userID, hostID uuid.UUID, clientIP string) (*CreateSessionResponse, error) {
	id := uuid.New()
	wsToken := generateWSToken()

	// WS 토큰을 Redis에 저장 (TTL: 5분 — 세션 연결 전까지 유효)
	hash := sha256Hash(wsToken)
	key := fmt.Sprintf("ws_session:%s", hash[:16])
	err := m.redis.Set(ctx, key, id.String()+":"+userID.String()+":"+hostID.String(), 5*time.Minute).Err()
	if err != nil {
		return nil, fmt.Errorf("WS 토큰 저장 실패: %w", err)
	}

	now := time.Now()
	var wsTokenHash *string
	th := sha256Hash(wsToken)
	wsTokenHash = &th

	_, err = m.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, host_id, started_at, status, client_ip, ws_token_hash, last_activity_at, created_at)
		 VALUES ($1,$2,$3,$4,'active',$5,$6,$7,$8)`,
		id, userID, hostID, now, clientIP, wsTokenHash, now, now,
	)
	if err != nil {
		m.redis.Del(ctx, key)
		return nil, fmt.Errorf("세션 생성 실패: %w", err)
	}

	session := Session{
		ID:             id,
		UserID:         userID,
		HostID:         hostID,
		StartedAt:      now,
		Status:         "active",
		LastActivityAt: now,
		CreatedAt:      now,
	}

	return &CreateSessionResponse{
		Session:   session,
		WSToken:   wsToken,
		WSEndpoint: "/ws/session/" + id.String(),
	}, nil
}

// ValidateWSToken — WS 연결 시 토큰 검증 → 세션 정보 반환
func (m *Manager) ValidateWSToken(ctx context.Context, wsToken string) (uuid.UUID, uuid.UUID, error) {
	hash := sha256Hash(wsToken)
	key := fmt.Sprintf("ws_session:%s", hash[:16])

	val, err := m.redis.GetDel(ctx, key).Result()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("만료되었거나 유효하지 않은 WS 토큰입니다")
	}

	// Redis 값: sessionID:userID:hostID
	var sessionID, userID, hostIDStr string
	_, err = fmt.Sscanf(val, "%36[^:]:%36[^:]:%s",
		&sessionID, &userID, &hostIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("잘못된 세션 데이터입니다")
	}

	uid, _ := uuid.Parse(userID)
	hid, _ := uuid.Parse(hostIDStr)
	return uid, hid, nil
}

// EndSession — 세션 종료
func (m *Manager) EndSession(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	_, err := m.pool.Exec(ctx,
		`UPDATE sessions SET ended_at=$1, status='ended', last_activity_at=$2 WHERE id=$3`,
		now, now, sessionID,
	)
	return err
}

// TerminateSession — 관리자에 의한 강제 종료
func (m *Manager) TerminateSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := m.pool.Exec(ctx,
		`UPDATE sessions SET status='terminated', ended_at=$1 WHERE id=$2`,
		time.Now(), sessionID,
	)
	return err
}

// UpdateActivity — 세션 활동 시간 갱신
func (m *Manager) UpdateActivity(ctx context.Context, sessionID uuid.UUID, bytesIn, bytesOut int) {
	m.pool.Exec(ctx,
		`UPDATE sessions SET last_activity_at=$1, total_bytes_in=total_bytes_in+$2, total_bytes_out=total_bytes_out+$3 WHERE id=$4`,
		time.Now(), bytesIn, bytesOut, sessionID,
	)
}

// GetSession — 세션 조회
func (m *Manager) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	var s Session
	var policyID *uuid.UUID
	var clientIP, wsTokenHash, recordingPath, detectedOS, detectedOSVer *string
	var endedAt, lastDetected *time.Time

	err := m.pool.QueryRow(ctx,
		`SELECT id,user_id,host_id,policy_id,started_at,ended_at,status,client_ip,ws_token_hash,
		 recording_path,last_activity_at,total_bytes_in,total_bytes_out,detected_os,detected_os_version,created_at
		 FROM sessions WHERE id=$1`, id,
	).Scan(&s.ID,&s.UserID,&s.HostID,&policyID,&s.StartedAt,&endedAt,&s.Status,&clientIP,&wsTokenHash,
		&recordingPath,&s.LastActivityAt,&s.TotalBytesIn,&s.TotalBytesOut,&detectedOS,&detectedOSVer,&s.CreatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("세션을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}

	s.PolicyID = policyID
	s.EndedAt = endedAt
	s.ClientIP = clientIP
	s.WSTokenHash = wsTokenHash
	s.RecordingPath = recordingPath
	s.DetectedOS = detectedOS
	s.DetectedOSVersion = detectedOSVer

	return &s, nil
}

func generateWSToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

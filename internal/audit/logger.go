package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mirdate/securegate/internal/db"
)

// Action types
const (
	ActionLogin       = "login"
	ActionLogout      = "logout"
	ActionSessionStart = "session_start"
	ActionSessionEnd   = "session_end"
	ActionPolicyChange = "policy_change"
	ActionUserCreate   = "user_create"
	ActionUserDelete   = "user_delete"
	ActionClipboard    = "clipboard"
	ActionFileTransfer = "file_transfer"
	ActionPrint        = "print"
	ActionSystemConfig = "system_config"
)

// Logger — 감사 로거 (INSERT-only + 해시 체인)
type Logger struct {
	pool *pgxpool.Pool
}

func NewLogger() *Logger {
	return &Logger{pool: db.Pool()}
}

// LogEntry — 감사 로그 엔트리
type LogEntry struct {
	ID         uuid.UUID       `json:"id"`
	EventTS    time.Time       `json:"event_ts"`
	ActorUserID *uuid.UUID     `json:"actor_user_id,omitempty"`
	ActorIP    *string         `json:"actor_ip,omitempty"`
	Action     string          `json:"action"`
	TargetType *string         `json:"target_type,omitempty"`
	TargetID   *uuid.UUID      `json:"target_id,omitempty"`
	Detail     json.RawMessage `json:"detail"`
	PrevHash   string          `json:"-"`
	ThisHash   string          `json:"-"`
	Verified   bool            `json:"verified"`
}

// Log — 감사 로그 기록 (INSERT-only)
func (l *Logger) Log(ctx context.Context, entry *LogEntry) error {
	// 마지막 로그의 해시 가져오기
	var prevHash string
	err := l.pool.QueryRow(ctx,
		"SELECT this_hash FROM audit_logs ORDER BY event_ts DESC, id DESC LIMIT 1",
	).Scan(&prevHash)
	if err != nil {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000" // 제네시스 블록
	}

	entry.ID = uuid.New()
	entry.EventTS = time.Now()
	if entry.Detail == nil {
		entry.Detail = json.RawMessage("{}")
	}
	entry.PrevHash = prevHash
	entry.ThisHash = computeHash(prevHash, entry.EventTS, entry.Action, string(entry.Detail))

	_, err = l.pool.Exec(ctx,
		`INSERT INTO audit_logs (id, event_ts, actor_user_id, actor_ip, action, target_type, target_id, detail, prev_hash, this_hash)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		entry.ID, entry.EventTS, entry.ActorUserID, entry.ActorIP,
		entry.Action, entry.TargetType, entry.TargetID,
		entry.Detail, entry.PrevHash, entry.ThisHash,
	)
	return err
}

// VerifyChain — 해시 체인 무결성 검증 (마지막 N개)
func (l *Logger) VerifyChain(ctx context.Context, limit int) (bool, int, error) {
	rows, err := l.pool.Query(ctx,
		`SELECT id, event_ts, action, detail, prev_hash, this_hash
		 FROM audit_logs ORDER BY event_ts, id LIMIT $1`, limit)
	if err != nil { return false, 0, err }
	defer rows.Close()

	var prev *LogEntry
	count := 0
	valid := true

	for rows.Next() {
		var entry LogEntry
		rows.Scan(&entry.ID, &entry.EventTS, &entry.Action, &entry.Detail, &entry.PrevHash, &entry.ThisHash)
		count++

		if prev != nil {
			if entry.PrevHash != prev.ThisHash {
				valid = false
				break
			}
		}
		expected := computeHash(entry.PrevHash, entry.EventTS, entry.Action, string(entry.Detail))
		if entry.ThisHash != expected {
			valid = false
			break
		}
		prev = &entry
	}
	return valid, count, rows.Err()
}

// QueryLogs — 감사 로그 조회
func (l *Logger) QueryLogs(ctx context.Context, action string, limit int) ([]LogEntry, error) {
	rows, err := l.pool.Query(ctx,
		`SELECT id, event_ts, actor_user_id, actor_ip, action, target_type, target_id, detail, verified
		 FROM audit_logs WHERE ($1='' OR action=$1) ORDER BY event_ts DESC LIMIT $2`,
		action, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var e LogEntry
		rows.Scan(&e.ID, &e.EventTS, &e.ActorUserID, &e.ActorIP, &e.Action, &e.TargetType, &e.TargetID, &e.Detail, &e.Verified)
		logs = append(logs, e)
	}
	if logs == nil { logs = []LogEntry{} }
	return logs, rows.Err()
}

func computeHash(prevHash string, ts time.Time, action string, detail string) string {
	data := fmt.Sprintf("%s|%d|%s|%s", prevHash, ts.UnixNano(), action, detail)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

package relay

import (
	"fmt"
	"strings"
	"time"
)

// ClipboardManager — 클립보드 정책 기반 제어
type ClipboardManager struct {
	mode string // both, host_to_guest, guest_to_host, disabled
}

func NewClipboardManager(mode string) *ClipboardManager {
	return &ClipboardManager{mode: mode}
}

func (c *ClipboardManager) Allow(direction string) bool {
	switch c.mode {
	case "both": return true
	case "host_to_guest": return direction == "host_to_guest"
	case "guest_to_host": return direction == "guest_to_host"
	default: return false
	}
}

// FileTransferManager — 파일 전송 정책 기반 제어
type FileTransferManager struct {
	uploadEnabled   bool
	downloadEnabled bool
	whitelist       []string
	blacklist       []string
	maxSizeMB       int
}

func NewFileTransferManager(upload, download bool, whitelist, blacklist []string, maxMB int) *FileTransferManager {
	return &FileTransferManager{upload, download, whitelist, blacklist, maxMB}
}

func (f *FileTransferManager) AllowUpload(filename string, sizeBytes int64) (bool, string) {
	if !f.uploadEnabled { return false, "업로드가 비활성화되었습니다" }
	return f.checkFile(filename, sizeBytes)
}

func (f *FileTransferManager) AllowDownload(filename string, sizeBytes int64) (bool, string) {
	if !f.downloadEnabled { return false, "다운로드가 비활성화되었습니다" }
	return f.checkFile(filename, sizeBytes)
}

func (f *FileTransferManager) checkFile(filename string, sizeBytes int64) (bool, string) {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	for _, blocked := range f.blacklist {
		if strings.EqualFold(ext, blocked) {
			return false, fmt.Sprintf("차단된 확장자: .%s", ext)
		}
	}
	if len(f.whitelist) > 0 {
		for _, allowed := range f.whitelist {
			if strings.EqualFold(ext, allowed) { break }
		}
		return false, fmt.Sprintf("허용되지 않은 확장자: .%s", ext)
	}
	if f.maxSizeMB > 0 && sizeBytes > int64(f.maxSizeMB)*1024*1024 {
		return false, fmt.Sprintf("파일 크기 초과 (최대 %dMB)", f.maxSizeMB)
	}
	return true, ""
}

// PrinterManager — 프린터 리다이렉트 제어
type PrinterManager struct {
	enabled bool
}

func NewPrinterManager(enabled bool) *PrinterManager {
	return &PrinterManager{enabled}
}

func (p *PrinterManager) Allow() bool { return p.enabled }

// SessionIdleTracker — 유휴 세션 추적
type SessionIdleTracker struct {
	lastActivity time.Time
	idleTimeout  time.Duration
	maxDuration  time.Duration
	startTime    time.Time
}

func NewSessionIdleTracker(idleTimeout, maxDuration time.Duration) *SessionIdleTracker {
	now := time.Now()
	return &SessionIdleTracker{lastActivity: now, idleTimeout: idleTimeout, maxDuration: maxDuration, startTime: now}
}

func (s *SessionIdleTracker) Activity() { s.lastActivity = time.Now() }
func (s *SessionIdleTracker) IsIdle() bool { return time.Since(s.lastActivity) > s.idleTimeout }
func (s *SessionIdleTracker) IsExpired() bool { return time.Since(s.startTime) > s.maxDuration }

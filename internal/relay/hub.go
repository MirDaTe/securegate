package relay

import (
	"log"
	"sync"
)

// Hub — WebSocket 클라이언트 ↔ 릴레이 중계
type Hub struct {
	mu       sync.RWMutex
	sessions map[string]*RelaySession
}

// RelaySession — 단일 세션 릴레이
type RelaySession struct {
	SessionID string
	UserID    string
	InputCh   chan []byte    // 클라이언트 → 릴레이 (stdin)
	SendFn    func([]byte) error
	CloseFn   func()
}

func NewHub() *Hub {
	return &Hub{sessions: make(map[string]*RelaySession)}
}

func (h *Hub) Register(sessionID string, rs *RelaySession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[sessionID] = rs
	log.Printf("Hub 등록: session=%s", sessionID)
}

func (h *Hub) Unregister(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if rs, ok := h.sessions[sessionID]; ok {
		close(rs.InputCh)
		delete(h.sessions, sessionID)
	}
	log.Printf("Hub 해제: session=%s", sessionID)
}

func (h *Hub) SendToClient(sessionID string, data []byte) error {
	h.mu.RLock()
	rs, ok := h.sessions[sessionID]
	h.mu.RUnlock()
	if !ok { return nil }
	return rs.SendFn(data)
}

func (h *Hub) ForwardToRelay(sessionID string, data []byte) {
	h.mu.RLock()
	rs, ok := h.sessions[sessionID]
	h.mu.RUnlock()
	if !ok { return }
	select {
	case rs.InputCh <- data:
	default:
		log.Printf("입력 채널 가득 참: session=%s", sessionID)
	}
}

func (h *Hub) ActiveCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

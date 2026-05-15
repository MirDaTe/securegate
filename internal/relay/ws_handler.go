package relay

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/mirdate/securegate/internal/session"
)

type WSHandler struct {
	hub       *Hub
	mgr       *session.Manager
	upgrader  websocket.Upgrader
}

func NewWSHandler(hub *Hub, mgr *session.Manager) *WSHandler {
	return &WSHandler{
		hub: hub,
		mgr: mgr,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ServeWS — WS 연결 수락 후 세션 릴레이 시작
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.PathValue("sessionId")
	wsToken := r.URL.Query().Get("token")

	if sessionIDStr == "" || wsToken == "" {
		http.Error(w, "sessionId와 token이 필요합니다", http.StatusBadRequest)
		return
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		http.Error(w, "잘못된 sessionId", http.StatusBadRequest)
		return
	}

	userID, hostID, err := h.mgr.ValidateWSToken(r.Context(), wsToken)
	if err != nil {
		http.Error(w, "유효하지 않은 토큰: "+err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS 업그레이드 실패: %v", err)
		return
	}

	sid := sessionID.String()

	// RelaySession 생성
	inputCh := make(chan []byte, 100)
	rs := &RelaySession{
		SessionID: sid,
		UserID:    userID.String(),
		InputCh:   inputCh,
		SendFn: func(data []byte) error {
			return conn.WriteMessage(websocket.BinaryMessage, data)
		},
		CloseFn: func() {
			conn.Close()
		},
	}
	h.hub.Register(sid, rs)
	defer func() {
		h.hub.Unregister(sid)
		h.mgr.EndSession(r.Context(), sessionID)
	}()

	// 클라이언트 → 서버 읽기 루프
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// 프레임 타입 파싱
			frameType, payload, err := DecodeFrame(data)
			if err != nil {
				continue
			}

			switch frameType {
			case FrameKey:
				// 키보드 입력 → 릴레이로
				h.hub.ForwardToRelay(sid, data)
			case FrameResize:
				// 크기 변경 → 릴레이로
				h.hub.ForwardToRelay(sid, data)
			case FramePing:
				conn.WriteMessage(websocket.BinaryMessage, EncodePing())
			default:
				h.hub.ForwardToRelay(sid, payload)
			}
		}
	}()

	// 응답: 연결 성공
	ack := map[string]interface{}{
		"type":      "connected",
		"session_id": sid,
		"host_id":   hostID.String(),
		"user_id":   userID.String(),
	}
	resp, _ := json.Marshal(ack)
	conn.WriteMessage(websocket.TextMessage, resp)

	log.Printf("WS 시작: session=%s user=%s host=%s", sid, userID, hostID)
}

var (
	_ = json.Marshal
	_ = uuid.Parse
)

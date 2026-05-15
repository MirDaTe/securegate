package relay

import (
	"encoding/binary"
	"fmt"
)

// 프레임 타입 상수
const (
	FrameKey       = 0x01
	FrameMouse     = 0x02
	FrameFrame     = 0x03
	FrameClipboard = 0x04
	FrameAudio     = 0x05
	FrameResize    = 0x06
	FramePing      = 0x07
	FrameFileChunk = 0x08
	FrameIMEState  = 0x09
	FrameOutput    = 0x0A // SSH 터미널 출력 (추가)
)

// KeyEvent — 키보드 이벤트
type KeyEvent struct {
	Down     bool   `json:"down"`
	ScanCode uint16 `json:"scan_code"`
}

// ResizeEvent — 창 크기 변경
type ResizeEvent struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// EncodeKey — 키보드 이벤트 → 바이너리 프레임
func EncodeKey(down bool, scanCode uint16) []byte {
	downByte := byte(0)
	if down { downByte = 1 }
	payload := []byte{downByte, byte(scanCode >> 8), byte(scanCode & 0xff)}
	return EncodeFrame(FrameKey, payload)
}

// DecodeKey — 바이너리 프레임 → 키보드 이벤트
func DecodeKey(data []byte) (KeyEvent, error) {
	if len(data) < 3 {
		return KeyEvent{}, fmt.Errorf("잘못된 key 프레임 길이: %d", len(data))
	}
	return KeyEvent{
		Down:     data[0] == 1,
		ScanCode: uint16(data[1])<<8 | uint16(data[2]),
	}, nil
}

// EncodeOutput — SSH 출력 → 바이너리 프레임
func EncodeOutput(data []byte) []byte {
	return EncodeFrame(FrameOutput, data)
}

// EncodeResize — 크기 변경 → 바이너리 프레임
func EncodeResize(width, height int) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], uint16(width))
	binary.BigEndian.PutUint16(payload[2:4], uint16(height))
	return EncodeFrame(FrameResize, payload)
}

// DecodeResize — 바이너리 프레임 → 크기 변경
func DecodeResize(data []byte) (ResizeEvent, error) {
	if len(data) < 4 {
		return ResizeEvent{}, fmt.Errorf("잘못된 resize 프레임 길이")
	}
	return ResizeEvent{
		Width:  int(binary.BigEndian.Uint16(data[0:2])),
		Height: int(binary.BigEndian.Uint16(data[2:4])),
	}, nil
}

// EncodePing — ping 프레임
func EncodePing() []byte {
	return EncodeFrame(FramePing, []byte{})
}

// EncodeFrame — [type:1][length:4][payload:N] 인코딩
func EncodeFrame(frameType byte, payload []byte) []byte {
	buf := make([]byte, 1+4+len(payload))
	buf[0] = frameType
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(payload)))
	copy(buf[5:], payload)
	return buf
}

// DecodeFrame — 프레임 타입과 페이로드 추출
func DecodeFrame(data []byte) (byte, []byte, error) {
	if len(data) < 5 {
		return 0, nil, fmt.Errorf("프레임 최소 길이 부족: %d", len(data))
	}
	frameType := data[0]
	length := binary.BigEndian.Uint32(data[1:5])
	if len(data) < 5+int(length) {
		return 0, nil, fmt.Errorf("프레임 페이로드 부족: expected %d, got %d", 5+length, len(data))
	}
	return frameType, data[5 : 5+length], nil
}

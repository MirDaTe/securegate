package rdp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mirdate/securegate/internal/relay"
)

// Handler — FreeRDP subprocess 기반 RDP 릴레이
type Handler struct {
	hub      *relay.Hub
	hostInfo *RDPHostInfo
}

// RDPHostInfo — RDP 접속 정보
type RDPHostInfo struct {
	Address  string
	Port     int
	Username string
	Password string
	Width    int
	Height   int
}

// NewHandler — 생성자
func NewHandler(hub *relay.Hub, hostInfo *RDPHostInfo) *Handler {
	return &Handler{hub: hub, hostInfo: hostInfo}
}

// Start — FreeRDP subprocess 시작 + 출력 → WS 릴레이
// inputCh: 클라이언트 키보드/마우스 이벤트
func (h *Handler) Start(ctx context.Context, sessionID string, inputCh <-chan []byte) error {
	args := []string{
		fmt.Sprintf("/v:%s:%d", h.hostInfo.Address, h.hostInfo.Port),
		fmt.Sprintf("/u:%s", h.hostInfo.Username),
		fmt.Sprintf("/p:%s", h.hostInfo.Password),
		fmt.Sprintf("/w:%d", h.hostInfo.Width),
		fmt.Sprintf("/h:%d", h.hostInfo.Height),
		"/cert:ignore",            // 자체 서명 인증서 허용
		"/sec:tls",                // TLS 보안
		"/network:auto",           // 자동 네트워크 감지
		"/gdi:sw",                 // 소프트웨어 렌더링 (화면 캡처용)
		"/gfx",                    // 그래픽 파이프라인 사용
		"/rfx",                    // RemoteFX 코덱
		"/clipboard",              // 클립보드 활성화
		"/fonts",                  // 폰트 스무딩
		"/aero",                   // Aero 테마
		"/window-drag",            // 창 드래그 지원
		"/menu-anims",             // 메뉴 애니메이션
		"/themes",                 // 테마 활성화
		"/wallpaper",              // 배경 화면
		"+async-input",            // 비동기 입력
		"+async-update",           // 비동기 업데이트
		"+bitmap-cache",           // 비트맵 캐싱
		"+offscreen-cache",        // 오프스크린 캐싱
		"+glyph-cache",            // 글리프 캐싱
	}

	cmd := exec.CommandContext(ctx, "xfreerdp", args...)

	// stderr: FreeRDP 로그 (OS 감지 정보 포함)
	stderr, _ := cmd.StderrPipe()
	stdin, _ := cmd.StdinPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("FreeRDP 시작 실패: %w (xfreerdp가 설치되어 있나요?)", err)
	}

	log.Printf("FreeRDP 시작: session=%s pid=%d", sessionID, cmd.Process.Pid)

	var mu sync.Mutex
	done := make(chan struct{})

	// stderr 읽기 → OS 감지 정보 파싱
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Server OS") || strings.Contains(line, "Windows") {
				log.Printf("RDP OS 감지 (session=%s): %s", sessionID, line)
			}
		}
	}()

	// 클라이언트 입력 → FreeRDP stdin (scan code based)
	go func() {
		for {
			select {
			case <-done:
				return
			case data, ok := <-inputCh:
				if !ok { return }
				mu.Lock()
				stdin.Write(data)
				mu.Unlock()
			}
		}
	}()

	// 프로세스 종료 대기
	go func() {
		cmd.Wait()
		close(done)
		log.Printf("FreeRDP 종료: session=%s", sessionID)
	}()

	// 연결 성공 후 일정 시간 대기
	time.Sleep(2 * time.Second)

	return nil
}

// EncodeRDPKey — RDP 키 이벤트를 FreeRDP 입력 형식으로 인코딩
// Scan code 기반: 하드웨어 키 입력을 게스트 OS에 직접 전달
func EncodeRDPKey(down bool, scanCode uint16) []byte {
	// FreeRDP stdin 형식 (단순화)
	flag := "dn"
	if !down { flag = "up" }
	return []byte(fmt.Sprintf("%s 0x%04x\n", flag, scanCode))
}

var (
	_ = io.EOF
	_ = exec.Command
	_ = bufio.Scanner
)

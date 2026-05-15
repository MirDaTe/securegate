package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"github.com/mirdate/securegate/internal/relay"
)

// Handler — SSH ↔ WebSocket 릴레이
type Handler struct {
	hub      *relay.Hub
	hostInfo *HostInfo
}

// HostInfo — 복호화된 접속 정보
type HostInfo struct {
	Address  string
	Port     int
	Username string
	Password string
	KeyBytes []byte
}

func NewHandler(hub *relay.Hub, hostInfo *HostInfo) *Handler {
	return &Handler{hub: hub, hostInfo: hostInfo}
}

// Start — SSH 연결 + 릴레이 고루틴 시작
// inputCh는 hub.ForwardToRelay에서 데이터가 들어오는 채널
func (h *Handler) Start(ctx context.Context, sessionID string, inputCh <-chan []byte) error {
	sshConfig := &gossh.ClientConfig{
		User:            h.hostInfo.Username,
		Auth:            []gossh.AuthMethod{},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if h.hostInfo.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, gossh.Password(h.hostInfo.Password))
	}
	if len(h.hostInfo.KeyBytes) > 0 {
		signer, err := gossh.ParsePrivateKey(h.hostInfo.KeyBytes)
		if err != nil {
			return fmt.Errorf("SSH 키 파싱 실패: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, gossh.PublicKeys(signer))
	}

	addr := fmt.Sprintf("%s:%d", h.hostInfo.Address, h.hostInfo.Port)
	client, err := gossh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH 연결 실패 (%s): %w", addr, err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("SSH 세션 생성 실패: %w", err)
	}

	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 80, 24, modes); err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("PTY 할당 실패: %w", err)
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return err
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return err
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("shell 시작 실패: %w", err)
	}

	done := make(chan struct{})
	var mu sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(2)

	// stdin: 클라이언트 입력 → SSH
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case data, ok := <-inputCh:
				if !ok { return }
				// 원시 키 이벤트를 그대로 SSH stdin으로 전달
				mu.Lock()
				stdinPipe.Write(data)
				mu.Unlock()
			}
		}
	}()

	// stdout: SSH 출력 → 클라이언트
	go func() {
		defer wg.Done()
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				frame := relay.EncodeOutput(buf[:n])
				h.hub.SendToClient(sessionID, frame)
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("SSH stdout 읽기 실패: %v", err)
				}
				return
			}
		}
	}()

	go func() {
		<-done
		mu.Lock()
		session.Close()
		client.Close()
		mu.Unlock()
		log.Printf("SSH 세션 종료: %s", sessionID)
	}()

	return nil
}

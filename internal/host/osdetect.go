package host

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// OSDetector — 게스트 OS 감지기
type OSDetector struct{}

// NewOSDetector — 생성자
func NewOSDetector() *OSDetector {
	return &OSDetector{}
}

// DetectSSH — SSH 접속 직후 OS 감지
// SSH 연결을 통해 원격 명령을 실행하여 OS 정보 수집
func (d *OSDetector) DetectSSH(ctx context.Context, address string, sshConfig *ssh.ClientConfig) (*OSInfo, error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", address, 22), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 연결 실패: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("SSH 세션 생성 실패: %w", err)
	}
	defer session.Close()

	// OS 감지 명령
	output, err := session.CombinedOutput("uname -a 2>/dev/null; cat /etc/os-release 2>/dev/null; ver 2>/dev/null; sw_vers 2>/dev/null")
	if err != nil {
		// 오류가 있어도 출력은 분석 시도
	}
	return d.parseOSOutput(string(output)), nil
}

// DetectRDP — RDP 협상에서 OS 정보 파싱 (FreeRDP 로그 기반)
func (d *OSDetector) DetectRDP(negotiationLog string) *OSInfo {
	// FreeRDP 로그에서 Windows 버전 정보 추출
	// 예: "Server OS build: Windows 10 Enterprise (19045)"
	for _, line := range strings.Split(negotiationLog, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Server OS build") || strings.Contains(line, "Windows") {
			parts := strings.SplitN(line, ":", 2)
			version := strings.TrimSpace(parts[len(parts)-1])
			return &OSInfo{OS: "windows", Version: version}
		}
	}
	return &OSInfo{OS: "windows", Version: "Unknown"} // 기본값
}

func (d *OSDetector) parseOSOutput(output string) *OSInfo {
	info := &OSInfo{}

	lower := strings.ToLower(output)

	switch {
	case strings.Contains(lower, "microsoft") || strings.Contains(lower, "windows"):
		info.OS = "windows"
		// 버전 추출 시도
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Microsoft Windows") {
				info.Version = strings.TrimSpace(line)
				break
			}
		}
		if info.Version == "" {
			info.Version = "Unknown Windows Version"
		}

	case strings.Contains(lower, "darwin") || strings.Contains(lower, "mac"):
		info.OS = "macos"
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "ProductVersion") || strings.Contains(line, "Mac OS X") {
				info.Version = strings.TrimSpace(line)
				break
			}
		}
		if info.Version == "" {
			info.Version = "Unknown macOS Version"
		}

	case strings.Contains(lower, "linux"):
		info.OS = "linux"
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.Version = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				break
			}
		}
		if info.Version == "" {
			info.Version = "Unknown Linux Distribution"
		}
	}

	return info
}

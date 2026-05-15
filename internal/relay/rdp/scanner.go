package rdp

// 한/영 키 특수 처리 — 브라우저 KeyboardEvent.code → RDP Scan Code 매핑
// PRD 5.4: Guacamole의 한/영 키 문제를 scan code 기반 전송으로 해결

// Lang1 (한/영), Lang2 (한자) 특수 키
const (
	ScanCodeHangul = 0x72 // 한/영 (Lang1)
	ScanCodeHanja  = 0x71 // 한자 (Lang2)
)

// KeyCodeToScanCode — 브라우저 KeyboardEvent.code → RDP/Win32 Scan Code
// 한국어 키보드 특수 키를 포함한 완전한 매핑
func KeyCodeToScanCode(code string) uint16 {
	switch code {
	// 한국어 특수 키 (최우선 처리)
	case "Lang1", "HangulMode", "HanjaMode": // 한/영
		return ScanCodeHangul
	case "Lang2", "Hanja": // 한자
		return ScanCodeHanja
	case "HangulMode", "HanjaMode":
		return ScanCodeHangul

	// ESC / F1-F12
	case "Escape": return 0x01
	case "F1": return 0x3B
	case "F2": return 0x3C
	case "F3": return 0x3D
	case "F4": return 0x3E
	case "F5": return 0x3F
	case "F6": return 0x40
	case "F7": return 0x41
	case "F8": return 0x42
	case "F9": return 0x43
	case "F10": return 0x44
	case "F11": return 0x57
	case "F12": return 0x58

	// 숫자 행
	case "Backquote": return 0x29
	case "Digit1": return 0x02
	case "Digit2": return 0x03
	case "Digit3": return 0x04
	case "Digit4": return 0x05
	case "Digit5": return 0x06
	case "Digit6": return 0x07
	case "Digit7": return 0x08
	case "Digit8": return 0x09
	case "Digit9": return 0x0A
	case "Digit0": return 0x0B
	case "Minus": return 0x0C
	case "Equal": return 0x0D
	case "Backspace": return 0x0E

	// 탭 / 문자 행
	case "Tab": return 0x0F
	case "KeyQ": return 0x10
	case "KeyW": return 0x11
	case "KeyE": return 0x12
	case "KeyR": return 0x13
	case "KeyT": return 0x14
	case "KeyY": return 0x15
	case "KeyU": return 0x16
	case "KeyI": return 0x17
	case "KeyO": return 0x18
	case "KeyP": return 0x19
	case "BracketLeft": return 0x1A
	case "BracketRight": return 0x1B
	case "Backslash": return 0x2B
	case "CapsLock": return 0x3A

	// 홈 행
	case "KeyA": return 0x1E
	case "KeyS": return 0x1F
	case "KeyD": return 0x20
	case "KeyF": return 0x21
	case "KeyG": return 0x22
	case "KeyH": return 0x23
	case "KeyJ": return 0x24
	case "KeyK": return 0x25
	case "KeyL": return 0x26
	case "Semicolon": return 0x27
	case "Quote": return 0x28
	case "Enter": return 0x1C

	// 아래 행
	case "ShiftLeft": return 0x2A
	case "KeyZ": return 0x2C
	case "KeyX": return 0x2D
	case "KeyC": return 0x2E
	case "KeyV": return 0x2F
	case "KeyB": return 0x30
	case "KeyN": return 0x31
	case "KeyM": return 0x32
	case "Comma": return 0x33
	case "Period": return 0x34
	case "Slash": return 0x35
	case "ShiftRight": return 0x36

	// 컨트롤 키
	case "ControlLeft": return 0x1D
	case "ControlRight": return 0x1D // 확장: 0xE01D
	case "AltLeft": return 0x38
	case "AltRight": return 0x38 // 확장: 0xE038
	case "MetaLeft", "OSLeft": return 0x5B // Left Windows
	case "MetaRight", "OSRight": return 0x5C // Right Windows
	case "Space": return 0x39
	case "ContextMenu": return 0x5D

	// 네비게이션
	case "Insert": return 0x52 // E0 52
	case "Delete": return 0x53 // E0 53
	case "Home": return 0x47  // E0 47
	case "End": return 0x4F   // E0 4F
	case "PageUp": return 0x49 // E0 49
	case "PageDown": return 0x51 // E0 51

	// 화살표 키
	case "ArrowUp": return 0x48   // E0 48
	case "ArrowDown": return 0x50 // E0 50
	case "ArrowLeft": return 0x4B // E0 4B
	case "ArrowRight": return 0x4D // E0 4D

	// NumPad
	case "NumLock": return 0x45
	case "NumpadDivide": return 0x35 // E0 35
	case "NumpadMultiply": return 0x37
	case "NumpadSubtract": return 0x4A
	case "NumpadAdd": return 0x4E
	case "NumpadEnter": return 0x1C // E0 1C
	case "NumpadDecimal": return 0x53
	case "Numpad0": return 0x52
	case "Numpad1": return 0x4F
	case "Numpad2": return 0x50
	case "Numpad3": return 0x51
	case "Numpad4": return 0x4B
	case "Numpad5": return 0x4C
	case "Numpad6": return 0x4D
	case "Numpad7": return 0x47
	case "Numpad8": return 0x48
	case "Numpad9": return 0x49

	// 미디어 / 기타
	case "PrintScreen": return 0x37 // E0 2A E0 37
	case "ScrollLock": return 0x46
	case "Pause": return 0x45 // E1 1D 45 E1 9D C5

	default:
		return 0
	}
}

// IsKoreanSpecialKey — 한/영 특수 키인지 확인
func IsKoreanSpecialKey(code string) bool {
	switch code {
	case "Lang1", "Lang2", "HangulMode", "HanjaMode", "Hanja":
		return true
	}
	return false
}

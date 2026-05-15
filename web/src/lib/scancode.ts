/**
 * 한/영 키 → Scan Code 매핑 (브라우저 → RDP)
 * PRD 5.4: KeyboardEvent.code 기반 scan code 전송
 * 
 * Guacamole의 한/영 키 문제 근본 해결:
 * - 브라우저 IME가 아닌 게스트 OS IME를 사용하도록 scan code 직접 전송
 * - Lang1(한/영), Lang2(한자)를 명시적으로 처리
 */

// 특수 키 코드
export const SCAN_HANGUL = 0x72;  // 한/영 (Lang1)
export const SCAN_HANJA  = 0x71;  // 한자 (Lang2)

/**
 * KeyboardEvent.code → RDP Scan Code 매핑
 * 한국어 키보드 특수 키 포함
 */
export function codeToScanCode(code: string): number {
  // ── 한국어 특수 키 (한/영, 한자) ──
  if (code === 'Lang1' || code === 'HangulMode') return SCAN_HANGUL;
  if (code === 'Lang2' || code === 'Hanja' || code === 'HanjaMode') return SCAN_HANJA;

  // ── ESC / F1-F12 ──
  if (code === 'Escape') return 0x01;
  if (code === 'F1')  return 0x3B;
  if (code === 'F2')  return 0x3C;
  if (code === 'F3')  return 0x3D;
  if (code === 'F4')  return 0x3E;
  if (code === 'F5')  return 0x3F;
  if (code === 'F6')  return 0x40;
  if (code === 'F7')  return 0x41;
  if (code === 'F8')  return 0x42;
  if (code === 'F9')  return 0x43;
  if (code === 'F10') return 0x44;
  if (code === 'F11') return 0x57;
  if (code === 'F12') return 0x58;

  // ── 숫자 행 ──
  if (code === 'Backquote') return 0x29;
  if (code === 'Digit1') return 0x02;
  if (code === 'Digit2') return 0x03;
  if (code === 'Digit3') return 0x04;
  if (code === 'Digit4') return 0x05;
  if (code === 'Digit5') return 0x06;
  if (code === 'Digit6') return 0x07;
  if (code === 'Digit7') return 0x08;
  if (code === 'Digit8') return 0x09;
  if (code === 'Digit9') return 0x0a;
  if (code === 'Digit0') return 0x0b;
  if (code === 'Minus')  return 0x0c;
  if (code === 'Equal')  return 0x0d;
  if (code === 'Backspace') return 0x0e;

  // ── 문자 행 ──
  if (code === 'Tab') return 0x0f;
  if (code === 'KeyQ') return 0x10;
  if (code === 'KeyW') return 0x11;
  if (code === 'KeyE') return 0x12;
  if (code === 'KeyR') return 0x13;
  if (code === 'KeyT') return 0x14;
  if (code === 'KeyY') return 0x15;
  if (code === 'KeyU') return 0x16;
  if (code === 'KeyI') return 0x17;
  if (code === 'KeyO') return 0x18;
  if (code === 'KeyP') return 0x19;
  if (code === 'BracketLeft')  return 0x1a;
  if (code === 'BracketRight') return 0x1b;
  if (code === 'Backslash') return 0x2b;
  if (code === 'CapsLock') return 0x3a;

  // ── 홈 행 ──
  if (code === 'KeyA') return 0x1e;
  if (code === 'KeyS') return 0x1f;
  if (code === 'KeyD') return 0x20;
  if (code === 'KeyF') return 0x21;
  if (code === 'KeyG') return 0x22;
  if (code === 'KeyH') return 0x23;
  if (code === 'KeyJ') return 0x24;
  if (code === 'KeyK') return 0x25;
  if (code === 'KeyL') return 0x26;
  if (code === 'Semicolon') return 0x27;
  if (code === 'Quote') return 0x28;
  if (code === 'Enter') return 0x1c;

  // ── 아래 행 ──
  if (code === 'ShiftLeft')  return 0x2a;
  if (code === 'ShiftRight') return 0x36;
  if (code === 'KeyZ') return 0x2c;
  if (code === 'KeyX') return 0x2d;
  if (code === 'KeyC') return 0x2e;
  if (code === 'KeyV') return 0x2f;
  if (code === 'KeyB') return 0x30;
  if (code === 'KeyN') return 0x31;
  if (code === 'KeyM') return 0x32;
  if (code === 'Comma')  return 0x33;
  if (code === 'Period') return 0x34;
  if (code === 'Slash')  return 0x35;

  // ── 컨트롤 키 ──
  if (code === 'ControlLeft' || code === 'ControlRight') return 0x1d;
  if (code === 'AltLeft' || code === 'AltRight') return 0x38;
  if (code === 'MetaLeft' || code === 'OSLeft')   return 0x5b;
  if (code === 'MetaRight' || code === 'OSRight') return 0x5c;
  if (code === 'Space') return 0x39;
  if (code === 'ContextMenu') return 0x5d;

  // ── 네비게이션 ──
  if (code === 'Insert')    return 0x52;
  if (code === 'Delete')    return 0x53;
  if (code === 'Home')      return 0x47;
  if (code === 'End')       return 0x4f;
  if (code === 'PageUp')    return 0x49;
  if (code === 'PageDown')  return 0x51;

  // ── 화살표 ──
  if (code === 'ArrowUp')    return 0x48;
  if (code === 'ArrowDown')  return 0x50;
  if (code === 'ArrowLeft')  return 0x4b;
  if (code === 'ArrowRight') return 0x4d;

  return 0;
}

/** 한/영 특수 키 확인 */
export function isKoreanKey(code: string): boolean {
  return code === 'Lang1' || code === 'Lang2' || 
         code === 'HangulMode' || code === 'Hanja' || code === 'HanjaMode';
}

/** Scan code → 한/영 상태 문자열 */
export function scanCodeToIMEState(scanCode: number): 'HANGUL' | 'ENGLISH' | null {
  if (scanCode === SCAN_HANGUL) return 'HANGUL';
  if (scanCode === SCAN_HANJA)  return 'HANGUL';
  return null;
}

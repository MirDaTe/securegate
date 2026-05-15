import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import { codeToScanCode, isKoreanKey, SCAN_HANGUL, SCAN_HANJA } from '@/lib/scancode';

const FRAME_FRAME  = 0x03;
const FRAME_RESIZE = 0x06;
const FRAME_KEY    = 0x01;
const FRAME_MOUSE  = 0x02;
const FRAME_PING   = 0x07;
const FRAME_IME    = 0x09;

function encodeFrame(type: number, payload: Uint8Array): ArrayBuffer {
  const buf = new ArrayBuffer(5 + payload.length);
  const v = new DataView(buf);
  v.setUint8(0, type);
  v.setUint32(1, payload.length, false);
  new Uint8Array(buf, 5).set(payload);
  return buf;
}

function encodeKey(down: boolean, scanCode: number): ArrayBuffer {
  const p = new Uint8Array(3);
  p[0] = down ? 1 : 0;
  p[1] = (scanCode >> 8) & 0xff;
  p[2] = scanCode & 0xff;
  return encodeFrame(FRAME_KEY, p);
}

/** RDP (Canvas 기반) 원격 데스크톱 세션 뷰 */
export default function RdpSessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');
  const [imeState, setImeState] = useState<'HANGUL' | 'ENGLISH'>('ENGLISH');
  const [imeMode, setImeMode] = useState<'guest' | 'host'>('guest'); // 게스트 IME 기본
  const [wsToken, setWsToken] = useState('');

  const connectRDP = useCallback(async () => {
    try {
      const resp = await api.post('/sessions', { host_id: sessionId, width: 1024, height: 768 });
      const { ws_token, ws_endpoint } = resp.data;
      setWsToken(ws_token);

      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      const ws = new WebSocket(`${proto}//${location.host}${ws_endpoint}?token=${ws_token}`);
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;

      ws.onopen = () => setStatus('connected');
      ws.onmessage = (e) => {
        if (typeof e.data === 'string') return;
        const data = new Uint8Array(e.data);
        if (data.length < 5) return;
        const type = data[0];
        const len = new DataView(data.buffer).getUint32(1, false);
        const payload = data.slice(5, 5 + len);

        switch (type) {
          case FRAME_FRAME:
            // TODO: H.264/PNG 프레임을 Canvas에 렌더링
            // MVP: 프레임 도착 확인만
            break;
          case FRAME_IME:
            // IME 상태 업데이트
            if (payload.length > 0) {
              setImeState(payload[0] === 1 ? 'HANGUL' : 'ENGLISH');
            }
            break;
        }
      };
      ws.onclose = () => setStatus('disconnected');
      ws.onerror = () => setStatus('disconnected');
    } catch (err) {
      console.error('RDP 연결 실패:', err);
      setStatus('disconnected');
    }
  }, [sessionId]);

  // 키보드 → Scan Code → WS 전송
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (wsRef.current?.readyState !== WebSocket.OPEN) return;

      // IME Bypass: composition 이벤트 무시, raw key만
      if (e.isComposing) return;

      // 한/영 특수 키 → Lang1/Lang2를 scan code로 변환
      if (isKoreanKey(e.code)) {
        e.preventDefault();
        const scanCode = e.code === 'Lang2' || e.code === 'Hanja' || e.code === 'HanjaMode'
          ? SCAN_HANJA : SCAN_HANGUL;
        wsRef.current.send(encodeKey(true, scanCode));
        // 키 뗌 이벤트
        setTimeout(() => {
          wsRef.current?.send(encodeKey(false, scanCode));
        }, 50);
        return;
      }

      const scanCode = codeToScanCode(e.code);
      if (scanCode === 0) return; // 매핑 불가능한 키 무시
      e.preventDefault();
      wsRef.current.send(encodeKey(true, scanCode));
    };

    const handleKeyUp = (e: KeyboardEvent) => {
      if (wsRef.current?.readyState !== WebSocket.OPEN) return;
      if (isKoreanKey(e.code)) return; // 이미 처리됨
      const scanCode = codeToScanCode(e.code);
      if (scanCode === 0) return;
      wsRef.current.send(encodeKey(false, scanCode));
    };

    window.addEventListener('keydown', handleKeyDown, true);
    window.addEventListener('keyup', handleKeyUp, true);

    return () => {
      window.removeEventListener('keydown', handleKeyDown, true);
      window.removeEventListener('keyup', handleKeyUp, true);
    };
  }, []);

  useEffect(() => {
    connectRDP();
    return () => { wsRef.current?.close(); };
  }, [connectRDP]);

  return (
    <div className="min-h-screen bg-gray-950 flex flex-col">
      {/* 툴바 */}
      <div className="bg-gray-900 px-4 py-2 flex items-center justify-between text-white text-sm">
        <div className="flex items-center gap-3">
          <span className={`w-2 h-2 rounded-full ${
            status === 'connected' ? 'bg-green-500' : status === 'connecting' ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'
          }`} />
          <span>{t(`session.${status}`)}</span>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setImeMode(m => m === 'guest' ? 'host' : 'guest')}
            className="px-2 py-1 bg-gray-700 rounded text-xs hover:bg-gray-600"
            title="한/영 모드 전환"
          >
            IME: {imeMode === 'guest' ? '게스트' : '호스트'}
          </button>
          <button onClick={() => { wsRef.current?.close(); navigate('/dashboard'); }}
            className="px-3 py-1 bg-red-600 hover:bg-red-700 rounded text-xs">
            {t('session.terminate')}
          </button>
        </div>
      </div>

      {/* Canvas 렌더링 영역 */}
      <div className="flex-1 flex items-center justify-center bg-gray-950">
        {status !== 'connected' ? (
          <div className="text-center text-gray-500">
            <p className="text-lg mb-2">{t(`session.${status}`)}</p>
            <p className="text-sm text-gray-600">RDP 화면은 Canvas로 렌더링됩니다</p>
          </div>
        ) : (
          <canvas ref={canvasRef} className="max-w-full max-h-full object-contain" />
        )}
      </div>

      {/* 하단 바 — 한/영 인디케이터 */}
      <div className="bg-gray-900 px-4 py-1 flex items-center justify-between text-xs text-gray-400">
        <div className="flex items-center gap-4">
          <span>
            {t('session.imeIndicator')}:{' '}
            <strong className={imeState === 'HANGUL' ? 'text-green-400' : 'text-white'}>
              {imeState === 'HANGUL' ? '가' : 'A'}
            </strong>
          </span>
          <span>모드: {imeMode === 'guest' ? '게스트 IME' : '호스트 IME'}</span>
        </div>
        <span>SecureGate RDP</span>
      </div>
    </div>
  );
}

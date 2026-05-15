import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import api from '@/lib/api';

/**
 * SessionPage — 웹 터미널 (xterm.js + WebSocket)
 * SSH 원격 접속을 브라우저에서 직접 렌더링
 */

// 바이너리 프레임 상수 (서버와 동기화)
const FRAME_OUTPUT = 0x0a;
const FRAME_RESIZE = 0x06;
const FRAME_KEY = 0x01;
const FRAME_PING = 0x07;

function encodeFrame(frameType: number, payload: Uint8Array): ArrayBuffer {
  const buf = new ArrayBuffer(5 + payload.length);
  const view = new DataView(buf);
  view.setUint8(0, frameType);
  view.setUint32(1, payload.length, false);
  new Uint8Array(buf, 5).set(payload);
  return buf;
}

function encodeKeyFrame(down: boolean, scanCode: number): ArrayBuffer {
  const payload = new Uint8Array(3);
  payload[0] = down ? 1 : 0;
  payload[1] = (scanCode >> 8) & 0xff;
  payload[2] = scanCode & 0xff;
  return encodeFrame(FRAME_KEY, payload);
}

function encodeResizeFrame(width: number, height: number): ArrayBuffer {
  const payload = new Uint8Array(4);
  const view = new DataView(payload.buffer);
  view.setUint16(0, width, false);
  view.setUint16(2, height, false);
  return encodeFrame(FRAME_RESIZE, payload);
}

export default function SessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const terminalRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');
  const [hostName, setHostName] = useState('');

  // 세션 생성 → WS 연결
  const connectSession = useCallback(async () => {
    try {
      // 1. 세션 생성 API 호출
      const resp = await api.post('/sessions', {
        host_id: sessionId,
        width: 80,
        height: 24,
      });
      const { ws_token, ws_endpoint, session } = resp.data;
      setHostName(session.host_id);

      // 2. WebSocket 연결
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}${ws_endpoint}?token=${ws_token}`;
      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus('connected');
        console.log('WS 연결 성공');
      };

      ws.onmessage = (event) => {
        if (typeof event.data === 'string') {
          // JSON 메시지 (connected ack 등)
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'connected') {
              console.log('세션 연결됨:', msg);
            }
          } catch {}
          return;
        }

        // 바이너리 메시지: 프레임 디코딩
        const data = new Uint8Array(event.data);
        if (data.length < 5) return;

        const frameType = data[0];
        const length = new DataView(data.buffer).getUint32(1, false);
        const payload = data.slice(5, 5 + length);

        switch (frameType) {
          case FRAME_OUTPUT:
            // SSH 터미널 출력 → xterm에 쓰기
            if (termRef.current) {
              termRef.current.write(payload);
            }
            break;
          case FRAME_PING:
            // keep-alive 응답
            ws.send(encodeFrame(FRAME_PING, new Uint8Array(0)));
            break;
        }
      };

      ws.onclose = () => {
        setStatus('disconnected');
        console.log('WS 연결 종료');
      };

      ws.onerror = (err) => {
        console.error('WS 오류:', err);
        setStatus('disconnected');
      };
    } catch (err) {
      console.error('세션 생성 실패:', err);
      setStatus('disconnected');
    }
  }, [sessionId]);

  // xterm 초기화
  useEffect(() => {
    if (!terminalRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'D2Coding', monospace",
      theme: {
        background: '#1a1b26',
        foreground: '#c0caf5',
        cursor: '#7aa2f7',
        selectionBackground: '#33467c',
        black: '#15161e',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#bb9af7',
        cyan: '#7dcfff',
        white: '#a9b1d6',
        brightBlack: '#414868',
        brightRed: '#f7768e',
        brightGreen: '#9ece6a',
        brightYellow: '#e0af68',
        brightBlue: '#7aa2f7',
        brightMagenta: '#bb9af7',
        brightCyan: '#7dcfff',
        brightWhite: '#c0caf5',
      },
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    // 키 입력 → WebSocket
    term.onData((data) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        // 텍스트 데이터를 그대로 전송 (SSH stdin)
        const encoder = new TextEncoder();
        wsRef.current.send(encoder.encode(data));
      }
    });

    // 창 크기 변경 → resize 프레임
    const handleResize = () => {
      fitAddon.fit();
      if (wsRef.current?.readyState === WebSocket.OPEN && termRef.current) {
        const frame = encodeResizeFrame(term.cols, term.rows);
        wsRef.current.send(frame);
      }
    };
    window.addEventListener('resize', handleResize);

    // 세션 연결 시작
    connectSession();

    return () => {
      window.removeEventListener('resize', handleResize);
      term.dispose();
      wsRef.current?.close();
    };
  }, [connectSession]);

  const handleTerminate = () => {
    wsRef.current?.close();
    navigate('/dashboard');
  };

  return (
    <div className="min-h-screen flex flex-col bg-gray-950">
      {/* 상단 툴바 */}
      <div className="bg-gray-900 px-4 py-2 flex items-center justify-between text-white text-sm">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <span
              className={`w-2 h-2 rounded-full ${
                status === 'connected'
                  ? 'bg-green-500'
                  : status === 'connecting'
                  ? 'bg-yellow-500 animate-pulse'
                  : 'bg-red-500'
              }`}
            />
            <span>
              {status === 'connected'
                ? t('session.connected')
                : status === 'connecting'
                ? t('session.connecting')
                : t('session.disconnected')}
            </span>
          </div>
          {hostName && (
            <span className="text-gray-500">| {hostName.slice(0, 8)}</span>
          )}
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleTerminate}
            className="px-3 py-1 bg-red-600 hover:bg-red-700 rounded text-xs font-medium transition-colors"
          >
            {t('session.terminate')}
          </button>
        </div>
      </div>

      {/* xterm 터미널 */}
      <div className="flex-1 p-1">
        <div
          ref={terminalRef}
          className="w-full h-full rounded"
          style={{ minHeight: '500px' }}
        />
      </div>

      {/* 하단 상태 표시줄 */}
      <div className="bg-gray-900 px-4 py-1 flex items-center justify-between text-xs text-gray-500">
        <div className="flex items-center gap-4">
          <span>
            {termRef.current
              ? `${termRef.current.cols}×${termRef.current.rows}`
              : ''}
          </span>
          <span>xterm.js</span>
        </div>
        <span>SecureGate SSH</span>
      </div>
    </div>
  );
}

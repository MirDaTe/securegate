import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

/**
 * SessionPage — 원격 데스크톱 세션 뷰
 * Step 4~5에서 WebSocket + Canvas/터미널 렌더링 구현
 */
export default function SessionPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting');

  useEffect(() => {
    // TODO Step 4: WebSocket 연결 → 세션 시작
    const timer = setTimeout(() => setStatus('connected'), 1500);
    return () => clearTimeout(timer);
  }, [sessionId]);

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      {/* 상단 툴바 */}
      <div className="bg-gray-800 px-4 py-2 flex items-center justify-between text-white text-sm">
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-2">
            <span className={`w-2 h-2 rounded-full ${
              status === 'connected' ? 'bg-green-500' :
              status === 'connecting' ? 'bg-yellow-500' : 'bg-red-500'
            }`} />
            {t(`session.${status}`)}
          </span>
          <span className="text-gray-400">| 세션: {sessionId}</span>
        </div>
        <div className="flex items-center gap-3">
          <button className="hover:text-gray-300">{t('session.keyboardSettings')}</button>
          <button
            onClick={() => navigate('/dashboard')}
            className="px-3 py-1 bg-red-600 hover:bg-red-700 rounded text-sm"
          >
            {t('session.terminate')}
          </button>
        </div>
      </div>

      {/* 화면 렌더링 영역 */}
      <div className="flex-1 flex items-center justify-center bg-gray-950">
        <div className="text-center text-gray-500">
          <p className="text-lg mb-2">
            {status === 'connecting' ? t('session.connecting') : '세션이 활성 상태입니다'}
          </p>
          <p className="text-sm">
            Canvas/터미널 렌더링은 Step 4~5에서 구현됩니다
          </p>
        </div>
      </div>

      {/* 하단 상태 표시줄 */}
      <div className="bg-gray-800 px-4 py-1 flex items-center justify-between text-xs text-gray-400">
        <div className="flex items-center gap-4">
          <span>{t('session.imeIndicator')}: <strong className="text-white">A</strong></span>
          <span>{t('session.clipboard')}: 허용</span>
        </div>
        <span>{t('session.connected')}</span>
      </div>
    </div>
  );
}

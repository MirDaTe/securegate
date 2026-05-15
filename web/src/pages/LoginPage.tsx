import { useState, FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { useUIStore } from '@/store/uiStore';

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const { language, setLanguage, theme, toggleTheme } = useUIStore();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // MFA 단계
  const [mfaSessionToken, setMfaSessionToken] = useState('');
  const [mfaCode, setMfaCode] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const resp = await api.post('/auth/login', { username, password });
      const data = resp.data;

      if (data.require_mfa) {
        // MFA 필요 → 임시 토큰 저장, MFA 화면으로
        setMfaSessionToken(data.access_token);
        return;
      }

      if (data.require_password_change) {
        // 비밀번호 강제 변경 필요 → 토큰 저장 후 변경 페이지로
        login(data.user, data.access_token, data.refresh_token);
        navigate('/change-password');
        return;
      }

      // 정상 로그인
      login(data.user, data.access_token, data.refresh_token);
      if (data.user.role === 'super_admin' || data.user.role === 'admin') {
        navigate('/admin');
      } else {
        navigate('/dashboard');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || t('common.error'));
    } finally {
      setLoading(false);
    }
  };

  const handleMFAVerify = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const resp = await api.post('/auth/mfa/verify', {
        session_token: mfaSessionToken,
        code: mfaCode,
      });
      const data = resp.data;

      if (data.require_password_change) {
        login(data.user, data.access_token, data.refresh_token);
        navigate('/change-password');
        return;
      }

      login(data.user, data.access_token, data.refresh_token);
      if (data.user.role === 'super_admin' || data.user.role === 'admin') {
        navigate('/admin');
      } else {
        navigate('/dashboard');
      }
    } catch (err: any) {
      setError(err.response?.data?.error || '잘못된 인증 코드입니다');
    } finally {
      setLoading(false);
    }
  };

  // MFA 화면
  if (mfaSessionToken) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              SecureGate
            </h1>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              {t('auth.mfaTitle')}
            </p>
          </div>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-8">
            <form onSubmit={handleMFAVerify} className="space-y-5">
              {error && (
                <div className="bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 px-4 py-3 rounded text-sm">
                  {error}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('auth.mfaCode')}
                </label>
                <input
                  type="text"
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent text-center text-2xl tracking-widest"
                  maxLength={6}
                  autoFocus
                  required
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full py-2 px-4 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-md disabled:opacity-50 transition-colors"
              >
                {loading ? t('common.loading') : t('auth.mfaVerify')}
              </button>
            </form>
            <div className="mt-4 text-center">
              <button
                onClick={() => { setMfaSessionToken(''); setError(''); }}
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                돌아가기
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            SecureGate
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            {t('auth.loginTitle')}
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-8">
          <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
              <div className="bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 px-4 py-3 rounded text-sm">
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('auth.username')}
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                required
                autoFocus
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('auth.password')}
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                required
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 px-4 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-md disabled:opacity-50 transition-colors"
            >
              {loading ? t('common.loading') : t('auth.loginButton')}
            </button>
          </form>

          <div className="mt-6 flex justify-between items-center text-sm">
            <button
              onClick={() => setLanguage(language === 'ko' ? 'en' : 'ko')}
              className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            >
              {language === 'ko' ? 'English' : '한국어'}
            </button>
            <button
              onClick={toggleTheme}
              className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            >
              {theme === 'dark' ? '☀️' : '🌙'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

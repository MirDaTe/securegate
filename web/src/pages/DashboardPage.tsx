import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/store/authStore';
import { useNavigate } from 'react-router-dom';

export default function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* 헤더 */}
      <header className="bg-white dark:bg-gray-800 shadow-sm px-6 py-4 flex justify-between items-center">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-white">
          {t('dashboard.title')}
        </h1>
        <div className="flex items-center gap-4">
          <span className="text-sm text-gray-600 dark:text-gray-400">
            {user?.username}
          </span>
          <button
            onClick={handleLogout}
            className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            {t('common.logout')}
          </button>
        </div>
      </header>

      {/* 메인 */}
      <main className="max-w-6xl mx-auto px-6 py-8">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-8 text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            {t('dashboard.noHosts')}
          </p>
          <p className="text-sm text-gray-400 dark:text-gray-500">
            관리자에게 호스트 등록을 요청하세요. (Step 3에서 구현)
          </p>
        </div>
      </main>
    </div>
  );
}

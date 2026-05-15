import { Outlet } from 'react-router-dom';

/** 관리자 콘솔 레이아웃 — Step 3에서 본격 구현 */
export default function AdminLayout() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex">
      {/* 사이드바 (Step 3에서 구현) */}
      <aside className="w-60 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
        <div className="p-4 font-semibold text-lg text-gray-900 dark:text-white">
          관리자 콘솔
        </div>
        <nav className="px-3 space-y-1">
          {['사용자 관리', '호스트 관리', '정책 관리', '세션 모니터링', '감사 로그'].map((item) => (
            <div key={item} className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer">
              {item}
            </div>
          ))}
        </nav>
      </aside>

      {/* 메인 콘텐츠 */}
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  );
}

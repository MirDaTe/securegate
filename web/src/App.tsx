import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useUIStore } from '@/store/uiStore';
import LoginPage from '@/pages/LoginPage';
import DashboardPage from '@/pages/DashboardPage';
import SessionPage from '@/pages/SessionPage';
import AdminLayout from '@/components/layout/AdminLayout';

// 관리자 콘솔 페이지 (Lazy load — Step 8에서 구현)
const HostManagement = () => <div>호스트 관리 (Step 3에서 구현)</div>;
const UserManagement = () => <div>사용자 관리 (Step 2에서 구현)</div>;

function App() {
  const { theme } = useUIStore();

  return (
    <div className={theme === 'dark' ? 'dark' : ''}>
      <Routes>
        {/* 공개 라우트 */}
        <Route path="/login" element={<LoginPage />} />

        {/* 인증 필요 라우트 */}
        <Route
          path="/dashboard"
          element={<ProtectedRoute><DashboardPage /></ProtectedRoute>}
        />
        <Route
          path="/session/:sessionId"
          element={<ProtectedRoute><SessionPage /></ProtectedRoute>}
        />

        {/* 관리자 콘솔 */}
        <Route
          path="/admin"
          element={<AdminRoute><AdminLayout /></AdminRoute>}
        >
          <Route path="users" element={<UserManagement />} />
          <Route path="hosts" element={<HostManagement />} />
        </Route>

        {/* 기본 리다이렉트 */}
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </div>
  );
}

/** 인증된 사용자만 접근 가능한 라우트 가드 */
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuthStore();
  if (!user) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

/** 관리자 전용 라우트 가드 */
function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuthStore();
  if (!user) return <Navigate to="/login" replace />;
  if (user.role !== 'super_admin' && user.role !== 'admin') {
    return <Navigate to="/dashboard" replace />;
  }
  return <>{children}</>;
}

export default App;

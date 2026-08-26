import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { resolveAdminBasename } from './path';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Users from './pages/Users';
import Admins from './pages/Admins';
import Reviews from './pages/Reviews';
import Plans from './pages/Plans';
import Payments from './pages/Payments';
import AuditLogs from './pages/AuditLogs';
import Settings from './pages/Settings';

export default function App() {
  // 动态取当前路径第一段作为 basename, 支持轮换 admin 路径后无需重新构建。
  // dev 模式挂在根路径, /login 等 SPA 路由段不会被误当托管前缀(见 path.ts)。
  const basename = resolveAdminBasename();
  return (
    <BrowserRouter basename={basename}>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<MainLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="users" element={<Users />} />
          <Route path="admins" element={<Admins />} />
          <Route path="reviews" element={<Reviews />} />
          <Route path="plans" element={<Plans />} />
          <Route path="payments" element={<Payments />} />
          <Route path="audit-logs" element={<AuditLogs />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

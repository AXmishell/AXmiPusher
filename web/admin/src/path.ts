// 管理后台路径工具: 区分"托管前缀"与"SPA 内部路由"。
//
// 生产模式(单端口 8080 / cmd/web 双端口)下管理后台始终托管在 /{admin_path}/ 前缀下,
// basename 取路径第一段即为 admin_path; 但 dev 模式(5174)挂在根路径, 直接访问
// /login、/users 等 SPA 路由时第一段是路由名, 若被误当前缀会导致路由错乱(已踩坑)。

// 管理后台 SPA 内部路由段(不可能作为托管前缀)。
const SPA_ROUTE_SEGMENTS = new Set([
  'login',
  'users',
  'admins',
  'plans',
  'payments',
  'audit-logs',
  'settings',
]);

/** 判断路径第一段是否为管理后台托管前缀(排除已知 SPA 路由段)。 */
export function isAdminPathSegment(seg: string | undefined): boolean {
  return !!seg && !SPA_ROUTE_SEGMENTS.has(seg);
}

/** 计算 BrowserRouter basename: 有托管前缀返回 /{前缀}, 否则返回 /(dev 根路径托管)。 */
export function resolveAdminBasename(): string {
  const seg = window.location.pathname.split('/').filter(Boolean)[0];
  return isAdminPathSegment(seg) ? `/${seg}` : '/';
}

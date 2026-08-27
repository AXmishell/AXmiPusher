/**
 * 全局主题常量(主色与项目 web 前端一致)。
 */

/** 颜色常量。 */
export const colors = {
  /** 主色(深蓝, 与 web 前端 AntD 主题一致)。 */
  primary: '#1e40af',
  /** 危险/退出登录按钮色。 */
  danger: '#dc2626',
  /** 正文文字色。 */
  text: '#1f2937',
  /** 次要文字(已读消息等)。 */
  muted: '#6b7280',
  /** 页面背景。 */
  background: '#f3f4f6',
  /** 卡片背景。 */
  card: '#ffffff',
  /** 边框。 */
  border: '#e5e7eb',
  /** 未读蓝点。 */
  unreadDot: '#1e40af',
} as const;

/** 通用间距。 */
export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
} as const;

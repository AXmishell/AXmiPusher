/**
 * 后端 API 数据类型定义。
 * 与 Go 端 internal/models 及 internal/api/handler 返回结构一一对应。
 */

/** 统一响应封装 {code, message, data}, code=0 表示成功。 */
export interface ApiResp<T> {
  code: number;
  message: string;
  data: T;
}

/** 登录用户信息(users 表字段)。 */
export interface User {
  id: number;
  email: string;
  nickname: string;
  role: string;
  status: string;
  totp_enabled: boolean;
  created_at: string;
}

/** 站内信(InappMessage)。 */
export interface InboxMessage {
  id: number;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

/** 登录成功结果。 */
export interface LoginResult {
  token: string;
  user: User;
}

/** 第一步登录返回: 账户已开启两步验证(TOTP), 需带 totp_token 走第二步。 */
export interface TotpPending {
  need_totp: true;
  totp_token: string;
}

/** 登录第一步返回的联合类型: 直接成功 或 需走验证码。 */
export type LoginData = LoginResult | TotpPending;

/** 收件箱列表响应 data 部分。 */
export interface InboxListData {
  data: InboxMessage[];
  total: number;
  success: boolean;
}

/** 未读数响应 data 部分。 */
export interface UnreadCountData {
  unread: number;
}

/** 当前用户信息响应 data 部分。 */
export interface MeData {
  user: User;
  is_admin: boolean;
}

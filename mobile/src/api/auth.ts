/**
 * 认证相关 API。
 * 契约(来自后端 Go 源码):
 * - POST /api/v1/auth/login {email,password}
 *   → 未开 TOTP: data={token,user}; 已开 TOTP: data={need_totp:true,totp_token}(凭证 5 分钟有效)
 * - POST /api/v1/auth/login/totp {totp_token,code} → data={token,user}
 * - GET  /api/v1/auth/me → data={user,is_admin}
 */
import { request } from './client';
import type { LoginData, LoginResult, MeData, User } from './types';

/**
 * 登录第一步。
 * @returns LoginResult(登录成功) 或 TotpPending(需走两步验证)。
 */
export async function login(
  serverUrl: string,
  email: string,
  password: string,
): Promise<LoginData> {
  return request<LoginData>(serverUrl, '/auth/login', {
    method: 'POST',
    body: { email, password },
  });
}

/** 登录第二步: 提交 TOTP 验证码, 换取正式 token。 */
export async function loginTotp(
  serverUrl: string,
  totpToken: string,
  code: string,
): Promise<LoginResult> {
  return request<LoginResult>(serverUrl, '/auth/login/totp', {
    method: 'POST',
    body: { totp_token: totpToken, code },
  });
}

/** 获取当前登录用户信息(启动时用于校验 token 有效性)。 */
export async function me(serverUrl: string, token: string): Promise<User> {
  const data = await request<MeData>(serverUrl, '/auth/me', { auth: token });
  return data.user;
}

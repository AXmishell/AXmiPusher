/**
 * fetch 请求封装: 统一拼接 URL、携带鉴权头、解包 {code, message, data}。
 * 业务码: 0 成功 / 40000 参数 / 40100 未认证 / 40300 无权限 / 40400 不存在 /
 *         40900 冲突 / 42900 限流 / 50000 服务端。
 */
import type { ApiResp } from './types';

/** 认证失效专用错误: 携带标志位, 供全局自动登出逻辑识别。 */
export class AuthExpiredError extends Error {
  /** 是否为认证过期错误。 */
  isAuthExpired = true;

  constructor(message = '登录已过期, 请重新登录') {
    super(message);
    this.name = 'AuthExpiredError';
  }
}

/**
 * 服务器地址规范化:
 * - 无协议时补 http://(如 192.168.1.5:8080 → http://192.168.1.5:8080);
 * - 去除尾部斜杠(如 https://x.com/ → https://x.com)。
 */
export function normalizeServerUrl(raw: string): string {
  let url = (raw ?? '').trim();
  if (!url) {
    return url;
  }
  // 不含协议前缀时默认补 http://
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(url)) {
    url = 'http://' + url;
  }
  // 去除尾部斜杠
  url = url.replace(/\/+$/, '');
  return url;
}

/** 请求选项。 */
interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  /** 用户 JWT, 以 Bearer 形式放入 Authorization 头。 */
  auth?: string | null;
}

/**
 * 通用请求: 拼 {serverUrl}/api/v1{path}。
 * - HTTP 401 或业务码 40100 → 抛 AuthExpiredError;
 * - 业务码非 0 → 抛带后端 message 的普通 Error;
 * - 网络层失败 → 抛中文"无法连接服务器"错误。
 */
export async function request<T>(
  serverUrl: string,
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = 'GET', body, auth } = options;
  const url = `${normalizeServerUrl(serverUrl)}/api/v1${path}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (auth) {
    headers.Authorization = `Bearer ${auth}`;
  }

  let resp: Response;
  try {
    resp = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    // fetch 网络层异常(域名无法解析 / 连接被拒 / 超时等)
    throw new Error('无法连接服务器, 请检查服务器地址与网络');
  }

  let json: ApiResp<T>;
  try {
    json = (await resp.json()) as ApiResp<T>;
  } catch {
    throw new Error('服务器响应异常, 请稍后重试');
  }

  // 认证过期: HTTP 401 或业务码 40100
  if (resp.status === 401 || json.code === 40100) {
    throw new AuthExpiredError(json.message || '登录已过期, 请重新登录');
  }
  // 其它业务错误
  if (json.code !== 0) {
    throw new Error(json.message || '请求失败, 请稍后重试');
  }
  return json.data;
}

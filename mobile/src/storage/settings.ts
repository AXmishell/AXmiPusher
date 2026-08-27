/**
 * 会话持久化: 用 react-native-keychain 存取 JSON blob(键名 axmipusher_session)。
 * 存的是完整会话 {serverUrl, token, email, user, pollIntervalMinutes}。
 */
import * as Keychain from 'react-native-keychain';
import type { User } from '../api/types';

/** keychain 服务键(即存储键名)。 */
const SESSION_KEY = 'axmipusher_session';

/** 登录会话。 */
export interface Session {
  serverUrl: string;
  token: string;
  email: string;
  user: User;
  /** 原生后台轮询间隔(分钟), 默认 15。 */
  pollIntervalMinutes?: number;
}

/** 读取会话; 无会话或 JSON 损坏时返回 null。 */
export async function loadSession(): Promise<Session | null> {
  try {
    const creds = await Keychain.getGenericPassword({ service: SESSION_KEY });
    if (!creds) {
      return null;
    }
    const parsed = JSON.parse(creds.password) as Session;
    // 基本完整性校验: 必须含 serverUrl 与 token
    if (!parsed || typeof parsed.serverUrl !== 'string' || typeof parsed.token !== 'string') {
      return null;
    }
    return parsed;
  } catch {
    // 读取失败(未安装/数据损坏)视为无会话
    return null;
  }
}

/** 保存会话(覆盖写)。 */
export async function saveSession(session: Session): Promise<void> {
  await Keychain.setGenericPassword('session', JSON.stringify(session), {
    service: SESSION_KEY,
  });
}

/** 清除会话(退出登录时调用)。 */
export async function clearSession(): Promise<void> {
  try {
    await Keychain.resetGenericPassword({ service: SESSION_KEY });
  } catch {
    // 清除失败忽略, 视为已无会话
  }
}

/** 便捷读取 token; 无会话返回 null。 */
export async function getToken(): Promise<string | null> {
  const session = await loadSession();
  return session ? session.token : null;
}

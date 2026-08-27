/**
 * 全局认证上下文。
 *
 * 职责:
 * - 启动时恢复会话(bootstrapping): loadSession → 有 token 则调 me() 校验 →
 *   成功配置原生轮询器(configure+start)并进入已登录;
 * - 登录/登出逻辑: 登录成功后保存会话 + 启动前台/原生轮询; 登出停止轮询并清会话;
 * - 401 自动登出: 请求遇 AuthExpiredError 时自动回到登录页并暴露"登录已过期"提示。
 *
 * 状态机: bootstrapping(启动中) → signedOut(未登录) / signedIn(已登录)。
 */
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import * as authApi from '../api/auth';
import { AuthExpiredError, normalizeServerUrl } from '../api/client';
import type { LoginData, LoginResult, User } from '../api/types';
import { getBackgroundPoller } from '../native/BackgroundPoller';
import { startPolling, stopPolling } from '../services/poller';
import {
  clearSession,
  loadSession,
  saveSession,
  type Session,
} from '../storage/settings';

/** 认证状态机: 启动中 / 已登出 / 已登录。 */
export type AuthStatus = 'bootstrapping' | 'signedOut' | 'signedIn';

/** 原生后台轮询默认间隔(分钟)。 */
export const DEFAULT_POLL_INTERVAL_MINUTES = 15;

/** 认证上下文值。 */
export interface AuthContextValue {
  status: AuthStatus;
  session: Session | null;
  /** 登录过期提示(非空时登录页展示 Alert)。 */
  expiredNotice: string | null;
  /** 第一步登录: 返回登录结果或 TOTP 待验证凭证(需两步验证时由登录页接管)。 */
  login: (serverUrl: string, email: string, password: string) => Promise<LoginData>;
  /** 第二步 TOTP 登录(成功后自动进入主界面)。 */
  loginTotp: (serverUrl: string, totpToken: string, code: string) => Promise<void>;
  /** 退出登录: 停止轮询 + 清 keychain 会话。 */
  logout: () => Promise<void>;
  /** 请求遇 401 时通知全局: 自动登出并暴露过期提示。 */
  notifyAuthExpired: (message?: string) => void;
  /** 更新会话(设置页保存轮询间隔后调用, 同步重新配置原生)。 */
  updateSession: (
    patch: Partial<Pick<Session, 'pollIntervalMinutes'>>,
  ) => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/** 读取认证上下文。 */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth 必须在 AuthProvider 内使用');
  }
  return ctx;
}

/** 认证上下文属性。 */
interface AuthProviderProps {
  children: React.ReactNode;
}

/**
 * 认证提供者。
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const [status, setStatus] = useState<AuthStatus>('bootstrapping');
  const [session, setSession] = useState<Session | null>(null);
  const [expiredNotice, setExpiredNotice] = useState<string | null>(null);
  // 会话 ref: 供异步回调读取最新值, 避免闭包过期
  const sessionRef = useRef<Session | null>(null);
  // 原生轮询是否已启动(避免重复 start)
  const nativeStartedRef = useRef(false);
  // 启动引导只执行一次
  const bootRef = useRef(false);

  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  /**
   * 登录成功后的统一处理:
   * 保存会话 → 配置并启动原生轮询 → 启动前台轮询 → 置为已登录。
   */
  const applySignIn = useCallback(
    async (result: { token: string; user: User }, serverUrl: string, interval?: number) => {
      const next: Session = {
        serverUrl: normalizeServerUrl(serverUrl),
        token: result.token,
        email: result.user.email,
        user: result.user,
        pollIntervalMinutes: interval ?? DEFAULT_POLL_INTERVAL_MINUTES,
      };
      await saveSession(next);
      // 配置并启动原生后台轮询 Worker
      const poller = getBackgroundPoller();
      try {
        await poller.configure({
          serverUrl: next.serverUrl,
          token: next.token,
          pollIntervalMinutes: next.pollIntervalMinutes ?? DEFAULT_POLL_INTERVAL_MINUTES,
        });
        if (!nativeStartedRef.current) {
          await poller.start();
          nativeStartedRef.current = true;
        }
      } catch {
        // 原生模块缺失时忽略, 前台轮询照常工作
      }
      // 启动前台轮询
      startPolling(next).catch(() => {
        // 忽略启动异常(原生缺失等)
      });
      sessionRef.current = next;
      setSession(next);
      setExpiredNotice(null);
      setStatus('signedIn');
    },
    [],
  );

  /** 第一步登录。 */
  const login = useCallback(
    async (serverUrl: string, email: string, password: string): Promise<LoginData> => {
      const data = await authApi.login(normalizeServerUrl(serverUrl), email, password);
      // 需两步验证: 不登录, 由登录页进入验证码流程
      if ('need_totp' in data && data.need_totp === true) {
        return data;
      }
      const result = data as LoginResult;
      await applySignIn({ token: result.token, user: result.user }, serverUrl);
      return data;
    },
    [applySignIn],
  );

  /** 第二步 TOTP 登录。 */
  const loginTotp = useCallback(
    async (serverUrl: string, totpToken: string, code: string): Promise<void> => {
      const result = await authApi.loginTotp(normalizeServerUrl(serverUrl), totpToken, code);
      await applySignIn({ token: result.token, user: result.user }, serverUrl);
    },
    [applySignIn],
  );

  /** 退出登录。 */
  const logout = useCallback(async () => {
    stopPolling();
    const poller = getBackgroundPoller();
    try {
      await poller.stop();
    } catch {
      // 原生缺失时忽略
    }
    nativeStartedRef.current = false;
    await clearSession();
    sessionRef.current = null;
    setSession(null);
    setExpiredNotice(null);
    setStatus('signedOut');
  }, []);

  /** 请求遇 401: 自动登出 + 暴露过期提示(无需 await)。 */
  const notifyAuthExpired = useCallback((message?: string) => {
    stopPolling();
    const poller = getBackgroundPoller();
    poller
      .stop()
      .then(() => {
        nativeStartedRef.current = false;
      })
      .catch(() => {
        nativeStartedRef.current = false;
      });
    clearSession().catch(() => {
      // 忽略
    });
    sessionRef.current = null;
    setSession(null);
    setExpiredNotice(message ?? '登录已过期, 请重新登录');
    setStatus('signedOut');
  }, []);

  /** 更新会话(设置页保存轮询间隔)。 */
  const updateSession = useCallback(
    async (patch: Partial<Pick<Session, 'pollIntervalMinutes'>>) => {
      const prev = sessionRef.current;
      if (!prev) {
        return;
      }
      const next: Session = { ...prev, ...patch };
      await saveSession(next);
      sessionRef.current = next;
      setSession(next);
      // 重新配置原生轮询(间隔变化)
      const poller = getBackgroundPoller();
      try {
        await poller.configure({
          serverUrl: next.serverUrl,
          token: next.token,
          pollIntervalMinutes: next.pollIntervalMinutes ?? DEFAULT_POLL_INTERVAL_MINUTES,
        });
      } catch {
        // 原生缺失时忽略
      }
    },
    [],
  );

  // 启动引导: 恢复会话并校验 token
  useEffect(() => {
    if (bootRef.current) {
      return;
    }
    bootRef.current = true;
    void (async () => {
      const saved = await loadSession();
      if (!saved) {
        setStatus('signedOut');
        return;
      }
      try {
        // 用 me() 校验 token 是否有效, 并刷新用户信息
        const user = await authApi.me(saved.serverUrl, saved.token);
        await applySignIn({ token: saved.token, user }, saved.serverUrl, saved.pollIntervalMinutes);
      } catch (err) {
        if (err instanceof AuthExpiredError) {
          // token 过期: 清会话, 回登录页并提示
          await clearSession();
          sessionRef.current = null;
          setSession(null);
          setExpiredNotice('登录已过期, 请重新登录');
          setStatus('signedOut');
        } else {
          // 网络等服务异常: 用缓存会话进入主界面, 轮询稍后自动恢复
          await applySignIn(
            { token: saved.token, user: saved.user },
            saved.serverUrl,
            saved.pollIntervalMinutes,
          );
        }
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      session,
      expiredNotice,
      login,
      loginTotp,
      logout,
      notifyAuthExpired,
      updateSession,
    }),
    [status, session, expiredNotice, login, loginTotp, logout, notifyAuthExpired, updateSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

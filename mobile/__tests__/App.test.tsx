/**
 * @format
 *
 * 单元测试:
 * 原生模块(BackgroundPoller / keychain / screens)在 jest 环境缺失,
 * 导航与原生轮询器无法真实渲染, 因此这里简化为渲染 AuthProvider 空壳,
 * 并 mock 掉 keychain 与网络请求, 验证启动引导流程:
 *  1) 无会话 → 进入已登出状态, 不发网络请求;
 *  2) 会话 token 过期(me() 返回 401)→ 自动登出并暴露"登录已过期"提示。
 */
import React from 'react';
import { Text, View } from 'react-native';
import ReactTestRenderer from 'react-test-renderer';
import { AuthProvider, useAuth } from '../src/context/AuthContext';

// keychain mock: 内存存储(变量以 mock 前缀命名以通过 jest.mock 闭包检查)
let mockStoredSession: string | null = null;
jest.mock('react-native-keychain', () => ({
  setGenericPassword: jest.fn(async (_username: string, password: string) => {
    mockStoredSession = password;
    return { service: 'axmipusher_session', storage: 'mock' };
  }),
  getGenericPassword: jest.fn(async () => {
    if (mockStoredSession) {
      return {
        username: 'session',
        password: mockStoredSession,
        service: 'axmipusher_session',
        storage: 'mock',
      };
    }
    return false;
  }),
  resetGenericPassword: jest.fn(async () => {
    mockStoredSession = null;
    return true;
  }),
}));

/** 探针组件: 读取认证上下文状态用于断言。 */
function Probe() {
  const { status, expiredNotice } = useAuth();
  return (
    <View>
      <Text testID="status">{status}</Text>
      <Text testID="notice">{expiredNotice ?? ''}</Text>
    </View>
  );
}

/** 渲染 AuthProvider 空壳并等待引导完成。 */
async function renderShell() {
  let renderer!: ReactTestRenderer.ReactTestRenderer;
  await ReactTestRenderer.act(async () => {
    renderer = ReactTestRenderer.create(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    );
    // 等待异步引导(loadSession → 校验 → setStatus)完成
    await new Promise<void>(resolve => setTimeout(() => resolve(), 20));
  });
  return renderer;
}

test('无会话时引导进入已登出状态, 不发网络请求', async () => {
  const g = globalThis as { fetch?: unknown };
  const originalFetch = g.fetch;
  // 若测试环境有 fetch, 断言引导期间不会被调用
  if (typeof originalFetch === 'function') {
    g.fetch = jest.fn().mockRejectedValue(new Error('测试环境不应发起请求'));
  }
  const renderer = await renderShell();
  expect(
    renderer.root.findByProps({ testID: 'status' }).props.children,
  ).toBe('signedOut');
  if (typeof originalFetch === 'function') {
    expect(g.fetch as jest.Mock).not.toHaveBeenCalled();
    g.fetch = originalFetch;
  }
});

test('会话 token 过期时自动登出并暴露登录过期提示', async () => {
  const g = globalThis as { fetch?: unknown };
  const originalFetch = g.fetch;
  // 无 fetch 环境跳过(Node 18+ 通常自带)
  if (typeof originalFetch !== 'function') {
    return;
  }
  // 模拟 me() 返回 401(业务码 40100)
  g.fetch = jest.fn().mockResolvedValue({
    status: 401,
    json: async () => ({ code: 40100, message: '登录已过期', data: null }),
  });
  mockStoredSession = JSON.stringify({
    serverUrl: 'http://192.168.1.5:8080',
    token: 'expired-token',
    email: 'user@example.com',
    user: {
      id: 1,
      email: 'user@example.com',
      nickname: '用户',
      role: 'tenant_user',
      status: 'active',
      totp_enabled: false,
      created_at: '2026-01-01T00:00:00Z',
    },
    pollIntervalMinutes: 15,
  });
  const renderer = await renderShell();
  expect(
    renderer.root.findByProps({ testID: 'status' }).props.children,
  ).toBe('signedOut');
  expect(renderer.root.findByProps({ testID: 'notice' }).props.children).toBe(
    '登录已过期, 请重新登录',
  );
  g.fetch = originalFetch;
  mockStoredSession = null;
});

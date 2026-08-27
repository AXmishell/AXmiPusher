/**
 * 原生后台轮询 Worker 模块的类型声明(由并行代理在 android/ 侧实现)。
 *
 * 职责划分:
 * - 原生侧: 应用进入后台后按配置间隔轮询站内信, 新消息发系统通知;
 * - JS 侧: 登录/登出时调用 configure+start / stop, 每轮前台轮询后用
 *   setLastMaxId 同步已消费的最大消息 id, 防止原生重复通知。
 *
 * 说明: 此处仅做类型声明与兜底, 不实现任何原生代码。
 */
import { NativeModules } from 'react-native';

/** 原生 BackgroundPoller 模块契约(全部方法返回 Promise)。 */
export interface BackgroundPollerModule {
  /** 配置服务器地址/token/轮询间隔(分钟), 可重复调用更新配置。 */
  configure(options: {
    serverUrl: string;
    token: string;
    pollIntervalMinutes: number;
  }): Promise<void>;
  /** 启动后台轮询(应幂等, 重复调用不重复启动)。 */
  start(): Promise<void>;
  /** 停止后台轮询(登出时调用)。 */
  stop(): Promise<void>;
  /** 记录已消费的最大消息 id, 原生 Worker 跳过这些消息避免重复通知。 */
  setLastMaxId(id: number): Promise<void>;
  /** 读取最近同步的最大消息 id(JS 轮询启动时用其初始化去重基线)。 */
  getLastMaxId(): Promise<number>;
  /** 是否已配置(供设置页展示后台轮询状态)。 */
  isConfigured(): Promise<boolean>;
}

/**
 * 获取原生模块实例; 原生模块缺失(jest 环境或未安装)时返回 null。
 */
export function getNativeModule(): BackgroundPollerModule | null {
  try {
    const mod = NativeModules.BackgroundPoller as BackgroundPollerModule | undefined;
    return mod ?? null;
  } catch {
    return null;
  }
}

/** 空实现兜底: 原生模块缺失时所有调用静默成功, 避免应用崩溃。 */
const noopModule: BackgroundPollerModule = {
  async configure() {
    // 空实现
  },
  async start() {
    // 空实现
  },
  async stop() {
    // 空实现
  },
  async setLastMaxId(_id: number) {
    // 空实现
  },
  async getLastMaxId() {
    return 0;
  },
  async isConfigured() {
    return false;
  },
};

/** 业务代码统一入口: 始终返回可用模块实例(真实原生或空实现)。 */
export function getBackgroundPoller(): BackgroundPollerModule {
  return getNativeModule() ?? noopModule;
}

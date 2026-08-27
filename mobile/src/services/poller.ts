/**
 * 前台轮询服务(模块级单例状态)。
 *
 * 生命周期由 AuthContext 控制: 登录成功后 startPolling, 登出时 stopPolling。
 *
 * 要点:
 * - 前台固定 30 秒轮询一次; AppState 非 active 时暂停(后台由原生 Worker 负责);
 * - 去重核心: JS 侧维护"已见消息 id"集合(内存 Set, 启动时用原生 getLastMaxId()
 *   初始化基线), 新消息 = 本次拉取的未读列表 id 与已见集之差;
 * - 每次轮询后调用 BackgroundPoller.setLastMaxId(maxId) 同步给原生 Worker,
 *   防止原生重复发通知。
 */
import { AppState } from 'react-native';
import { listInbox, unreadCount } from '../api/inbox';
import type { InboxMessage } from '../api/types';
import { getBackgroundPoller } from '../native/BackgroundPoller';
import type { Session } from '../storage/settings';

/** 前台轮询间隔(毫秒): 固定 30 秒。 */
export const FOREGROUND_POLL_INTERVAL_MS = 30 * 1000;

/** 每轮拉取未读列表的分页大小(够覆盖一般量级)。 */
const POLL_PAGE_SIZE = 50;

/** 轮询事件订阅者(UI 注册)。 */
export interface PollerSubscriber {
  /** 发现新消息(UI 顶部插入 + 提示)。 */
  onNewMessages: (messages: InboxMessage[]) => void;
  /** 未读数变化(UI 同步未读角标)。 */
  onUnreadCountChange: (count: number) => void;
}

interface PollerState {
  session: Session;
  timer: ReturnType<typeof setInterval> | null;
  appStateSub: ReturnType<typeof AppState.addEventListener> | null;
  /** 已见最大消息 id(启动时从原生 getLastMaxId 初始化, 每轮上移)。 */
  lastMaxId: number;
  /** 本会话已上报过的消息 id 集合(防止重复上报)。 */
  seenIds: Set<number>;
  running: boolean;
  polling: boolean;
}

/** 订阅者集合(收件箱页挂载时注册)。 */
const subscribers = new Set<PollerSubscriber>();

/** 当前轮询状态; null = 未启动。 */
let state: PollerState | null = null;

/** 订阅轮询事件; 返回取消订阅函数。 */
export function subscribe(sub: PollerSubscriber): () => void {
  subscribers.add(sub);
  return () => {
    subscribers.delete(sub);
  };
}

/** 向所有订阅者广播新消息。 */
function broadcastNewMessages(messages: InboxMessage[]): void {
  for (const sub of subscribers) {
    try {
      sub.onNewMessages(messages);
    } catch {
      // 订阅方异常不影响轮询
    }
  }
}

/** 向所有订阅者广播未读数。 */
function broadcastUnreadCount(count: number): void {
  for (const sub of subscribers) {
    try {
      sub.onUnreadCountChange(count);
    } catch {
      // 订阅方异常不影响轮询
    }
  }
}

/**
 * 启动前台轮询(可重复调用, 内部先停止旧实例)。
 * 先等待原生 getLastMaxId() 建立去重基线, 再立即轮询一次并启动定时器。
 */
export async function startPolling(session: Session): Promise<void> {
  stopPolling();
  const s: PollerState = {
    session,
    timer: null,
    appStateSub: null,
    lastMaxId: 0,
    seenIds: new Set<number>(),
    running: true,
    polling: false,
  };
  state = s;
  // 用原生记录的 lastMaxId 初始化去重基线, 防止历史消息被重复上报
  try {
    const maxId = await getBackgroundPoller().getLastMaxId();
    if (state === s) {
      s.lastMaxId = Number(maxId) || 0;
    }
  } catch {
    // 原生缺失时忽略, 从 0 开始
  }
  if (!state || !s.running) {
    return;
  }
  // 立即执行一次, 快速同步未读数
  void pollOnce();
  // 定时轮询
  s.timer = setInterval(() => {
    void pollOnce();
  }, FOREGROUND_POLL_INTERVAL_MS);
  // 应用切回前台时立即补一轮, 弥补后台期间的遗漏
  s.appStateSub = AppState.addEventListener('change', next => {
    if (next === 'active') {
      void pollOnce();
    }
  });
}

/** 停止前台轮询(登出时调用)。 */
export function stopPolling(): void {
  const s = state;
  if (!s) {
    return;
  }
  if (s.timer) {
    clearInterval(s.timer);
  }
  if (s.appStateSub) {
    s.appStateSub.remove();
  }
  s.running = false;
  state = null;
}

/** 单次轮询: 前台时拉未读数, 有未读则拉列表并做去重 diff。 */
async function pollOnce(): Promise<void> {
  const s = state;
  if (!s || !s.running || s.polling) {
    return;
  }
  // 应用不在前台时暂停(后台通知由原生 Worker 负责)
  if (AppState.currentState !== 'active') {
    return;
  }
  s.polling = true;
  try {
    const count = await unreadCount(s.session.serverUrl, s.session.token);
    if (!state || !s.running) {
      return;
    }
    broadcastUnreadCount(count);
    if (count > 0) {
      const { data: list } = await listInbox(s.session.serverUrl, s.session.token, {
        current: 1,
        pageSize: POLL_PAGE_SIZE,
        read: false,
      });
      if (!state || !s.running) {
        return;
      }
      // 去重 diff: 新消息 = 列表 id 大于上次同步 maxId 且不在已见集
      const news = list.filter(m => m.id > s.lastMaxId && !s.seenIds.has(m.id));
      // 全部列表 id 记入已见集, 并上移 lastMaxId
      for (const m of list) {
        s.seenIds.add(m.id);
        if (m.id > s.lastMaxId) {
          s.lastMaxId = m.id;
        }
      }
      if (news.length > 0) {
        // 按 id 升序上报, UI 自行决定插入顺序
        broadcastNewMessages(news.sort((a, b) => a.id - b.id));
      }
    }
    // 每轮结束把最大消息 id 同步给原生 Worker, 防止原生重复通知
    if (state === s && s.running) {
      await getBackgroundPoller()
        .setLastMaxId(s.lastMaxId)
        .catch(() => {
          // 原生缺失时忽略
        });
    }
  } catch {
    // 网络/服务异常静默, 下一轮自动重试
    // (认证过期由 client 抛 AuthExpiredError, 各请求方自行处理)
  } finally {
    if (state === s) {
      s.polling = false;
    }
  }
}

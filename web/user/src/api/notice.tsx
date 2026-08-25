import { App } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';

// antd message 静态方法无法消费动态主题上下文, 通过 App 组件捕获实例。
let messageApi: MessageInstance | null = null;

/** 在根组件内挂载一次, 捕获 App.useApp() 的 message 实例。 */
export function MessageHolder() {
  const { message } = App.useApp();
  messageApi = message;
  return null;
}

/** 全局消息调用(供非组件模块如 axios 拦截器使用)。 */
export function notify(message: string) {
  if (messageApi) {
    messageApi.error(message);
  }
}

import { App } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';

let messageApi: MessageInstance | null = null;

/** 在根组件内挂载一次, 捕获 App.useApp() 的 message 实例。 */
export function MessageHolder() {
  const { message } = App.useApp();
  messageApi = message;
  return null;
}

/** 全局消息调用(供非组件模块如 axios 拦截器使用)。 */
export function notify(message: string) {
  if (messageApi) messageApi.error(message);
}

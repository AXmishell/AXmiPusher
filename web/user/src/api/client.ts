import axios from 'axios';
import { notify } from './notice';

// API 统一响应格式。
export interface ApiResp<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// 后端返回的业务对象类型。
export interface User {
  id: number;
  email: string;
  nickname: string;
  role: string;
  status: string;
  quota?: string;
  plan_id?: number;
  qq?: string;
  last_login_ip?: string;
  last_login_at?: string | null;
  created_at: string;
}

export interface ApiKey {
  id: number;
  name: string;
  key_prefix: string;
  scopes: string;
  status: string;
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
}

export interface CompatKey {
  id: number;
  external_key: string;
  source: string;
  default_channel: string;
  description: string;
  status: string;
  last_used_at: string | null;
  created_at: string;
}

export interface MessageRecord {
  message_id: number;
  request_id: string;
  channel: string;
  title: string;
  content: string;
  recipient: string;
  status: string;
  error: string;
  created_at: string;
}

export interface Callback {
  id: number;
  url: string;
  events: string;
  status: string;
  created_at: string;
}

export interface Plan {
  id: number;
  name: string;
  price: number;
  duration_days: number;
  quota: string;
  description: string;
  status: string;
}

// 创建 axios 实例。
const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('mp_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const status = error.response?.status;
    const body = error.response?.data;
    if (status === 401) {
      localStorage.removeItem('mp_token');
      if (!location.pathname.startsWith('/login')) {
        location.href = '/login';
      }
    }
    notify(body?.message || error.message || '请求失败');
    return Promise.reject(error);
  },
);

// 统一的请求辅助: 解包 {code, message, data}, code!=0 抛错。
export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  const resp = await client.request<ApiResp<T>>(config);
  const body = resp.data;
  if (body.code !== 0) {
    throw new Error(body.message || `业务错误 ${body.code}`);
  }
  return body.data;
}

export default client;

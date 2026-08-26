import axios from 'axios';
import { notify } from './notice';

export interface ApiResp<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface User {
  id: number;
  tenant_id: number;
  email: string;
  nickname: string;
  role: string;
  status: string;
}

export interface Admin {
  id: number;
  email: string;
  nickname: string;
  role: string;
  status: string;
  last_login_at?: string | null;
  created_at: string;
}

export interface TenantRow {
  id: number;
  name: string;
  status: string;
  plan_id: number;
  created_at: string;
  user_count: number;
  msg_24h: number;
}

export interface ReviewItem {
  template_id: number;
  tenant_id: number;
  tenant_name: string;
  code: string;
  name: string;
  channel_type: string;
  version_id: number;
  version: number;
  content: string;
  review_status: string;
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
  sort_order: number;
}

export interface PaymentOrder {
  id: number;
  tenant_id: number;
  plan_id: number;
  out_trade_no: string;
  epay_trade_no: string;
  amount: number;
  status: string;
  created_at: string;
}

export interface AuditLog {
  id: number;
  actor_email: string;
  action: string;
  detail: string;
  ip: string;
  created_at: string;
}

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('mp_admin_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

client.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const status = error.response?.status;
    const body = error.response?.data;
    if (status === 401) {
      localStorage.removeItem('mp_admin_token');
      // 跳转登录页(动态取当前前缀, 支持轮换 admin 路径)。
      if (!location.pathname.endsWith('/login')) {
        const seg = location.pathname.split('/').filter(Boolean)[0];
        location.href = `${seg ? '/' + seg : ''}/login`;
      }
    }
    notify(body?.message || error.message || '请求失败');
    return Promise.reject(error);
  },
);

export async function request<T>(config: Parameters<typeof client.request>[0]): Promise<T> {
  const resp = await client.request<ApiResp<T>>(config);
  const body = resp.data;
  if (body.code !== 0) throw new Error(body.message || `业务错误 ${body.code}`);
  return body.data;
}

export default client;

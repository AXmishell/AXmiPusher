import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { request, type Admin } from '../api/client';

interface LoginResp {
  token?: string;
  admin?: Admin;
  need_totp?: boolean;
  totp_token?: string;
}

// 管理后台登录: 支持两步验证(密码 → TOTP 验证码)。
export default function Login() {
  const [loading, setLoading] = useState(false);
  const [totpLoading, setTotpLoading] = useState(false);
  const [totpToken, setTotpToken] = useState<string | null>(null);
  const [email, setEmail] = useState('');
  const navigate = useNavigate();

  // 第一步: 密码登录。
  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      const d = await request<LoginResp>({ url: '/admin/auth/login', method: 'POST', data: values });
      if (d.need_totp && d.totp_token) {
        setTotpToken(d.totp_token);
        setEmail(values.email);
        return;
      }
      localStorage.setItem('mp_admin_token', d.token!);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false);
    }
  };

  // 第二步: TOTP 验证码。
  const onTotpFinish = async (values: { code: string }) => {
    if (!totpToken) return;
    setTotpLoading(true);
    try {
      const d = await request<LoginResp>({ url: '/admin/auth/login/totp', method: 'POST', data: { totp_token: totpToken, code: values.code } });
      localStorage.setItem('mp_admin_token', d.token!);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTotpLoading(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg,#1e1b4b,#312e81)' }}>
      <Card style={{ width: 400, boxShadow: '0 24px 48px rgba(0,0,0,.3)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <SafetyCertificateOutlined style={{ fontSize: 40, color: '#7c3aed' }} />
          <Typography.Title level={3} style={{ marginTop: 12, marginBottom: 4 }}>
            AXmiPusher 管理后台
          </Typography.Title>
          <Typography.Text type="secondary">
            {totpToken ? '两步验证 · 请输入验证码' : '平台管理员专属入口'}
          </Typography.Text>
        </div>

        {totpToken ? (
          <Form layout="vertical" onFinish={onTotpFinish}>
            <Form.Item name="code" label="动态验证码" rules={[{ required: true, len: 6, message: '请输入 6 位验证码' }]}>
              <Input placeholder="认证器中的 6 位数字" size="large" maxLength={6} />
            </Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={totpLoading}>
              验证并登录
            </Button>
            <div style={{ textAlign: 'center', marginTop: 12 }}>
              <a onClick={() => setTotpToken(null)}>返回密码登录</a>
            </div>
            <Typography.Text type="secondary" style={{ display: 'block', textAlign: 'center', marginTop: 8, fontSize: 12 }}>
              {email}
            </Typography.Text>
          </Form>
        ) : (
          <Form layout="vertical" onFinish={onFinish} initialValues={{ email: 'admin@messagepusher.local', password: 'admin123456' }}>
            <Form.Item name="email" label="管理员邮箱" rules={[{ required: true, message: '请输入邮箱' }]}>
              <Input placeholder="admin@example.com" size="large" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="密码" size="large" />
            </Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              登录管理后台
            </Button>
          </Form>
        )}
      </Card>
    </div>
  );
}

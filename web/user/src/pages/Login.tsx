import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SendOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { request, type User } from '../api/client';

interface LoginResp {
  token?: string;
  user?: User;
  need_totp?: boolean;
  totp_token?: string;
}

// 登录页: 支持两步验证(密码 → TOTP 验证码)。
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
      const d = await request<LoginResp>({ url: '/auth/login', method: 'POST', data: values });
      if (d.need_totp && d.totp_token) {
        setTotpToken(d.totp_token);
        setEmail(values.email);
        return;
      }
      localStorage.setItem('mp_token', d.token!);
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
      const d = await request<LoginResp>({ url: '/auth/login/totp', method: 'POST', data: { totp_token: totpToken, code: values.code } });
      localStorage.setItem('mp_token', d.token!);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTotpLoading(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg,#0f172a,#1e293b)' }}>
      <Card style={{ width: 400, boxShadow: '0 24px 48px rgba(0,0,0,.3)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <SendOutlined style={{ fontSize: 40, color: '#1e40af' }} />
          <Typography.Title level={3} style={{ marginTop: 12, marginBottom: 4 }}>
            AXmiPusher
          </Typography.Title>
          <Typography.Text type="secondary">
            {totpToken ? '两步验证 · 请输入验证码' : '消息推送平台 · 用户中心'}
          </Typography.Text>
        </div>

        {totpToken ? (
          <Form layout="vertical" onFinish={onTotpFinish}>
            <Form.Item name="code" label="动态验证码" rules={[{ required: true, len: 6, message: '请输入 6 位验证码' }]}>
              <Input placeholder="认证器中的 6 位数字" size="large" prefix={<SafetyCertificateOutlined />} maxLength={6} />
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
          <Form layout="vertical" onFinish={onFinish} initialValues={{ email: 'user1@test.com', password: 'user123456' }}>
            <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }]}>
              <Input placeholder="you@example.com" size="large" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="密码" size="large" />
            </Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              登录
            </Button>
          </Form>
        )}
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Link to="/register">还没有账号？立即注册</Link>
        </div>
      </Card>
    </div>
  );
}

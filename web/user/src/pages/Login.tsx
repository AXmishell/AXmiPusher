import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { request, type User } from '../api/client';

export default function Login() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      const d = await request<{ token: string; user: User }>({
        url: '/auth/login',
        method: 'POST',
        data: values,
      });
      localStorage.setItem('mp_token', d.token);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false);
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
          <Typography.Text type="secondary">消息推送平台 · 用户中心</Typography.Text>
        </div>
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
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Link to="/register">还没有账号？立即注册</Link>
        </div>
      </Card>
    </div>
  );
}

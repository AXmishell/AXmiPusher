import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { request } from '../api/client';

export default function Register() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const onFinish = async (values: { email: string; password: string; nickname: string; tenant_name: string }) => {
    setLoading(true);
    try {
      const d = await request<{ token: string }>({ url: '/auth/register', method: 'POST', data: values });
      localStorage.setItem('mp_token', d.token);
      message.success('注册成功，已自动登录');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg,#0f172a,#1e293b)' }}>
      <Card style={{ width: 420, boxShadow: '0 24px 48px rgba(0,0,0,.3)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <SendOutlined style={{ fontSize: 40, color: '#1e40af' }} />
          <Typography.Title level={3} style={{ marginTop: 12, marginBottom: 4 }}>
            注册账号
          </Typography.Title>
          <Typography.Text type="secondary">注册即创建独立租户空间</Typography.Text>
        </div>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
            <Input placeholder="you@example.com" size="large" />
          </Form.Item>
          <Form.Item name="nickname" label="昵称">
            <Input placeholder="你的昵称" size="large" />
          </Form.Item>
          <Form.Item name="tenant_name" label="租户名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input placeholder="例如: 我的团队" size="large" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 8, message: '密码至少 8 位' }]}>
            <Input.Password placeholder="至少 8 位" size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large" loading={loading}>
            注册
          </Button>
        </Form>
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Link to="/login">已有账号？去登录</Link>
        </div>
      </Card>
    </div>
  );
}

import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { request } from '../api/client';

export default function Register() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const onFinish = async (values: { email: string; password: string; confirm: string; nickname: string }) => {
    setLoading(true);
    try {
      const { confirm, ...payload } = values;
      const d = await request<{ token: string }>({ url: '/auth/register', method: 'POST', data: payload });
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
          <Typography.Text type="secondary">注册即创建独立空间</Typography.Text>
        </div>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
            <Input placeholder="you@example.com" size="large" />
          </Form.Item>
          <Form.Item name="nickname" label="用户名" rules={[{ max: 64, message: '最多 64 字' }]}>
            <Input placeholder="你的用户名(可空, 默认邮箱)" size="large" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 8, message: '密码至少 8 位' }]}>
            <Input.Password placeholder="至少 8 位" size="large" />
          </Form.Item>
          <Form.Item
            name="confirm"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请再次输入密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) return Promise.resolve();
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="再次输入密码" size="large" />
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

import { useState } from 'react';
import { Form, Input, Button, Card, Typography, message } from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { request, type User } from '../api/client';

export default function Login() {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      const d = await request<{ token: string; user: User }>({ url: '/auth/login', method: 'POST', data: values });
      if (d.user.role !== 'platform_admin') {
        message.error('该账号不是平台管理员');
        return;
      }
      localStorage.setItem('mp_admin_token', d.token);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg,#1e1b4b,#312e81)' }}>
      <Card style={{ width: 400, boxShadow: '0 24px 48px rgba(0,0,0,.3)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <SafetyCertificateOutlined style={{ fontSize: 40, color: '#7c3aed' }} />
          <Typography.Title level={3} style={{ marginTop: 12, marginBottom: 4 }}>
            MessagePusher 管理后台
          </Typography.Title>
          <Typography.Text type="secondary">平台管理员专属入口</Typography.Text>
        </div>
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
      </Card>
    </div>
  );
}

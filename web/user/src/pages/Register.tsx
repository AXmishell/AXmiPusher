import { useState, useEffect } from 'react';
import { Form, Input, Button, Card, Typography, message, Space } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { request } from '../api/client';

export default function Register() {
  const [loading, setLoading] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  // 验证码重发倒计时(60s), 每秒递减
  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setInterval(() => setCountdown((c) => (c > 0 ? c - 1 : 0)), 1000);
    return () => clearInterval(timer);
  }, [countdown]);

  const sendCode = async () => {
    const email = form.getFieldValue('email');
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      message.warning('请先填写有效邮箱');
      return;
    }
    try {
      await request({ url: '/auth/register/send-code', method: 'POST', data: { email } });
      message.success('验证码已发送到邮箱');
      setCountdown(60);
    } catch {
      /* 拦截器已提示, 不重复弹 */
    }
  };

  const onFinish = async (values: { email: string; code: string; password: string; confirm: string; nickname: string }) => {
    setLoading(true);
    try {
      const d = await request<{ token: string }>({
        url: '/auth/register',
        method: 'POST',
        data: {
          email: values.email,
          code: values.code,
          password: values.password,
          confirm_password: values.confirm,
          nickname: values.nickname,
        },
      });
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
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
            <Input placeholder="you@example.com" size="large" />
          </Form.Item>
          <Form.Item name="code" label="邮箱验证码" rules={[{ required: true, message: '请输入邮箱验证码' }]}>
            <Space.Compact style={{ width: '100%' }}>
              <Input placeholder="6 位验证码" size="large" />
              <Button size="large" disabled={countdown > 0} onClick={sendCode}>
                {countdown > 0 ? `重新发送(${countdown}s)` : '获取验证码'}
              </Button>
            </Space.Compact>
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
        <div style={{ textAlign: 'center', marginTop: 8 }}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>验证码 5 分钟内有效</Typography.Text>
        </div>
      </Card>
    </div>
  );
}

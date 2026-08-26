import { useEffect, useState } from 'react';
import { Card, Descriptions, Form, Input, Button, message } from 'antd';
import { UserOutlined, MailOutlined, LockOutlined, IdcardOutlined } from '@ant-design/icons';
import { request, type User } from '../api/client';
import ChangePasswordModal from '../components/ChangePasswordModal';

// 账户设置页: 账户信息展示 + 用户名/昵称/QQ + 邮箱 + 密码修改。
export default function Profile() {
  const [user, setUser] = useState<User | null>(null);
  const [profileForm] = Form.useForm();
  const [emailForm] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);
  const [pwdOpen, setPwdOpen] = useState(false);

  useEffect(() => {
    request<{ user: User }>({ url: '/auth/me', method: 'GET' })
      .then((d) => {
        setUser(d.user);
        profileForm.setFieldsValue({ nickname: d.user.nickname, qq: d.user.qq });
        emailForm.setFieldsValue({ email: d.user.email });
      })
      .catch(() => {});
  }, [profileForm, emailForm]);

  // 保存用户名/QQ。
  const saveProfile = async (values: { nickname: string; qq?: string }) => {
    setSaving(true);
    try {
      const d = await request<{ user: User }>({ url: '/auth/profile', method: 'PUT', data: values });
      setUser(d.user);
      message.success('账户资料已更新');
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false);
    }
  };

  // 修改登录邮箱。
  const saveEmail = async (values: { email: string }) => {
    setSavingEmail(true);
    try {
      const d = await request<{ user: User }>({ url: '/auth/email', method: 'PUT', data: values });
      setUser(d.user);
      emailForm.setFieldsValue({ email: d.user.email });
      message.success('邮箱已修改, 下次登录请使用新邮箱');
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSavingEmail(false);
    }
  };

  return (
    <div>
      <Card title="账户信息" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="middle">
          <Descriptions.Item label="账号 ID">{user?.id ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="用户名">{user?.nickname || '-'}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="注册时间">
            {user?.created_at ? new Date(user.created_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="最近登录 IP">{user?.last_login_ip || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="账户资料" style={{ marginBottom: 16 }}>
        <Form form={profileForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={saveProfile}>
          <Form.Item name="nickname" label="用户名" rules={[{ required: true, message: '请输入用户名' }, { max: 64, message: '最多 64 字' }]}>
            <Input placeholder="用户名" prefix={<UserOutlined />} />
          </Form.Item>
          <Form.Item name="qq" label="QQ 号码" rules={[{ max: 32, message: '最多 32 位' }]}>
            <Input placeholder="QQ 号码(可空)" prefix={<IdcardOutlined />} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>
            保存资料
          </Button>
        </Form>
      </Card>

      <Card title="邮箱修改" style={{ marginBottom: 16 }}>
        <Form form={emailForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={saveEmail}>
          <Form.Item name="email" label="登录邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
            <Input placeholder="新邮箱" prefix={<MailOutlined />} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={savingEmail}>
            修改邮箱
          </Button>
          <div style={{ color: 'rgba(0,0,0,.45)', fontSize: 12, marginTop: 8 }}>
            修改后旧邮箱立即失效, 下次登录请使用新邮箱。
          </div>
        </Form>
      </Card>

      <Card title="密码修改">
        <Button icon={<LockOutlined />} onClick={() => setPwdOpen(true)}>
          修改密码
        </Button>
      </Card>

      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </div>
  );
}
